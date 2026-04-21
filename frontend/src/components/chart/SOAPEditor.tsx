import { useState, useCallback, useEffect } from 'react';
import { post } from '../../api/client';
import InlineCompletionTextarea from '../slm/InlineCompletionTextarea';
import RAGEvidencePanel from '../slm/RAGEvidencePanel';
import type { SoapDraftEntry } from '../../hooks/useSoapDraftCache';

interface SOAPData {
  subjective: string;
  objective: string;
  assessment: string;
  plan: string;
}

type SectionKey = keyof SOAPData;
type SectionStatus = 'idle' | 'generating' | 'draft' | 'accepted' | 'manual';

interface SOAPEditorProps {
  patientName: string;
  encounterId?: number;
  patientId?: number;
  interviewText: string;
  /** SOAP記録が既にある場合、自動ドラフトはスキップする */
  hasExistingSOAP?: boolean;
  /** 親でキャッシュ済みのドラフト状態（useSoapDraftCache から） */
  draftEntry?: SoapDraftEntry | null;
}

interface SectionConfig {
  key: SectionKey;
  label: string;
  letter: string;
  description: string;
  context: string;
  borderColor: string;
  labelColor: string;
}

const SECTIONS: SectionConfig[] = [
  { key: 'subjective', label: '主観的所見', letter: 'S', description: '患者の訴え、自覚症状',
    context: 'soap_subjective', borderColor: 'border-blue-400', labelColor: 'text-blue-700 bg-blue-50' },
  { key: 'objective', label: '客観的所見', letter: 'O', description: '検査結果、バイタル、身体所見',
    context: 'soap_objective', borderColor: 'border-emerald-400', labelColor: 'text-emerald-700 bg-emerald-50' },
  { key: 'assessment', label: '評価', letter: 'A', description: '診断、アセスメント',
    context: 'soap_assessment', borderColor: 'border-amber-400', labelColor: 'text-amber-700 bg-amber-50' },
  { key: 'plan', label: '計画', letter: 'P', description: '治療計画、処方、次回予約',
    context: 'soap_plan', borderColor: 'border-purple-400', labelColor: 'text-purple-700 bg-purple-50' },
];

type StatusMap = Record<SectionKey, SectionStatus>;

const initialStatus: StatusMap = {
  subjective: 'idle',
  objective: 'idle',
  assessment: 'idle',
  plan: 'idle',
};

