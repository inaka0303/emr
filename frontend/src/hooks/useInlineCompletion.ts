import { useState, useRef, useCallback, useEffect } from 'react';

const BASE_URL = '/api';

interface AutocompleteResponse {
  data: {
    completion: string;
    full_text: string;
  };
  meta: {
    model: string;
    is_mock: boolean;
    latency_ms: number;
  };
}

interface UseInlineCompletionOptions {
  context: string;
  debounceMs?: number;
  patientId?: number;
  enabled?: boolean;
  experimentAttemptId?: string | null;
  /** 問診全文（suggest LoRA訓練分布に合わせてSLMに渡す） */
  interviewText?: string;
  /** 既記載セクション（キー "S"/"O"/"A" など、現セクションより前のもの） */
  priorSections?: Record<string, string>;
}

interface UseInlineCompletionReturn {
  completion: string;
  isLoading: boolean;
  accept: () => string;
  dismiss: () => void;
  clear: () => void;
}

export function useInlineCompletion(
  text: string,
  options: UseInlineCompletionOptions,
): UseInlineCompletionReturn {
  const {
    context,
    debounceMs = 300,
    patientId,
    enabled = true,
    experimentAttemptId,
    interviewText,
    priorSections,
  } = options;

  const [completion, setCompletion] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const fullTextRef = useRef('');
  const abortControllerRef = useRef<AbortController | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // テキスト変更のたびにデバウンス後にリクエストを送る
  useEffect(() => {
    // 前のデバウンスタイマーをクリア
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // AI無効時、またはテキストが空/短い場合はリクエストしない
    if (!enabled || text.length < 2) {
      setCompletion('');
      fullTextRef.current = '';
      setIsLoading(false);
      return;
    }

    debounceTimerRef.current = setTimeout(() => {
      // 前のリクエストをキャンセル
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const controller = new AbortController();
      abortControllerRef.current = controller;

      setIsLoading(true);

      fetch(`${BASE_URL}/slm/autocomplete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(experimentAttemptId ? { 'X-Experiment-Attempt': experimentAttemptId } : {}),
        },
        body: JSON.stringify({
          text,
          context,
          ...(patientId != null ? { patient_id: patientId } : {}),
          ...(interviewText ? { interview_text: interviewText } : {}),
          ...(priorSections && Object.keys(priorSections).length > 0
            ? { prior_sections: priorSections }
            : {}),
        }),
        signal: controller.signal,
      })
        .then((res) => {
          if (!res.ok) throw new Error('Autocomplete request failed');
          return res.json() as Promise<AutocompleteResponse>;
        })
        .then((data) => {
          if (!controller.signal.aborted) {
            setCompletion(data.data.completion);
            fullTextRef.current = data.data.full_text;
            setIsLoading(false);
          }
        })
        .catch((err: unknown) => {
          if (err instanceof DOMException && err.name === 'AbortError') return;
          if (!controller.signal.aborted) {
            setCompletion('');
            fullTextRef.current = '';
            setIsLoading(false);
          }
        });
    }, debounceMs);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
    // priorSections は参照が毎回変わりうるのでJSON化して依存を安定させる
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [text, context, patientId, debounceMs, enabled, experimentAttemptId, interviewText, JSON.stringify(priorSections ?? {})]);

  // クリーンアップ
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  const accept = useCallback((): string => {
    const result = fullTextRef.current || text;
    setCompletion('');
    fullTextRef.current = '';
    return result;
  }, [text]);

  const dismiss = useCallback(() => {
    setCompletion('');
    fullTextRef.current = '';
  }, []);

  const clear = useCallback(() => {
    setCompletion('');
    fullTextRef.current = '';
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
  }, []);

  return { completion, isLoading, accept, dismiss, clear };
}
