import { useState, useEffect, useRef, useCallback } from 'react';
import { post } from '../../api/client';
import type { InterviewNote } from '../../hooks/useEncounterInterview';

interface InterviewViewerProps {
  encounterId: number | null;
  notes: InterviewNote[];
  isLoading: boolean;
  error: string | null;
  /** 問診/お薬/所見/検査 のいずれかが保存された時に呼ばれる */
  onInterviewUpdated?: (encounterId: number) => void;
  /** 医師が「問診を確定してSOAP生成」を押した時に呼ばれる */
  onFinalize?: (encounterId: number) => void;
  /** 既に確定済みか */
  finalized?: boolean;
  /** Control条件ではSOAPの裏生成をしない */
  aiEnabled?: boolean;
  /** 実験モードでは症例情報を固定し、被験者による誤編集を防ぐ */
  readOnly?: boolean;
}

const DEBOUNCE_MS = 1000;

type Section = {
  key: 'raw_text' | 'medication_list' | 'exam_findings' | 'lab_results';
  label: string;
  icon: string;
  placeholder: string;
  minRows: number;
};

// 4セクション設計 (医学ワークフローと整合、情報源で分離):
//   raw_text       : 患者から聞く内容 (問診)
//   medication_list: 持参薬情報 (お薬手帳)
//   exam_findings  : 医師の視触聴診 (診察所見)
//   lab_results    : システムから来る客観データ (検査)
const SECTIONS: Section[] = [
  {
    key: 'raw_text',
    label: '問診記録',
    icon: '📋',
    placeholder:
      '患者から聞き取った内容を記載:\n・主訴・現病歴\n・既往歴・アレルギー\n・家族歴・社会歴（喫煙・飲酒・職業など）\n※薬の詳細・バイタル・所見は他セクションに分けて入力',
    minRows: 6,
  },
  {
    key: 'medication_list',
    label: 'お薬手帳',
    icon: '💊',
    placeholder:
      '持参薬・お薬手帳の内容:\n・エナラプリル 5mg 1日1回朝\n・カルベジロール 10mg 1日2回\n・（自己中断中などのメモも）',
    minRows: 3,
  },
  {
    key: 'exam_findings',
    label: '診察所見',
    icon: '🩺',
    placeholder:
      '医師が視触聴診で取る客観的所見:\n腹部: 軟、臍周圧痛あり、反跳痛なし\n胸部: 呼吸音清、心音整\n四肢: 浮腫なし',
    minRows: 3,
  },
  {
    key: 'lab_results',
    label: '検査結果',
    icon: '🧪',
    placeholder:
      'バイタル・採血・画像などの客観データ:\nBP 140/97, HR 50, RR 13, SpO2 98%\nWBC 11200, CRP 5.8, Cr 0.9\n腹部エコー: 胆嚢腫大なし',
    minRows: 3,
  },
];

type Values = Record<Section['key'], string>;
const EMPTY_VALUES: Values = { raw_text: '', medication_list: '', exam_findings: '', lab_results: '' };

/**
 * 問診記録ペイン（4セクション分離 + 縦スタック）。
 *
 * どれか1セクションに文字があれば「入力モード」として保存 + SOAP生成を促す。
 * 初期表示時に既存notesがあればそれをロードし、空でないセクションは閲覧的に、
 * 空セクションは入力可能にする（常時編集可能）。
 */
