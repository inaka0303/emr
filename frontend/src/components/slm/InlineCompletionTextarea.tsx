import { useRef, useState, useCallback, useEffect } from 'react';
import { useInlineCompletion } from '../../hooks/useInlineCompletion';

interface InlineCompletionTextareaProps {
  value: string;
  onChange: (value: string) => void;
  context: string;
  patientId?: number;
  placeholder?: string;
  rows?: number;
  className?: string;
  label?: string;
  description?: string;
}

export default function InlineCompletionTextarea({
  value,
  onChange,
  context,
  patientId,
  placeholder,
  rows = 3,
  className = '',
  label,
  description,
}: InlineCompletionTextareaProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const overlayRef = useRef<HTMLDivElement>(null);
  const [isFocused, setIsFocused] = useState(false);
  const [flashAccept, setFlashAccept] = useState(false);

  const { completion, isLoading, accept, dismiss } = useInlineCompletion(
    value,
    { context, patientId },
  );

  // テキストエリアとオーバーレイのスクロールを同期
  const syncScroll = useCallback(() => {
    if (textareaRef.current && overlayRef.current) {
      overlayRef.current.scrollTop = textareaRef.current.scrollTop;
      overlayRef.current.scrollLeft = textareaRef.current.scrollLeft;
    }
  }, []);

  // 自動リサイズ
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${Math.max(el.scrollHeight, rows * 24)}px`;
  }, [value, rows]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Tab' && completion) {
        e.preventDefault();
        const fullText = accept();
        onChange(fullText);
        // フラッシュアニメーション
        setFlashAccept(true);
        setTimeout(() => setFlashAccept(false), 400);
        return;
      }
      if (e.key === 'Escape' && completion) {
        e.preventDefault();
        dismiss();
        return;
      }
    },
    [completion, accept, dismiss, onChange],
  );

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      onChange(e.target.value);
    },
    [onChange],
  );

  // フォント等のスタイル一致用の共通クラス
  const sharedTextStyle =
    'text-sm font-sans leading-relaxed whitespace-pre-wrap break-words';
  const sharedPadding = 'px-3 py-2';

  return (
    <div className={className}>
      {label && (
        <label className="block text-sm font-medium text-gray-700 mb-1">
          {label}
        </label>
      )}
      {description && (
        <p className="text-xs text-gray-400 mb-1">{description}</p>
      )}
      <div
        className={`relative rounded-md border transition-all ${
          flashAccept
            ? 'border-emerald-400 bg-emerald-50'
            : isFocused
              ? 'border-primary-400 ring-2 ring-primary-200'
              : 'border-gray-200'
        }`}
      >
        {/* 表示レイヤー（下層） */}
        <div
          ref={overlayRef}
          aria-hidden="true"
          className={`absolute inset-0 overflow-hidden pointer-events-none ${sharedTextStyle} ${sharedPadding}`}
        >
          {/* 入力テキスト（透明 -- 位置を合わせるためだけ） */}
          <span className="invisible">{value}</span>
          {/* 補完テキスト（グレー、イタリック） */}
          {completion && (
            <span
              className={`italic text-gray-400 transition-opacity duration-200 ${
                isFocused ? 'opacity-70' : 'opacity-50'
              }`}
            >
              {completion}
            </span>
          )}
        </div>

        {/* 実際の textarea（上層） */}
        <textarea
          ref={textareaRef}
          value={value}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setIsFocused(false)}
          onScroll={syncScroll}
          rows={rows}
          placeholder={placeholder}
          className={`relative z-10 w-full bg-transparent resize-none focus:outline-none ${sharedTextStyle} ${sharedPadding} placeholder:text-gray-300`}
          style={{ caretColor: 'auto' }}
        />

        {/* ローディングインジケーター */}
        {isLoading && (
          <div className="absolute top-2 right-2 z-20">
            <div className="w-3 h-3 border-2 border-primary-300 border-t-transparent rounded-full animate-spin" />
          </div>
        )}
      </div>

      {/* ヒント */}
      {completion && isFocused && (
        <p className="text-xs text-gray-400 mt-1">
          <kbd className="px-1 py-0.5 text-xs bg-gray-100 border border-gray-200 rounded">
            Tab
          </kbd>{' '}
          で補完を適用 /{' '}
          <kbd className="px-1 py-0.5 text-xs bg-gray-100 border border-gray-200 rounded">
            Esc
          </kbd>{' '}
          で却下
        </p>
      )}
    </div>
  );
}
