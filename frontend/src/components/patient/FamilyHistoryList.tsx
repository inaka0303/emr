import type { FamilyHistory } from '../../types/api';

interface FamilyHistoryListProps {
  histories: FamilyHistory[];
}

export default function FamilyHistoryList({ histories }: FamilyHistoryListProps) {
  if (histories.length === 0) {
    return (
      <p className="text-sm text-gray-400 py-4 text-center">家族歴の記録はありません</p>
    );
  }

  return (
    <div className="space-y-3">
      {histories.map((h) => (
        <div
          key={h.id}
          className="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-sm transition-shadow"
        >
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center justify-center w-8 h-8 rounded-full bg-indigo-50 text-indigo-700 text-xs font-bold flex-shrink-0">
              {h.relation}
            </span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h4 className="text-sm font-semibold text-gray-800">{h.condition}</h4>
                {h.is_slm_suggested && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-violet-50 text-violet-700 font-medium flex-shrink-0">
                    SLM提案
                  </span>
                )}
              </div>
              {h.notes && (
                <p className="text-xs text-gray-500 mt-0.5">{h.notes}</p>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
