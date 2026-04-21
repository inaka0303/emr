import { useState, useEffect, useRef } from 'react';
import { suggestAdmissionSummary } from '../../api/slm';
import { get, post } from '../../api/client';

interface AdmissionSummaryDrafterProps {
  encounterId: number;
  interviewText: string;
}

type Status = 'loading-cache' | 'generating' | 'draft' | 'accepted' | 'manual' | 'error';

interface SavedSummary {
  encounter_id: number;
  content: string;
  author: string;
  is_slm_draft: boolean;
}

/**
 * 入院時サマリ drafter + editor。
 *  - 既に保存済みのサマリがあれば読み込んで編集モード
 *  - 未保存の場合は admission LoRA で自動ドラフト生成 → 採用/却下
 *  - 採用後は編集可能な textarea + 保存ボタン
 *
 * 備考: admission LoRA は学習データ30件で品質不十分。参考表示用。
 */
export default function AdmissionSummaryDrafter({
  encounterId,
  interviewText,
}: AdmissionSummaryDrafterProps) {
  const [draftText, setDraftText] = useState('');
  const [finalText, setFinalText] = useState('');
  const [status, setStatus] = useState<Status>('loading-cache');
  const [meta, setMeta] = useState<{ latency_ms?: number; is_mock?: boolean } | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');

  const fetchedRef = useRef<number | null>(null);

  useEffect(() => {
    setDraftText('');
    setFinalText('');
    setMeta(null);
    setErrorMsg(null);
    setSaveState('idle');
    fetchedRef.current = null;
    setStatus('loading-cache');

    // まず既存保存を確認
    get<{ data: SavedSummary | null }>(`/encounters/${encounterId}/admission-summary`)
      .then((res) => {
        if (res.data && res.data.content) {
          setFinalText(res.data.content);
          setStatus(res.data.is_slm_draft ? 'accepted' : 'manual');
          return;
        }
        // 未保存 → SLM生成
        if (!interviewText) {
          setStatus('manual');
          return;
        }
        if (fetchedRef.current === encounterId) return;
        fetchedRef.current = encounterId;
        setStatus('generating');
        return suggestAdmissionSummary(interviewText)
          .then((sres) => {
            setDraftText(sres.data.text ?? '');
            setMeta({ latency_ms: sres.meta.latency_ms, is_mock: sres.meta.is_mock });
            setStatus(sres.data.text ? 'draft' : 'manual');
          })
          .catch((err: unknown) => {
            setErrorMsg(err instanceof Error ? err.message : 'サマリ生成に失敗しました');
            setStatus('error');
          });
      })
      .catch(() => {
        // 取得失敗時は新規扱い
        if (!interviewText) {
          setStatus('manual');
          return;
        }
        setStatus('generating');
        suggestAdmissionSummary(interviewText)
          .then((sres) => {
            setDraftText(sres.data.text ?? '');
            setMeta({ latency_ms: sres.meta.latency_ms, is_mock: sres.meta.is_mock });
            setStatus(sres.data.text ? 'draft' : 'manual');
          })
          .catch((err: unknown) => {
            setErrorMsg(err instanceof Error ? err.message : 'サマリ生成に失敗しました');
            setStatus('error');
          });
      });
  }, [encounterId, interviewText]);

  const accept = () => {
    setFinalText(draftText);
    setStatus('accepted');
  };
  const reject = () => {
    setFinalText('');
    setDraftText('');
    setStatus('manual');
  };

  const save = async () => {
    setSaveState('saving');
    try {
      await post(`/encounters/${encounterId}/admission-summary`, {
        content: finalText,
        author: '',
        is_slm_draft: status === 'accepted',
      });
      setSaveState('saved');
      setTimeout(() => setSaveState('idle'), 3000);
    } catch (err) {
      console.warn('入院時サマリ保存エラー', err);
      setSaveState('error');
    }
  };

  return (
    <section className="bg-white rounded-lg border border-gray-200 shadow-sm">
      <header className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <div>
          <h3 className="text-base font-semibold text-gray-800">入院時サマリ</h3>
          <p className="text-xs text-gray-500 mt-0.5">
            admission LoRA によるドラフト
            {status === 'generating' && <span className="ml-2 text-violet-500">生成中...</span>}
            {status === 'draft' && meta?.latency_ms != null && (
              <span className="ml-2 text-amber-600">{(meta.latency_ms / 1000).toFixed(1)}秒で生成完了</span>
            )}
          </p>
        </div>
        {status === 'draft' && (
          <div className="flex items-center gap-1.5">
            <button
              onClick={accept}
              className="px-3 py-1 text-xs bg-emerald-500 hover:bg-emerald-600 text-white rounded"
            >
              採用
            </button>
            <button
              onClick={reject}
              className="px-3 py-1 text-xs bg-white hover:bg-gray-100 text-gray-600 border border-gray-300 rounded"
            >
              却下
            </button>
          </div>
        )}
      </header>

      <div className="p-4 space-y-3">
        {status === 'loading-cache' && (
          <div className="text-sm text-gray-400">読み込み中...</div>
        )}

        {status === 'generating' && (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="animate-pulse bg-gray-200 rounded h-3 w-full" />
            ))}
          </div>
        )}

        {status === 'error' && (
          <div className="p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">
            {errorMsg}
          </div>
        )}

        {status === 'draft' && (
          <div className="bg-amber-50/40 border border-dashed border-amber-300 rounded px-3 py-2">
            <pre className="text-sm text-gray-500 italic whitespace-pre-wrap leading-relaxed font-sans">
              {draftText}
            </pre>
          </div>
        )}

        {(status === 'accepted' || status === 'manual') && (
          <>
            <textarea
              value={finalText}
              onChange={(e) => setFinalText(e.target.value)}
              placeholder={status === 'manual' ? '入院時サマリを記載...' : ''}
              rows={14}
              className="w-full text-sm leading-relaxed border border-gray-200 rounded p-2 font-sans focus:outline-none focus:ring-2 focus:ring-primary-400"
            />
            <div className="flex items-center justify-between">
              <div className="text-xs">
                {saveState === 'saved' && <span className="text-emerald-600">保存しました ✓</span>}
                {saveState === 'error' && <span className="text-red-600">保存失敗</span>}
              </div>
              <button
                onClick={save}
                disabled={saveState === 'saving' || finalText.trim() === ''}
                className="px-4 py-1.5 bg-primary-600 hover:bg-primary-700 disabled:bg-gray-300 disabled:cursor-not-allowed text-white text-sm font-medium rounded shadow-sm"
              >
                {saveState === 'saving' ? '保存中...' : '入院時サマリを保存'}
              </button>
            </div>
          </>
        )}
      </div>
    </section>
  );
}
