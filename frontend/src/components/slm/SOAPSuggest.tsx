import { useEffect, useCallback } from 'react';
import type { SLMSoapSuggestion } from '../../types/api';
import type { SuggestionState } from '../../hooks/useSLMSuggestion';
import { useTypingEffect } from '../../hooks/useTypingEffect';

interface SectionConfig {
  key: keyof SLMSoapSuggestion;
  label: string;
  letter: string;
  description: string;
  borderColor: string;
  activeBorderColor: string;
  labelColor: string;
  bgColor: string;
}

const SECTIONS: SectionConfig[] = [
  {
    key: 'subjective',
    label: '主観的所見',
    letter: 'S',
    description: '患者の訴え、自覚症状',
    borderColor: 'border-blue-400',
    activeBorderColor: 'border-blue-600',
    labelColor: 'text-blue-700 bg-blue-50',
    bgColor: 'focus-within:ring-blue-300',
  },
  {
    key: 'objective',
    label: '客観的所見',
    letter: 'O',
    description: '検査結果、バイタル、身体所見',
    borderColor: 'border-emerald-400',
    activeBorderColor: 'border-emerald-600',
    labelColor: 'text-emerald-700 bg-emerald-50',
    bgColor: 'focus-within:ring-emerald-300',
  },
  {
    key: 'assessment',
    label: '評価',
    letter: 'A',
    description: '診断、アセスメント',
    borderColor: 'border-amber-400',
    activeBorderColor: 'border-amber-600',
    labelColor: 'text-amber-700 bg-amber-50',
    bgColor: 'focus-within:ring-amber-300',
  },
  {
    key: 'plan',
    label: '計画',
    letter: 'P',
    description: '治療計画、処方、次回予約',
    borderColor: 'border-purple-400',
    activeBorderColor: 'border-purple-600',
    labelColor: 'text-purple-700 bg-purple-50',
    bgColor: 'focus-within:ring-purple-300',
  },
];

interface SOAPSuggestProps {
  suggestionState: SuggestionState;
  /** 各フィールドの現在の編集値 */
  fieldValues: Record<keyof SLMSoapSuggestion, string>;
  /** フィールドの値をセットする */
  onFieldChange: (field: keyof SLMSoapSuggestion, value: string) => void;
  /** 提案をacceptする */
  onAccept: (field: keyof SLMSoapSuggestion) => void;
  /** 提案をdismissする */
  onDismiss: (field: keyof SLMSoapSuggestion) => void;
  /** 次のフィールドに移動 */
  onMoveNext: () => void;
  /** アクティブフィールドをセット */
  onSetActiveField: (field: keyof SLMSoapSuggestion | null) => void;
  /** 保存 */
  onSave: () => void;
  saveStatus: 'idle' | 'saving' | 'success' | 'error';
  errorMessage: string;
  patientName: string;
  encounterId?: number;
}

function SkeletonLines() {
  return (
    <div className="space-y-2 py-2">
      <div className="animate-pulse bg-gray-200 rounded h-4 w-full" />
      <div className="animate-pulse bg-gray-200 rounded h-4 w-5/6" />
      <div className="animate-pulse bg-gray-200 rounded h-4 w-4/6" />
    </div>
  );
}

function TypingSuggestionText({ text, isActive }: { text: string; isActive: boolean }) {
  const { displayText, isTyping } = useTypingEffect(
    isActive ? text : '',
    25,
  );

  // アクティブでないフィールドは全文を静的に表示
  if (!isActive) {
    return (
      <span className="whitespace-pre-wrap leading-relaxed">{text}</span>
    );
  }

  return (
    <span className="whitespace-pre-wrap leading-relaxed">
      {displayText}
      {isTyping && (
        <span className="inline-block w-[2px] h-[1em] bg-gray-400 ml-[1px] align-text-bottom animate-pulse" />
      )}
    </span>
  );
}

