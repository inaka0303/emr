import type { SLMSummarySuggestionItem } from '../../types/api';
import type { SummaryState } from '../../hooks/useSLMSuggestion';

interface SummarySuggestProps {
  summaryState: SummaryState;
  onAcceptItem: (index: number) => void;
  onDismissItem: (index: number) => void;
  onAcceptAll: () => void;
}

function ConfidenceBar({ confidence }: { confidence: number }) {
  const percent = Math.round(confidence * 100);
  let barColor = 'bg-red-400';
  let textColor = 'text-red-600';
  if (percent >= 90) {
    barColor = 'bg-emerald-400';
    textColor = 'text-emerald-600';
  } else if (percent >= 70) {
    barColor = 'bg-amber-400';
    textColor = 'text-amber-600';
  }

  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-1.5 bg-gray-100 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${barColor}`}
          style={{ width: `${percent}%` }}
        />
      </div>
      <span className={`text-xs font-medium ${textColor} min-w-[2.5rem] text-right`}>
        {percent}%
      </span>
    </div>
  );
}

function SkeletonCard() {
  return (
    <div className="p-3 border border-gray-200 rounded-lg animate-pulse">
      <div className="flex items-center gap-2 mb-2">
        <div className="h-4 w-16 bg-gray-200 rounded" />
        <div className="h-4 w-24 bg-gray-200 rounded" />
      </div>
      <div className="h-1.5 bg-gray-200 rounded-full w-3/4" />
    </div>
  );
}

export default function SummarySuggest({
  summaryState,
  onAcceptItem,
  onDismissItem,
  onAcceptAll,
}: SummarySuggestProps) {
  const { suggestions, acceptedItems, dismissedItems, isLoading, meta, error, category } =
    summaryState;

  const categoryLabel =
    category === 'family_history' ? '家族歴' : category === 'social_history' ? '社会歴' : '要約';

  // 表示対象（dismiss済みは除外）
  const visibleSuggestions = suggestions
    .map((item, index) => ({ item, index }))
    .filter(({ index }) => !dismissedItems.has(index));

  const pendingCount = visibleSuggestions.filter(
    ({ index }) => !acceptedItems.has(index),
  ).length;

  const acceptedList = suggestions
    .map((item, index) => ({ item, index }))
    .filter(({ index }) => acceptedItems.has(index));

  if (!isLoading && suggestions.length === 0 && !error) {
    return null;
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
      <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-gray-800">
            {categoryLabel}抽出結果
          </h3>
          {meta && (
            <p className="text-xs text-gray-400 mt-0.5">
              {meta.is_mock ? 'モック' : 'SLM'} |{' '}
              {meta.model}
              {meta.latency_ms ? ` | ${meta.latency_ms}ms` : ''}
            </p>
          )}
        </div>
        {pendingCount > 0 && (
          <button
            onClick={onAcceptAll}
            className="px-3 py-1.5 text-xs font-medium text-emerald-700 bg-emerald-50
              border border-emerald-200 rounded-lg hover:bg-emerald-100 transition-colors"
          >
            全て採用 ({pendingCount})
          </button>
        )}
      </div>

      <div className="p-4 space-y-3">
        {/* ローディング */}
        {isLoading && (
          <>
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
          </>
        )}

        {/* エラー */}
        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        {/* 提案リスト */}
        {visibleSuggestions.map(({ item, index }) => {
          const isAccepted = acceptedItems.has(index);
          return (
            <SuggestionCard
              key={index}
              item={item}
              isAccepted={isAccepted}
              onAccept={() => onAcceptItem(index)}
              onDismiss={() => onDismissItem(index)}
            />
          );
        })}

        {/* 全てdismissした場合 */}
        {!isLoading &&
          suggestions.length > 0 &&
          visibleSuggestions.length === 0 && (
            <p className="text-sm text-gray-400 text-center py-4">
              すべての提案が処理されました
            </p>
          )}

        {/* 採用済みリスト */}
        {acceptedList.length > 0 && (
          <div className="mt-4 pt-3 border-t border-gray-100">
            <h4 className="text-xs font-medium text-gray-500 mb-2">
              採用済み ({acceptedList.length})
            </h4>
            <div className="space-y-2">
              {acceptedList.map(({ item, index }) => (
                <div
                  key={index}
                  className="flex items-center gap-2 px-3 py-2 bg-emerald-50 rounded-lg text-sm"
                >
                  <svg
                    className="w-4 h-4 text-emerald-500 flex-shrink-0"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                  <span className="text-gray-600">{item.field}:</span>
                  <span className="text-gray-800 font-medium">{item.value}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

interface SuggestionCardProps {
  item: SLMSummarySuggestionItem;
  isAccepted: boolean;
  onAccept: () => void;
  onDismiss: () => void;
}

function SuggestionCard({
  item,
  isAccepted,
  onAccept,
  onDismiss,
}: SuggestionCardProps) {
  return (
    <div
      className={`p-3 border rounded-lg transition-all duration-300 ${
        isAccepted
          ? 'border-emerald-200 bg-emerald-50/50'
          : 'border-gray-200 hover:border-gray-300 hover:shadow-sm'
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xs font-medium text-gray-500 bg-gray-100 px-2 py-0.5 rounded">
              {item.field}
            </span>
          </div>
          <p className="text-sm font-medium text-gray-800 break-words">
            {item.value}
          </p>
          <div className="mt-2">
            <ConfidenceBar confidence={item.confidence} />
          </div>
        </div>

        {!isAccepted && (
          <div className="flex items-center gap-1.5 flex-shrink-0">
            <button
              onClick={onAccept}
              className="px-2.5 py-1.5 text-xs font-medium text-emerald-700
                bg-emerald-50 border border-emerald-200 rounded-md
                hover:bg-emerald-100 transition-colors"
            >
              採用
            </button>
            <button
              onClick={onDismiss}
              className="px-2.5 py-1.5 text-xs font-medium text-gray-500
                bg-gray-50 border border-gray-200 rounded-md
                hover:bg-gray-100 transition-colors"
            >
              却下
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
