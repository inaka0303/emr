import { useRef, useState, useCallback, useEffect } from 'react';
import type { SLMSoapSuggestion, SLMMeta } from '../types/api';

export interface SoapDraftEntry {
  encounterId: number;
  isLoading: boolean;
  /** 現在までに受信したセクション（セクション単位でUIを順次更新する） */
  suggestion: SLMSoapSuggestion | null;
  meta: SLMMeta | null;
  error: string | null;
  /** 完了フラグ（全セクション + done イベント受信済み） */
  done: boolean;
  /** このエントリを駆動しているストリームの世代番号。古い世代のコールバックは無視する */
  generation: number;
}

const EMPTY_SOAP: SLMSoapSuggestion = {
  subjective: '',
  objective: '',
  assessment: '',
  plan: '',
};

const SECTION_KEY_BY_LETTER: Record<string, keyof SLMSoapSuggestion> = {
  S: 'subjective',
  O: 'objective',
  A: 'assessment',
  P: 'plan',
};

/**
 * encounterId をキーにSOAPドラフトをキャッシュし、SSEでセクション逐次に反映する。
 *
 * 設計ポイント:
 *  - 完了したPromiseは inflightRef から削除しない（親の再レンダー時の再発火を防ぐ）
 *  - force=true で再プリフェッチされた場合、古いstreamを AbortController でキャンセル
 *    （問診編集中に debounce 毎に prefetch が走るとき、古いstreamが残ると
 *     遅れて到着したonSectionで新しい世代のエントリが上書きされ、P だけが残る等のバグが起きる）
 *  - 世代(generation)番号を使って、古いstreamのコールバックを無視する二重防御
 */
export function useSoapDraftCache() {
  const inflightRef = useRef<Map<number, Promise<void>>>(new Map());
  const abortRef = useRef<Map<number, AbortController>>(new Map());
  const entriesRef = useRef<Map<number, SoapDraftEntry>>(new Map());
  const [entries, setEntries] = useState<Map<number, SoapDraftEntry>>(new Map());

  useEffect(() => {
    entriesRef.current = entries;
  }, [entries]);

  const updateEntry = useCallback(
    (encounterId: number, generation: number, update: (prev: SoapDraftEntry) => SoapDraftEntry) => {
      setEntries((prev) => {
        const curr = prev.get(encounterId);
        if (!curr) return prev;
        // 古い世代のコールバックは無視（古いstreamが遅れて来た更新を捨てる）
        if (curr.generation !== generation) return prev;
        const next = new Map(prev);
        next.set(encounterId, update(curr));
        return next;
      });
    },
    [],
  );

  const prefetch = useCallback((encounterId: number, force = false) => {
    if (!encounterId) return;

    if (!force) {
      if (inflightRef.current.has(encounterId)) return;
      const existing = entriesRef.current.get(encounterId);
      if (existing && (existing.done || existing.error)) return;
    } else {
      // force=true: 進行中のSSEをキャンセル（コールバック抑止）
      const prevAbort = abortRef.current.get(encounterId);
      if (prevAbort) {
        prevAbort.abort();
      }
      // inflight は残しておくが、世代が変わるので旧streamの更新は無視される
    }

    // 新しい世代番号を決定（既存 +1）
    const prevEntry = entriesRef.current.get(encounterId);
    const nextGeneration = (prevEntry?.generation ?? 0) + 1;

    // 初期エントリ（最新世代として書き込む）
    setEntries((prev) => {
      const next = new Map(prev);
      next.set(encounterId, {
        encounterId,
        isLoading: true,
        suggestion: { ...EMPTY_SOAP },
        meta: null,
        error: null,
        done: false,
        generation: nextGeneration,
      });
      return next;
    });

    const controller = new AbortController();
    abortRef.current.set(encounterId, controller);

    const p = streamSoapDraft(encounterId, force, controller.signal, {
      onSection: (letter, text) => {
        const key = SECTION_KEY_BY_LETTER[letter];
        if (!key) return;
        updateEntry(encounterId, nextGeneration, (prev) => ({
          ...prev,
          suggestion: { ...(prev.suggestion ?? EMPTY_SOAP), [key]: text },
        }));
      },
      onDone: (meta) => {
        updateEntry(encounterId, nextGeneration, (prev) => ({
          ...prev,
          isLoading: false,
          meta,
          done: true,
        }));
      },
      onError: (message) => {
        updateEntry(encounterId, nextGeneration, (prev) => ({
          ...prev,
          isLoading: false,
          error: message,
        }));
      },
    }).catch((err: unknown) => {
      if (err instanceof DOMException && err.name === 'AbortError') return;
      updateEntry(encounterId, nextGeneration, (prev) => ({
        ...prev,
        isLoading: false,
        error: err instanceof Error ? err.message : 'ドラフト生成に失敗しました',
      }));
    });

    inflightRef.current.set(encounterId, p);
  }, [updateEntry]);

  const get = useCallback(
    (encounterId: number | null): SoapDraftEntry | null => {
      if (encounterId == null) return null;
      return entries.get(encounterId) ?? null;
    },
    [entries],
  );

  const invalidate = useCallback((encounterId: number) => {
    const prevAbort = abortRef.current.get(encounterId);
    if (prevAbort) prevAbort.abort();
    abortRef.current.delete(encounterId);
    setEntries((prev) => {
      const next = new Map(prev);
      next.delete(encounterId);
      return next;
    });
    inflightRef.current.delete(encounterId);
  }, []);

  return { prefetch, get, invalidate };
}