export default function SOAPEditor({
  patientName,
  encounterId,
  patientId,
  interviewText,
  hasExistingSOAP = false,
  draftEntry = null,
}: SOAPEditorProps) {
  const [data, setData] = useState<SOAPData>({ subjective: '', objective: '', assessment: '', plan: '' });
  const [draftText, setDraftText] = useState<SOAPData>({ subjective: '', objective: '', assessment: '', plan: '' });
  const [statuses, setStatuses] = useState<StatusMap>(initialStatus);
  const [draftError, setDraftError] = useState<string | null>(null);
  const [draftMeta, setDraftMeta] = useState<{ latency_ms?: number; is_mock?: boolean } | null>(null);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');

  // encounter変更時、前回の状態をリセット
  useEffect(() => {
    setData({ subjective: '', objective: '', assessment: '', plan: '' });
    setDraftText({ subjective: '', objective: '', assessment: '', plan: '' });
    setStatuses(initialStatus);
    setDraftError(null);
    setDraftMeta(null);
    setSaveStatus('idle');
    setErrorMessage('');
  }, [encounterId]);

  // 親から渡される draftEntry を反映（SSE で逐次到着するセクションに追従）
  //
  // 方針: accepted / manual 状態のセクションは「ユーザーの編集物」なのでストリーム更新で上書きしない。
  // generating / idle / draft はまだ SLM 管轄なので、到着したテキストで更新する。
  useEffect(() => {
    if (encounterId == null) return;
    if (hasExistingSOAP) return;
    if (!draftEntry) return;
    if (draftEntry.encounterId !== encounterId) return;

    if (draftEntry.error) {
      setDraftError(draftEntry.error);
      setStatuses((prev) => ({
        subjective: prev.subjective === 'accepted' || prev.subjective === 'manual' ? prev.subjective : 'manual',
        objective: prev.objective === 'accepted' || prev.objective === 'manual' ? prev.objective : 'manual',
        assessment: prev.assessment === 'accepted' || prev.assessment === 'manual' ? prev.assessment : 'manual',
        plan: prev.plan === 'accepted' || prev.plan === 'manual' ? prev.plan : 'manual',
      }));
      return;
    }

    const s = draftEntry.suggestion ?? { subjective: '', objective: '', assessment: '', plan: '' };

    setDraftText((prev) => ({
      subjective: s.subjective || prev.subjective,
      objective: s.objective || prev.objective,
      assessment: s.assessment || prev.assessment,
      plan: s.plan || prev.plan,
    }));

    // 各セクションの状態を更新
    // - accepted / manual: ユーザー操作済みなので触らない
    // - それ以外: テキストがあれば 'draft'、なければ 'generating'（完了後は 'manual'）
    const isDone = draftEntry.done;
    setStatuses((prev) => {
      const keys: SectionKey[] = ['subjective', 'objective', 'assessment', 'plan'];
      const next = { ...prev };
      for (const key of keys) {
        if (prev[key] === 'accepted' || prev[key] === 'manual') continue;
        const text = s[key];
        if (text) {
          next[key] = 'draft';
        } else if (isDone) {
          next[key] = 'manual';
        } else {
          next[key] = 'generating';
        }
      }
      return next;
    });

    if (draftEntry.meta) {
      setDraftMeta(draftEntry.meta);
    }
  }, [encounterId, hasExistingSOAP, draftEntry]);

  const handleChange = useCallback((key: SectionKey, value: string) => {
    setData((prev) => ({ ...prev, [key]: value }));
  }, []);

  // 全ドラフトを一括採用
  const acceptAll = useCallback(() => {
    setData((prev) => ({
      subjective: statuses.subjective === 'draft' ? draftText.subjective : prev.subjective,
      objective: statuses.objective === 'draft' ? draftText.objective : prev.objective,
      assessment: statuses.assessment === 'draft' ? draftText.assessment : prev.assessment,
      plan: statuses.plan === 'draft' ? draftText.plan : prev.plan,
    }));
    setStatuses((prev) => ({
      subjective: prev.subjective === 'draft' ? 'accepted' : prev.subjective,
      objective: prev.objective === 'draft' ? 'accepted' : prev.objective,
      assessment: prev.assessment === 'draft' ? 'accepted' : prev.assessment,
      plan: prev.plan === 'draft' ? 'accepted' : prev.plan,
    }));
  }, [draftText, statuses]);

  const acceptSection = useCallback((key: SectionKey) => {
    setData((prev) => ({ ...prev, [key]: draftText[key] }));
    setStatuses((prev) => ({ ...prev, [key]: 'accepted' }));
  }, [draftText]);

  const rejectSection = useCallback((key: SectionKey) => {
    setData((prev) => ({ ...prev, [key]: '' }));
    setDraftText((prev) => ({ ...prev, [key]: '' }));
    setStatuses((prev) => ({ ...prev, [key]: 'manual' }));
  }, []);

  const handleSave = useCallback(async () => {
    if (!encounterId) {
      setSaveStatus('error');
      setErrorMessage('受診を選択してください');
      return;
    }
    setSaveStatus('saving');
    setErrorMessage('');
    try {
      await post(`/encounters/${encounterId}/soap`, data);
      setSaveStatus('success');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } catch (err) {
      setSaveStatus('error');
      setErrorMessage(err instanceof Error ? err.message : '保存に失敗しました');
    }
  }, [data, encounterId]);

  const hasAnyDraft = (Object.keys(statuses) as SectionKey[]).some((k) => statuses[k] === 'draft');
  const isGenerating = (Object.keys(statuses) as SectionKey[]).some((k) => statuses[k] === 'generating');

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
      <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <div>
          <h3 className="text-base font-semibold text-gray-800">新規カルテ記録</h3>
          <p className="text-xs text-gray-500 mt-0.5">
            {patientName} - SOAP形式
            {isGenerating && <span className="ml-2 text-violet-500">SLMがドラフト生成中...</span>}
            {!isGenerating && hasAnyDraft && (
              <span className="ml-2 text-amber-600">
                グレーのドラフトを確認してください
                {draftMeta?.latency_ms ? ` (${(draftMeta.latency_ms / 1000).toFixed(1)}秒)` : ''}
              </span>
            )}
            {!isGenerating && !hasAnyDraft && (
              <span className="ml-2 text-violet-500">Tab で補完を適用</span>
            )}
          </p>
        </div>
        {hasAnyDraft && (
          <button
            onClick={acceptAll}
            className="px-3 py-1 bg-emerald-50 hover:bg-emerald-100 text-emerald-700 text-xs font-medium rounded border border-emerald-200"
          >
            全ドラフトを採用
          </button>
        )}
      </div>

      {draftError && (
        <div className="mx-4 mt-3 p-2 bg-red-50 border border-red-200 rounded text-xs text-red-700">
          ドラフト生成エラー: {draftError}（各セクションを自分で記載してください）
        </div>
      )}

      <div className="p-4 space-y-4">
        {SECTIONS.map((section) => {
          const status = statuses[section.key];
          return (
            <div key={section.key} className={`border-l-4 ${section.borderColor} pl-3`}>
              <div className="flex items-center gap-2 mb-1.5">
                <span className={`inline-flex items-center justify-center w-6 h-6 rounded text-xs font-bold ${section.labelColor}`}>
                  {section.letter}
                </span>
                <span className="text-sm font-medium text-gray-700">{section.label}</span>
                <span className="text-xs text-gray-400">{section.description}</span>

                {status === 'draft' && (
                  <div className="ml-auto flex items-center gap-1.5">
                    <span className="text-xs text-amber-600 font-medium">SLMドラフト</span>
                    <button
                      onClick={() => acceptSection(section.key)}
                      className="px-2 py-0.5 text-xs bg-emerald-500 hover:bg-emerald-600 text-white rounded"
                      title="このドラフトを採用"
                    >
                      採用
                    </button>
                    <button
                      onClick={() => rejectSection(section.key)}
                      className="px-2 py-0.5 text-xs bg-white hover:bg-gray-100 text-gray-600 border border-gray-300 rounded"
                      title="却下して自分で記載"
                    >
                      却下
                    </button>
                  </div>
                )}
                {status === 'accepted' && (
                  <span className="ml-auto text-xs text-emerald-600">✓ 採用済み（編集可）</span>
                )}
              </div>

              {status === 'generating' ? (
                <DraftSkeleton />
              ) : status === 'draft' ? (
                <DraftPreview text={draftText[section.key]} />
              ) : (
                <>
                  <InlineCompletionTextarea
                    value={data[section.key]}
                    onChange={(v) => handleChange(section.key, v)}
                    context={section.context}
                    patientId={patientId}
                    interviewText={interviewText}
                    priorSections={buildPriorSectionsFor(section.key, data)}
                    placeholder={`${section.label}を入力...`}
                    rows={3}
                  />
                  {/* A/P セクションでは RAG で根拠を確認できる */}
                  {(section.key === 'assessment' || section.key === 'plan') &&
                    data[section.key] && data[section.key].trim().length > 5 && (
                      <RAGEvidencePanel
                        query={`${section.label}: ${data[section.key]}`}
                        label={`${section.letter}記載の根拠を確認`}
                      />
                    )}
                </>
              )}
            </div>
          );
        })}
      </div>

      <div className="px-4 py-3 border-t border-gray-100 flex items-center justify-between">
        <div className="text-sm">
          {saveStatus === 'success' && <span className="text-emerald-600">保存しました</span>}
          {saveStatus === 'error' && <span className="text-red-600">{errorMessage}</span>}
        </div>
        <button
          onClick={handleSave}
          disabled={saveStatus === 'saving' || hasAnyDraft}
          title={hasAnyDraft ? 'ドラフトを採用または却下してから保存してください' : ''}
          className="px-5 py-2 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {saveStatus === 'saving' ? '保存中...' : '保存'}
        </button>
      </div>
    </div>
  );
}

