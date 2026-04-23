import { useState } from 'react';
import { post } from '../../api/client';

interface RAGResult {
  parent_id: string;
  text: string;
  title: string;
  publication_year?: number | null;
  score: number;
  child_hits: number;
}

interface RAGResponse {
  data: {
    query: string;
    results: RAGResult[];
    elapsed_ms: number;
  };
}

interface RAGEvidencePanelProps {
  query: string;
  label?: string;
}

/**
 * RAGでガイドラインから根拠を検索・表示するパネル。
 * 「根拠を確認」ボタン押下でRAG検索を起動（~10秒）し、
 * ヒットしたガイドライン名と引用文を表示する。
 */
export default function RAGEvidencePanel({ query, label = '根拠を確認' }: RAGEvidencePanelProps) {
  const [results, setResults] = useState<RAGResult[] | null>(null);
  const [elapsed, setElapsed] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());

  const search = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const resp = await post<RAGResponse>('/rag/search', { query, n: 5 });
      setResults(resp.data.results);
      setElapsed(resp.data.elapsed_ms);
    } catch (err) {
      setError(err instanceof Error ? err.message : '検索失敗');
    } finally {
      setIsLoading(false);
    }
  };

  const toggleExpand = (idx: number) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(idx)) next.delete(idx);
      else next.add(idx);
      return next;
    });
  };

  return (
    <div className="mt-2 border-t border-dashed border-gray-200 pt-2">
      {results === null && !isLoading && (
        <button
          onClick={search}
          disabled={!query.trim()}
          className="text-xs text-teal-700 hover:text-teal-900 disabled:text-gray-400 underline underline-offset-2"
        >
          📖 {label}（RAGでガイドライン検索）
        </button>
      )}

      {isLoading && (
        <div className="flex items-center gap-2 text-xs text-gray-500">
          <div className="w-3 h-3 border-2 border-teal-400 border-t-transparent rounded-full animate-spin" />
          ガイドライン検索中... (~10秒)
        </div>
      )}

      {error && (
        <div className="text-xs text-red-600 bg-red-50 border border-red-200 rounded p-2">
          {error}
        </div>
      )}

      {results !== null && results.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-xs font-semibold text-gray-600">
              💡 根拠（関連ガイドライン {results.length} 件）
              {elapsed != null && <span className="text-gray-400 ml-2">({(elapsed / 1000).toFixed(1)}秒)</span>}
            </p>
            <button
              onClick={search}
              className="text-xs text-teal-700 hover:underline"
            >
              🔄 再検索
            </button>
          </div>
          <div className="space-y-1.5">
            {results.map((r, i) => {
              const isExp = expanded.has(i);
              const preview = r.text.substring(0, 150);
              return (
                <div key={i} className="bg-teal-50/60 border border-teal-200 rounded px-2.5 py-1.5">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <div className="text-xs font-medium text-teal-900 truncate">
                        {i + 1}. {r.title || '（タイトル不明）'}
                        {r.publication_year && (
                          <span className="ml-1 text-teal-600 font-normal">（{r.publication_year}年）</span>
                        )}
                      </div>
                      <div className="text-xs text-gray-500 mt-0.5">
                        score={r.score.toFixed(2)} / hits={r.child_hits}
                      </div>
                    </div>
                  </div>
                  <div className="mt-1 text-xs text-gray-700 leading-relaxed whitespace-pre-wrap">
                    {isExp ? r.text : preview}
                    {r.text.length > 150 && (
                      <button
                        onClick={() => toggleExpand(i)}
                        className="ml-1 text-teal-600 hover:underline"
                      >
                        {isExp ? '閉じる' : `...[全${r.text.length}文字を見る]`}
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {results !== null && results.length === 0 && (
        <div className="text-xs text-gray-500">該当するガイドラインは見つかりませんでした。</div>
      )}
    </div>
  );
}