export default function InterviewViewer({
  encounterId,
  notes,
  isLoading,
  error,
  onInterviewUpdated,
  onFinalize,
  finalized = false,
  aiEnabled = true,
  readOnly = false,
}: InterviewViewerProps) {
  const [values, setValues] = useState<Values>(EMPTY_VALUES);
  const [saveState, setSaveState] = useState<'idle' | 'debouncing' | 'saving' | 'saved' | 'error'>('idle');
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const initializedForRef = useRef<number | null>(null);

  // encounter切替時 or notes受信時に初期値ロード
  useEffect(() => {
    if (encounterId == null) {
      setValues(EMPTY_VALUES);
      setSaveState('idle');
      initializedForRef.current = null;
      return;
    }
    if (initializedForRef.current === encounterId) return;
    if (isLoading) return;
    initializedForRef.current = encounterId;
    if (notes.length > 0) {
      const n = notes[0];
      setValues({
        raw_text: n.raw_text ?? '',
        medication_list: n.medication_list ?? '',
        exam_findings: n.exam_findings ?? '',
        lab_results: n.lab_results ?? '',
      });
      setSaveState('idle');
    } else {
      setValues(EMPTY_VALUES);
      setSaveState('idle');
    }
  }, [encounterId, notes, isLoading]);

  const handleFieldChange = useCallback((key: Section['key'], v: string) => {
    setValues((prev) => ({ ...prev, [key]: v }));
  }, []);

  // debounce で保存（いずれかのフィールド変更で 1秒後に全フィールドまとめて POST）
  const isUserTyping = Object.values(values).some((v) => v.trim() !== '');
  useEffect(() => {
    if (readOnly) return;
    if (encounterId == null) return;
    if (!isUserTyping) {
      setSaveState('idle');
      return;
    }
    // 既存notesと値が同じなら保存しない
    if (notes.length > 0) {
      const n = notes[0];
      const same =
        (n.raw_text ?? '') === values.raw_text &&
        (n.medication_list ?? '') === values.medication_list &&
        (n.exam_findings ?? '') === values.exam_findings &&
        (n.lab_results ?? '') === values.lab_results;
      if (same) {
        setSaveState('saved');
        return;
      }
    }

    setSaveState('debouncing');
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(async () => {
      setSaveState('saving');
      try {
        await post(`/encounters/${encounterId}/interviews`, {
          raw_text: values.raw_text,
          medication_list: values.medication_list,
          exam_findings: values.exam_findings,
          lab_results: values.lab_results,
        });
        setSaveState('saved');
        onInterviewUpdated?.(encounterId);
      } catch (err) {
        console.warn('問診保存エラー', err);
        setSaveState('error');
      }
    }, DEBOUNCE_MS);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [readOnly, values, encounterId]);

  const hasAnyContent = Object.values(values).some((v) => v.trim() !== '');

  return (
    <section className="bg-white rounded-lg border border-gray-200 shadow-sm flex flex-col">
      <header className="px-4 py-3 border-b border-gray-100 flex items-center gap-2">
        <svg className="w-4 h-4 text-teal-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
            d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
        </svg>
        <h3 className="text-base font-semibold text-gray-800">診療情報</h3>
        <span className="text-xs text-gray-400">4セクション構造</span>
        <span className="ml-auto text-xs">
          {saveState === 'debouncing' && <span className="text-gray-400">入力待ち…</span>}
          {saveState === 'saving' && <span className="text-violet-500">保存中…</span>}
          {saveState === 'saved' && <span className="text-emerald-600">保存済み ✓</span>}
          {saveState === 'error' && <span className="text-red-600">保存失敗</span>}
        </span>
      </header>

      <div className="p-3 space-y-3">
        {isLoading && <div className="text-sm text-gray-400">読み込み中…</div>}
        {error && (
          <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded p-2">{error}</div>
        )}

        {!isLoading && !error &&
          SECTIONS.map((sec) => (
            <SectionField
              key={sec.key}
              section={sec}
              value={values[sec.key]}
              onChange={(v) => handleFieldChange(sec.key, v)}
              readOnly={readOnly}
            />
          ))}

        {!isLoading && !error && (
          <div className="flex items-center justify-between pt-1 px-1">
            <p className="text-xs text-gray-500">
              {readOnly ? '症例情報は固定されています' : aiEnabled ? '1秒停止ごとに保存' : '1秒停止ごとに保存（AI補助なし）'}
            </p>
            {!finalized && encounterId != null && hasAnyContent && (
              <button
                onClick={() => onFinalize?.(encounterId)}
                disabled={saveState === 'debouncing' || saveState === 'saving'}
                className="px-4 py-1.5 bg-primary-600 hover:bg-primary-700 disabled:bg-gray-300 disabled:cursor-not-allowed text-white text-sm font-medium rounded shadow-sm whitespace-nowrap"
                title="保存された情報でSOAPドラフトを表示"
              >
                情報を確定してSOAP表示
              </button>
            )}
            {finalized && (
              <span className="text-xs text-emerald-600">
                確定済み ✓{readOnly ? '' : '（編集継続可）'}
              </span>
            )}
          </div>
        )}
      </div>
    </section>
  );
}

// ============= 単一セクション =============
function SectionField({
  section,
  value,
  onChange,
  readOnly = false,
}: {
  section: Section;
  value: string;
  onChange: (v: string) => void;
  readOnly?: boolean;
}) {
  const [collapsed, setCollapsed] = useState(false);
  const trimmed = value.trim();
  const filled = trimmed.length > 0;

  return (
    <div className={`border rounded-md ${filled ? 'border-teal-300' : 'border-gray-200'}`}>
      <div
        className={`flex items-center gap-2 px-2.5 py-1.5 cursor-pointer select-none ${
          filled ? 'bg-teal-50/60' : 'bg-gray-50'
        }`}
        onClick={() => setCollapsed((c) => !c)}
      >
        <span className="text-base">{section.icon}</span>
        <span className="text-sm font-medium text-gray-800">{section.label}</span>
        {filled && (
          <span className="text-xs text-emerald-600">入力済（{trimmed.length}文字）</span>
        )}
        {!filled && <span className="text-xs text-gray-400">未入力</span>}
        <button
          className="ml-auto text-xs text-gray-500 hover:text-gray-700"
          onClick={(e) => {
            e.stopPropagation();
            setCollapsed((c) => !c);
          }}
        >
          {collapsed ? '展開' : '折りたたむ'}
        </button>
      </div>
      {!collapsed && (
        <textarea
          value={value}
          onChange={(e) => {
            if (!readOnly) onChange(e.target.value);
          }}
          readOnly={readOnly}
          placeholder={section.placeholder}
          rows={section.minRows}
          className={`w-full text-sm leading-relaxed p-2.5 focus:outline-none font-sans resize-y ${
            readOnly
              ? 'bg-white text-gray-800 cursor-default'
              : 'focus:ring-2 focus:ring-teal-300'
          }`}
        />
      )}
    </div>
  );
}