// 現セクションより前のセクション (S/O/A) を抽出してSLMに渡す
// 例: 現セクション='assessment' → S/O を priorSections として返す
function buildPriorSectionsFor(currentKey: SectionKey, data: SOAPData): Record<string, string> {
  const order: { key: SectionKey; letter: string }[] = [
    { key: 'subjective', letter: 'S' },
    { key: 'objective', letter: 'O' },
    { key: 'assessment', letter: 'A' },
    { key: 'plan', letter: 'P' },
  ];
  const out: Record<string, string> = {};
  for (const { key, letter } of order) {
    if (key === currentKey) break;
    if (data[key] && data[key].trim() !== '') {
      out[letter] = data[key];
    }
  }
  return out;
}

function DraftSkeleton() {
  return (
    <div className="space-y-2 py-2">
      <div className="animate-pulse bg-gray-200 rounded h-3 w-full" />
      <div className="animate-pulse bg-gray-200 rounded h-3 w-5/6" />
      <div className="animate-pulse bg-gray-200 rounded h-3 w-2/3" />
    </div>
  );
}

function DraftPreview({ text }: { text: string }) {
  return (
    <div className="bg-amber-50/40 border border-dashed border-amber-300 rounded px-3 py-2">
      <p className="text-sm text-gray-500 italic whitespace-pre-wrap leading-relaxed">
        {text || '（ドラフトなし）'}
      </p>
    </div>
  );
}
