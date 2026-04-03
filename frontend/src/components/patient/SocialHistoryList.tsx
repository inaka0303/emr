import type { SocialHistory } from '../../types/api';

interface SocialHistoryListProps {
  histories: SocialHistory[];
}

const CATEGORY_STYLES: Record<string, { icon: string; color: string }> = {
  '喫煙': { icon: '🚬', color: 'bg-red-50 text-red-700 border-red-200' },
  '飲酒': { icon: '🍺', color: 'bg-amber-50 text-amber-700 border-amber-200' },
  '職業': { icon: '💼', color: 'bg-blue-50 text-blue-700 border-blue-200' },
  '運動': { icon: '🏃', color: 'bg-emerald-50 text-emerald-700 border-emerald-200' },
};

export default function SocialHistoryList({ histories }: SocialHistoryListProps) {
  if (histories.length === 0) {
    return (
      <p className="text-sm text-gray-400 py-4 text-center">社会歴の記録はありません</p>
    );
  }

  return (
    <div className="space-y-3">
      {histories.map((h) => {
        const style = CATEGORY_STYLES[h.category] ?? { icon: '📋', color: 'bg-gray-50 text-gray-700 border-gray-200' };
        return (
          <div
            key={h.id}
            className="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-sm transition-shadow"
          >
            <div className="flex items-start gap-3">
              <span className="text-lg flex-shrink-0" role="img" aria-label={h.category}>
                {style.icon}
              </span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full border font-medium ${style.color}`}
                  >
                    {h.category}
                  </span>
                  {h.is_slm_suggested && (
                    <span className="text-xs px-2 py-0.5 rounded-full bg-violet-50 text-violet-700 font-medium">
                      SLM提案
                    </span>
                  )}
                </div>
                <p className="text-sm text-gray-800 mt-1.5">{h.description}</p>
                {h.notes && (
                  <p className="text-xs text-gray-500 mt-1">{h.notes}</p>
                )}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
