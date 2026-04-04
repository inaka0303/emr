import { useState, useCallback } from 'react';
import type { SocialHistory } from '../../types/api';
import InlineCompletionTextarea from '../slm/InlineCompletionTextarea';

interface SocialHistoryListProps {
  histories: SocialHistory[];
  patientId?: number;
}

const CATEGORY_STYLES: Record<string, { icon: string; color: string }> = {
  '喫煙': { icon: '🚬', color: 'bg-red-50 text-red-700 border-red-200' },
  '飲酒': { icon: '🍺', color: 'bg-amber-50 text-amber-700 border-amber-200' },
  '職業': { icon: '💼', color: 'bg-blue-50 text-blue-700 border-blue-200' },
  '運動': { icon: '🏃', color: 'bg-emerald-50 text-emerald-700 border-emerald-200' },
};

export default function SocialHistoryList({ histories, patientId }: SocialHistoryListProps) {
  const [isAdding, setIsAdding] = useState(false);
  const [inputText, setInputText] = useState('');
  const [localHistories, setLocalHistories] = useState<SocialHistory[]>([]);

  const handleSave = useCallback(() => {
    if (!inputText.trim()) return;

    const newEntry: SocialHistory = {
      id: Date.now(),
      patient_id: patientId ?? 0,
      category: 'その他',
      description: inputText.trim(),
      notes: '',
      is_slm_suggested: true,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    setLocalHistories((prev) => [...prev, newEntry]);
    setInputText('');
    setIsAdding(false);
  }, [inputText, patientId]);

  const handleCancel = useCallback(() => {
    setInputText('');
    setIsAdding(false);
  }, []);

  const allHistories = [...histories, ...localHistories];

  return (
    <div className="space-y-3">
      {allHistories.length === 0 && !isAdding && (
        <p className="text-sm text-gray-400 py-4 text-center">社会歴の記録はありません</p>
      )}

      {allHistories.map((h) => {
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

      {/* 追加フォーム */}
      {isAdding ? (
        <div className="bg-white rounded-lg border border-dashed border-primary-300 p-4 space-y-3">
          <InlineCompletionTextarea
            value={inputText}
            onChange={setInputText}
            context="social_history"
            patientId={patientId}
            placeholder="社会歴を入力... (例: 喫煙 - 20本/日、30年間)"
            rows={2}
            label="社会歴の追加"
            description="入力中にSLMが補完候補を表示します"
          />
          <div className="flex gap-2 justify-end">
            <button
              onClick={handleCancel}
              className="px-3 py-1.5 text-sm text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
            >
              キャンセル
            </button>
            <button
              onClick={handleSave}
              disabled={!inputText.trim()}
              className="px-3 py-1.5 text-sm text-white bg-primary-600 rounded-md hover:bg-primary-700 transition-colors disabled:opacity-50"
            >
              保存
            </button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setIsAdding(true)}
          className="w-full py-2.5 text-sm text-primary-600 bg-primary-50 border border-dashed border-primary-200 rounded-lg hover:bg-primary-100 transition-colors"
        >
          + 社会歴を追加
        </button>
      )}
    </div>
  );
}