// ===================== 内部: SSE クライアント =====================

interface StreamHandlers {
  onSection: (letter: string, text: string) => void;
  onDone: (meta: SLMMeta) => void;
  onError: (message: string) => void;
}

/** POST /api/encounters/:id/soap-draft/stream を SSE で読み取る（AbortSignalで中断可） */
async function streamSoapDraft(
  encounterId: number,
  force: boolean,
  signal: AbortSignal,
  handlers: StreamHandlers,
): Promise<void> {
  const resp = await fetch(`/api/encounters/${encounterId}/soap-draft/stream`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ force }),
    signal,
  });
  if (!resp.ok || !resp.body) {
    if (!signal.aborted) handlers.onError(`SSE接続失敗: ${resp.status}`);
    return;
  }

  const reader = resp.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  // 中断された reader を確実にクリーンアップ
  const abortHandler = () => {
    reader.cancel().catch(() => {});
  };
  signal.addEventListener('abort', abortHandler);

  try {
    while (true) {
      if (signal.aborted) return;
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });

      let idx: number;
      while ((idx = buffer.indexOf('\n\n')) >= 0) {
        const block = buffer.slice(0, idx);
        buffer = buffer.slice(idx + 2);
        if (signal.aborted) return;
        const { event, data } = parseSSEBlock(block);
        if (!event) continue;
        if (event === 'section') {
          try {
            const obj = JSON.parse(data) as { section: string; text: string };
            handlers.onSection(obj.section, obj.text ?? '');
          } catch { /* skip */ }
        } else if (event === 'done') {
          try {
            const obj = JSON.parse(data) as SLMMeta;
            handlers.onDone(obj);
          } catch {
            handlers.onDone({ model: '', is_mock: false });
          }
        } else if (event === 'error') {
          try {
            const obj = JSON.parse(data) as { message?: string };
            handlers.onError(obj.message ?? 'ストリーミング中にエラーが発生しました');
          } catch {
            handlers.onError('ストリーミング中にエラーが発生しました');
          }
        }
      }
    }
  } finally {
    signal.removeEventListener('abort', abortHandler);
  }
}

function parseSSEBlock(block: string): { event: string | null; data: string } {
  let event: string | null = null;
  let data = '';
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim();
    } else if (line.startsWith('data:')) {
      data += (data ? '\n' : '') + line.slice(5).trim();
    }
  }
  return { event, data };
}
