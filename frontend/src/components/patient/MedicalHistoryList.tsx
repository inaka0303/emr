import type { MedicalHistory } from '../../types/api';

interface MedicalHistoryListProps {
  histories: MedicalHistory[];
}

const STATUS_STYLES: Record<string, string> = {
  '治療中': 'bg-amber-50 text-amber-700 border-amber-200',
  '寛解': 'bg-emerald-50 text-emerald-700 border-emerald-200',
  '完治': 'bg-gray-50 text-gray-500 border-gray-200',
};

export default function MedicalHistoryList({ histories }: MedicalHistoryListProps) {
  if (histories.length === 0) {
    return (
      <p className="text-sm text-gray-400 py-4 text-center">既往歴の記録はありません</p>
    );
  }

  return (
    <div className="space-y-3">
      {histories.map((h) => (
        <div
          key={h.id}
          className="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-sm transition-shadow"
        >
          <div className="flex items-start justify-between">
            <div>
              <h4 className="text-sm font-semibold text-gray-800">{h.condition}</h4>
              <p className="text-xs text-gray-500 mt-0.5">発症: {h.onset_date}</p>
            </div>
            <span
              className={`text-xs px-2 py-0.5 rounded-full border font-medium ${
                STATUS_STYLES[h.status] ?? 'bg-gray-50 text-gray-600 border-gray-200'
              }`}
            >
              {h.status}
            </span>
          </div>
          {h.notes && (
            <p className="text-sm text-gray-600 mt-2 leading-relaxed">{h.notes}</p>
          )}
        </div>
      ))}
    </div>
  );
}
