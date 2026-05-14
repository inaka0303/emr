import { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import { post, put } from '../../api/client';
import InlineCompletionTextarea from '../slm/InlineCompletionTextarea';
import RAGEvidencePanel from '../slm/RAGEvidencePanel';
import type { SoapDraftEntry } from '../../hooks/useSoapDraftCache';
import type { ApiResponse, SOAPNote } from '../../types/api';

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
  aiEnabled?: boolean;
  experimentAttemptId?: string | null;
  draftStorageVersion?: string | null;
  onExperimentEvent?: (eventType: string, payload?: Record<string, unknown>) => void;
  onSaved?: () => void;
  /** SOAP記録が既にある場合、自動ドラフトはスキップする */
  hasExistingSOAP?: boolean;
  existingSOAPNote?: SOAPNote | null;
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

const emptySOAPData: SOAPData = { subjective: '', objective: '', assessment: '', plan: '' };

function hasSOAPContent(data: SOAPData): boolean {
  return Object.values(data).some((v) => v.trim() !== '');
}

function fromSOAPNote(note: SOAPNote): SOAPData {
  return {
    subjective: note.subjective ?? '',
    objective: note.objective ?? '',
    assessment: note.assessment ?? '',
    plan: note.plan ?? '',
  };
}

function parseSavedAt(note: SOAPNote | null | undefined): Date | null {
  const value = note?.updated_at || note?.created_at;
  if (!value) return null;
  const normalized = value.includes('T') ? value : value.replace(' ', 'T');
  const ms = Date.parse(normalized);
  return Number.isNaN(ms) ? null : new Date(ms);
}

function formatSavedTime(date: Date): string {
  return date.toLocaleTimeString('ja-JP', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function soapDraftStorageKey(
  encounterId?: number,
  experimentAttemptId?: string | null,
  draftStorageVersion?: string | null,
): string | null {
  if (encounterId == null) return null;
  const attemptKey = experimentAttemptId ?? 'general';
  const versionKey = draftStorageVersion ?? 'current';
  return `emr:soap-editor-draft:${attemptKey}:${versionKey}:${encounterId}`;
}

function readStoredSOAPDraft(key: string): SOAPData | null {
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<SOAPData>;
    return {
      subjective: typeof parsed.subjective === 'string' ? parsed.subjective : '',
      objective: typeof parsed.objective === 'string' ? parsed.objective : '',
      assessment: typeof parsed.assessment === 'string' ? parsed.assessment : '',
      plan: typeof parsed.plan === 'string' ? parsed.plan : '',
    };
  } catch {
    return null;
  }
}

function writeStoredSOAPDraft(key: string, data: SOAPData) {
  try {
    if (!hasSOAPContent(data)) {
      window.localStorage.removeItem(key);
      return;
    }
    window.localStorage.setItem(key, JSON.stringify(data));
  } catch {
    /* localStorage unavailable: ignore and keep normal in-memory behavior */
  }
}

function removeStoredSOAPDraft(key: string | null) {
  if (!key) return;
  try {
    window.localStorage.removeItem(key);
  } catch {
    /* ignore */
  }
}

export default function SOAPEditor({
  patientName,
  encounterId,
  patientId,
  interviewText,
  aiEnabled = true,
  experimentAttemptId = null,
  draftStorageVersion = null,
  onExperimentEvent,
  onSaved,
  hasExistingSOAP = false,
  existingSOAPNote = null,
  draftEntry = null,
}: SOAPEditorProps) {
  const [data, setData] = useState<SOAPData>(emptySOAPData);
  const [draftText, setDraftText] = useState<SOAPData>(emptySOAPData);
  const [statuses, setStatuses] = useState<StatusMap>(initialStatus);
  const [draftError, setDraftError] = useState<string | null>(null);
  const [draftMeta, setDraftMeta] = useState<{ latency_ms?: number; is_mock?: boolean } | null>(null);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'success' | 'error'>('idle');
  const [savedNoteId, setSavedNoteId] = useState<number | null>(null);
  const [lastSavedAt, setLastSavedAt] = useState<Date | null>(null);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [editedDraftSections, setEditedDraftSections] = useState<Set<SectionKey>>(new Set());
  const draftStorageKey = useMemo(
    () => soapDraftStorageKey(encounterId, experimentAttemptId, draftStorageVersion),
    [encounterId, experimentAttemptId, draftStorageVersion],
  );
  const skipNextPersistRef = useRef(false);

  // encounter変更時、前回の状態をリセット
  useEffect(() => {
    skipNextPersistRef.current = true;
    const stored = draftStorageKey ? readStoredSOAPDraft(draftStorageKey) : null;
    const existingData = existingSOAPNote ? fromSOAPNote(existingSOAPNote) : null;
    setData(stored ?? existingData ?? emptySOAPData);
    setDraftText(emptySOAPData);
    setStatuses(initialStatus);
    setDraftError(null);
    setDraftMeta(null);
    setSavedNoteId(existingSOAPNote?.id ?? null);
    setLastSavedAt(parseSavedAt(existingSOAPNote));
    setHasUnsavedChanges(stored != null);
    setSaveStatus(existingSOAPNote && stored == null ? 'success' : 'idle');
    setErrorMessage('');
    setEditedDraftSections(new Set());
  }, [draftStorageKey, existingSOAPNote?.id, existingSOAPNote?.updated_at]);

  // 保存ボタン前の入力をブラウザに退避し、誤リロードでも復元できるようにする。
  useEffect(() => {
    if (!draftStorageKey) return;
    if (skipNextPersistRef.current) {
      skipNextPersistRef.current = false;
      return;
    }
    writeStoredSOAPDraft(draftStorageKey, data);
  }, [data, draftStorageKey]);

  // 親から渡される draftEntry を反映（SSE で逐次到着するセクションに追従）
  //
  // 方針: accepted / manual 状態のセクションは「ユーザーの編集物」なのでストリーム更新で上書きしない。
  // generating / idle / draft はまだ SLM 管轄なので、到着したテキストで更新する。
  useEffect(() => {
    if (encounterId == null) return;
    if (!aiEnabled) return;
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
  }, [encounterId, aiEnabled, hasExistingSOAP, draftEntry]);

  const markUnsaved = useCallback(() => {
    setHasUnsavedChanges(true);
    setSaveStatus((prev) => (prev === 'success' ? 'idle' : prev));
  }, []);

  const handleChange = useCallback((key: SectionKey, value: string) => {
    if (aiEnabled && statuses[key] === 'accepted' && !editedDraftSections.has(key)) {
      setEditedDraftSections((prev) => {
        const next = new Set(prev);
        next.add(key);
        return next;
      });
      onExperimentEvent?.('ai_draft_edited', { section: key });
    }
    setData((prev) => ({ ...prev, [key]: value }));
    markUnsaved();
  }, [aiEnabled, editedDraftSections, markUnsaved, onExperimentEvent, statuses]);

  // 全ドラフトを一括採用
  const acceptAll = useCallback(() => {
    const acceptedSections = (Object.keys(statuses) as SectionKey[]).filter((k) => statuses[k] === 'draft');
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
    if (acceptedSections.length > 0) {
      markUnsaved();
      onExperimentEvent?.('ai_draft_accepted', { section: 'all', count: acceptedSections.length });
    }
  }, [draftText, markUnsaved, onExperimentEvent, statuses]);

  const acceptSection = useCallback((key: SectionKey) => {
    setData((prev) => ({ ...prev, [key]: draftText[key] }));
    setStatuses((prev) => ({ ...prev, [key]: 'accepted' }));
    markUnsaved();
    onExperimentEvent?.('ai_draft_accepted', { section: key, count: 1 });
  }, [draftText, markUnsaved, onExperimentEvent]);

  const rejectSection = useCallback((key: SectionKey) => {
    setData((prev) => ({ ...prev, [key]: '' }));
    setDraftText((prev) => ({ ...prev, [key]: '' }));
    setStatuses((prev) => ({ ...prev, [key]: 'manual' }));
    markUnsaved();
    onExperimentEvent?.('ai_draft_rejected', { section: key, count: 1 });
  }, [markUnsaved, onExperimentEvent]);

  const handleSave = useCallback(async () => {
    if (!encounterId) {
      setSaveStatus('error');
      setErrorMessage('受診を選択してください');
      return;
    }
    setSaveStatus('saving');
    setErrorMessage('');
    try {
      if (savedNoteId != null) {
        await put(`/soap/${savedNoteId}`, data);
        setLastSavedAt(new Date());
      } else {
        const res = await post<ApiResponse<SOAPNote>>(`/encounters/${encounterId}/soap`, data);
        setSavedNoteId(res.data.id);
        setLastSavedAt(parseSavedAt(res.data) ?? new Date());
      }
      onExperimentEvent?.('soap_saved', {
        subjective_len: data.subjective.length,
        objective_len: data.objective.length,
        assessment_len: data.assessment.length,
        plan_len: data.plan.length,
      });
      removeStoredSOAPDraft(draftStorageKey);
      setHasUnsavedChanges(false);
      setSaveStatus('success');
      onSaved?.();
    } catch (err) {
      setSaveStatus('error');
      setErrorMessage(err instanceof Error ? err.message : '保存に失敗しました');
    }
  }, [data, draftStorageKey, encounterId, onExperimentEvent, onSaved, savedNoteId]);

  const hasAnyDraft = (Object.keys(statuses) as SectionKey[]).some((k) => statuses[k] === 'draft');
  const isGenerating = (Object.keys(statuses) as SectionKey[]).some((k) => statuses[k] === 'generating');
  const canSave = hasSOAPContent(data) && !hasAnyDraft && saveStatus !== 'saving' && (savedNoteId == null || hasUnsavedChanges);
  const saveButtonLabel =
    saveStatus === 'saving'
      ? '保存中...'
      : savedNoteId != null && !hasUnsavedChanges
        ? '保存済み'
        : savedNoteId != null
          ? '変更を保存'
          : '保存';

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
            {!aiEnabled && (
              <span className="ml-2 text-gray-500">Control条件: AI補助なし</span>
            )}
            {aiEnabled && !isGenerating && !hasAnyDraft && (
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
                    encounterId={encounterId}
                    enabled={aiEnabled}
                    experimentAttemptId={experimentAttemptId}
                    interviewText={interviewText}
                    priorSections={buildPriorSectionsFor(section.key, data)}
                    placeholder={`${section.label}を入力...`}
                    rows={3}
                  />
                  {/* A/P セクションでは RAG で根拠を確認できる */}
                  {aiEnabled && (section.key === 'assessment' || section.key === 'plan') &&
                    data[section.key] && data[section.key].trim().length > 5 && (
                      <RAGEvidencePanel
                        query={`${section.label}: ${data[section.key]}`}
                        label={`${section.letter}記載の根拠を確認`}
                        experimentAttemptId={experimentAttemptId}
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
          {saveStatus === 'saving' && <span className="text-violet-600">保存中...</span>}
          {lastSavedAt && !hasUnsavedChanges && saveStatus !== 'saving' && saveStatus !== 'error' && (
            <span className="text-emerald-700 font-medium">
              保存済み ✓ <span className="font-normal text-emerald-600">{formatSavedTime(lastSavedAt)}</span>
            </span>
          )}
          {lastSavedAt && hasUnsavedChanges && saveStatus !== 'saving' && saveStatus !== 'error' && (
            <span className="text-amber-700">保存後に未保存の変更があります</span>
          )}
          {!lastSavedAt && hasUnsavedChanges && saveStatus !== 'saving' && saveStatus !== 'error' && (
            <span className="text-amber-700">未保存の記載があります</span>
          )}
          {saveStatus === 'error' && <span className="text-red-600">{errorMessage}</span>}
        </div>
        <button
          onClick={handleSave}
          disabled={!canSave}
          title={hasAnyDraft ? 'ドラフトを採用または却下してから保存してください' : ''}
          className="px-5 py-2 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {saveButtonLabel}
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