export default function SOAPSuggest({
  suggestionState,
  fieldValues,
  onFieldChange,
  onAccept,
  onDismiss,
  onMoveNext,
  onSetActiveField,
  onSave,
  saveStatus,
  errorMessage,
  patientName,
}: SOAPSuggestProps) {
  const {
    suggestions,
    acceptedFields,
    dismissedFields,
    activeField,
    isLoading,
  } = suggestionState;

  // グローバルキーボードハンドラー
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!suggestions || !activeField) return;
      // accept済みやdismiss済みのフィールドでは操作しない
      if (acceptedFields.has(activeField) || dismissedFields.has(activeField))
        return;

      if (e.key === 'Tab' && !e.shiftKey) {
        e.preventDefault();
        // 提案をacceptし、テキストをセット
        const value = suggestions[activeField];
        if (value) {
          onFieldChange(activeField, value);
          onAccept(activeField);
        }
      } else if (e.key === 'Escape') {
        e.preventDefault();
        onDismiss(activeField);
      } else if (e.key === 'ArrowRight') {
        const target = e.target as HTMLElement;
        // textarea/input にフォーカスがある場合はカーソル移動を優先
        if (target.tagName !== 'TEXTAREA' && target.tagName !== 'INPUT') {
          e.preventDefault();
          onMoveNext();
        }
      }
    },
    [
      suggestions,
      activeField,
      acceptedFields,
      dismissedFields,
      onFieldChange,
      onAccept,
      onDismiss,
      onMoveNext,
    ],
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  const hasSuggestions = suggestions !== null;

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
      <div className="px-4 py-3 border-b border-gray-100">
        <h3 className="text-base font-semibold text-gray-800">
          {hasSuggestions ? 'SOAP提案 (SLM)' : '新規カルテ記録'}
        </h3>
        <p className="text-xs text-gray-500 mt-0.5">
          {patientName} - SOAP形式
          {hasSuggestions && (
            <span className="ml-2 text-primary-600">
              Tab: 採用 / Esc: 却下 / →: 次へ
            </span>
          )}
        </p>
      </div>

      <div className="p-4 space-y-4">
        {SECTIONS.map((section) => {
          const isActive = activeField === section.key;
          const isAccepted = acceptedFields.has(section.key);
          const isDismissed = dismissedFields.has(section.key);
          const suggestionText = suggestions?.[section.key] ?? '';
          const currentValue = fieldValues[section.key];
          const showSuggestion =
            hasSuggestions &&
            !isAccepted &&
            !isDismissed &&
            suggestionText;
          // currentValue がある場合は提案を薄く表示（opacity を下げる）
          const suggestionOpacity = currentValue ? 'opacity-25' : 'opacity-50';

          return (
            <div
              key={section.key}
              className={`border-l-4 pl-3 transition-all duration-200 ${
                isActive && hasSuggestions
                  ? `${section.activeBorderColor} border-l-[6px] bg-gray-50/50 rounded-r-lg pr-3 py-1`
                  : section.borderColor
              }`}
              onClick={() => {
                if (
                  hasSuggestions &&
                  !isAccepted &&
                  !isDismissed
                ) {
                  onSetActiveField(section.key);
                }
              }}
            >
              <div className="flex items-center gap-2 mb-1.5">
                <span
                  className={`inline-flex items-center justify-center w-6 h-6 rounded text-xs font-bold ${section.labelColor}`}
                >
                  {section.letter}
                </span>
                <span className="text-sm font-medium text-gray-700">
                  {section.label}
                </span>
                <span className="text-xs text-gray-400">
                  {section.description}
                </span>
                {isAccepted && (
                  <span className="ml-auto text-xs text-emerald-600 font-medium flex items-center gap-1">
                    <svg
                      className="w-3.5 h-3.5"
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
                    採用済み
                  </span>
                )}
                {isDismissed && (
                  <span className="ml-auto text-xs text-gray-400 font-medium">
                    却下
                  </span>
                )}
              </div>

              {/* テキストエリア */}
              <div className="relative">
                <textarea
                  value={currentValue}
                  onChange={(e) =>
                    onFieldChange(section.key, e.target.value)
                  }
                  rows={3}
                  placeholder={`${section.label}を入力...`}
                  className={`w-full px-3 py-2 text-sm border border-gray-200 rounded-md resize-none
                    focus:outline-none focus:ring-2 ${section.bgColor} focus:border-transparent
                    placeholder:text-gray-300 transition-colors ${
                      isAccepted ? 'text-gray-800' : ''
                    }`}
                />

                {/* ローディングスケルトン */}
                {isLoading && (
                  <div className="absolute inset-0 px-3 py-2">
                    <SkeletonLines />
                  </div>
                )}

                {/* 提案テキスト（Copilotスタイル + タイピングエフェクト） */}
                {showSuggestion && (
                  <div
                    className={`absolute inset-0 px-3 py-2 pointer-events-none transition-opacity duration-300 ${
                      currentValue
                        ? suggestionOpacity
                        : isActive
                          ? 'opacity-100'
                          : 'opacity-60'
                    }`}
                  >
                    <p className="text-sm text-gray-400 italic">
                      <TypingSuggestionText text={suggestionText} isActive={isActive} />
                    </p>
                  </div>
                )}
              </div>

              {/* 提案操作ボタン（アクティブフィールドに表示） */}
              {isActive && showSuggestion && (
                <div className="flex items-center gap-2 mt-1.5">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      if (suggestionText) {
                        onFieldChange(section.key, suggestionText);
                        onAccept(section.key);
                      }
                    }}
                    className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-emerald-700
                      bg-emerald-50 border border-emerald-200 rounded hover:bg-emerald-100 transition-colors"
                  >
                    <kbd className="px-1 py-0.5 bg-white rounded border border-emerald-300 font-mono text-[10px]">
                      Tab
                    </kbd>
                    採用
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onDismiss(section.key);
                    }}
                    className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-gray-500
                      bg-gray-50 border border-gray-200 rounded hover:bg-gray-100 transition-colors"
                  >
                    <kbd className="px-1 py-0.5 bg-white rounded border border-gray-300 font-mono text-[10px]">
                      Esc
                    </kbd>
                    却下
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onMoveNext();
                    }}
                    className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-gray-500
                      bg-gray-50 border border-gray-200 rounded hover:bg-gray-100 transition-colors"
                  >
                    <kbd className="px-1 py-0.5 bg-white rounded border border-gray-300 font-mono text-[10px]">
                      →
                    </kbd>
                    次へ
                  </button>
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="px-4 py-3 border-t border-gray-100 flex items-center justify-between">
        <div className="text-sm">
          {saveStatus === 'success' && (
            <span className="text-emerald-600">保存しました</span>
          )}
          {saveStatus === 'error' && (
            <span className="text-red-600">{errorMessage}</span>
          )}
          {suggestionState.meta && (
            <span className="text-xs text-gray-400 ml-2">
              {suggestionState.meta.is_mock ? 'モック' : 'SLM'} |{' '}
              {suggestionState.meta.latency_ms
                ? `${suggestionState.meta.latency_ms}ms`
                : ''}
            </span>
          )}
        </div>
        <button
          onClick={onSave}
          disabled={saveStatus === 'saving'}
          className="px-5 py-2 bg-primary-600 text-white text-sm font-medium rounded-lg
            hover:bg-primary-700 transition-colors focus:outline-none focus:ring-2
            focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50"
        >
          {saveStatus === 'saving' ? '保存中...' : '保存'}
        </button>
      </div>
    </div>
  );
}
