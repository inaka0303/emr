import { useState, useCallback } from 'react';
import type { FamilyHistory } from '../../types/api';
import InlineCompletionTextarea from '../slm/InlineCompletionTextarea';

interface FamilyHistoryListProps {
  histories: FamilyHistory[];
  patientId?: number;
}

export default function FamilyHistoryList({ histories, patientId }: FamilyHistoryListProps) {
  const [isAdding, setIsAdding] = useState(false);
  const [inputText, setInputText] = useState('');
  const [localHistories, setLocalHistories] = useState<FamilyHistory[]>([]);

  const handleSave = useCallback(() => {
    if (!inputText.trim()) return;

    // ローカルに追加（デモ用 -- 実際はAPI経由で保存）
    const newEntry: FamilyHistory = {
      id: Date.now(),
      patient_id: patientId ?? 0,
      relation: '',
      condition: inputText.trim(),
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
        <p className="text-sm text-gray-400 py-4 text-center">家族歴の記録はありません</p>
      )}

      {allHistories.map((h) => (
        <div
          key={h.id}
          className="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-sm transition-shadow"
        >
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center justify-center w-8 h-8 rounded-full bg-indigo-50 text-indigo-700 text-xs font-bold flex-shrink-0">
              {h.relation || '他'}
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

      {/* 追加フォーム */}
      {isAdding ? (
        <div className="bg-white rounded-lg border border-dashed border-primary-300 p-4 space-y-3">
          <InlineCompletionTextarea
            value={inputText}
            onChange={setInputText}
            context="family_history"
            patientId={patientId}
            placeholder="家族歴を入力... (例: 父 - 2型糖尿病)"
            rows={2}
            label="家族歴の追加"
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
          + 家族歴を追加
        </button>
      )}
    </div>
  );
}
