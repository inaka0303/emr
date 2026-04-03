import { useState, useEffect } from 'react';
import { getSLMHealth } from '../../api/slm';
import type { SLMSoapSuggestion, SLMMeta } from '../../types/api';
import type { SuggestionState, SummaryState } from '../../hooks/useSLMSuggestion';

interface SuggestionPanelProps {
  visible: boolean;
  soapState: SuggestionState | null;
  summaryState: SummaryState | null;
}

type SLMStatus = 'loading' | 'connected' | 'mock_mode' | 'error';

export default function SuggestionPanel({
  visible,
  soapState,
  summaryState,
}: SuggestionPanelProps) {
  const [slmStatus, setSlmStatus] = useState<SLMStatus>('loading');
  const [slmModel, setSlmModel] = useState('qwen3.5-0.8b-medical');

  useEffect(() => {
    if (!visible) return;

    const fetchHealth = async () => {
      try {
        const resp = await getSLMHealth();
        setSlmStatus(resp.data.status === 'connected' ? 'connected' : 'mock_mode');
        setSlmModel(resp.data.model || 'qwen3.5-0.8b-medical');
      } catch {
        setSlmStatus('mock_mode');
      }
    };

    fetchHealth();
    const interval = setInterval(fetchHealth, 30000);
    return () => clearInterval(interval);
  }, [visible]);

  if (!visible) return null;

  return (
    <aside className="hidden lg:flex lg:flex-col lg:w-suggestion lg:min-w-[320px] bg-white border-l border-gray-200 h-screen">
      {/* SLMステータスヘッダー */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center gap-2">
          <StatusDot status={slmStatus} />
          <h2 className="text-sm font-semibold text-text">SLM提案</h2>
        </div>
        <div className="flex items-center justify-between mt-1">
          <p className="text-xs text-text-muted">
            モデル: {slmModel}
          </p>
          <StatusBadge status={slmStatus} />
        </div>
      </div>

      {/* 提案コンテンツ */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* SOAP提案プレビュー */}
        {soapState && soapState.isLoading && (
          <div className="space-y-3">
            <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
              SOAP提案を生成中...
            </h3>
            <SOAPSkeleton />
          </div>
        )}

        {soapState && soapState.suggestions && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
                SOAP提案
              </h3>
              <MetaInfo meta={soapState.meta} />
            </div>
            <SOAPPreview
              suggestions={soapState.suggestions}
              acceptedFields={soapState.acceptedFields}
              dismissedFields={soapState.dismissedFields}
              activeField={soapState.activeField}
            />
          </div>
        )}

        {soapState && soapState.error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-xs text-red-600">{soapState.error}</p>
          </div>
        )}

        {/* 要約提案プレビュー */}
        {summaryState && summaryState.isLoading && (
          <div className="space-y-3">
            <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
              {summaryState.category === 'family_history' ? '家族歴' : '社会歴'}を抽出中...
            </h3>
            <div className="space-y-2">
              <div className="animate-pulse bg-gray-200 rounded h-12 w-full" />
              <div className="animate-pulse bg-gray-200 rounded h-12 w-full" />
            </div>
          </div>
        )}

        {summaryState &&
          summaryState.suggestions.length > 0 &&
          !summaryState.isLoading && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  {summaryState.category === 'family_history'
                    ? '家族歴'
                    : '社会歴'}
                  提案 ({summaryState.suggestions.length}件)
                </h3>
                <MetaInfo meta={summaryState.meta} />
              </div>
              <div className="space-y-2">
                {summaryState.suggestions.map((item, idx) => {
                  const isAccepted = summaryState.acceptedItems.has(idx);
                  const isDismissed = summaryState.dismissedItems.has(idx);
                  if (isDismissed) return null;
                  return (
                    <div
                      key={idx}
                      className={`p-2 rounded text-xs border ${
                        isAccepted
                          ? 'bg-emerald-50 border-emerald-200'
                          : 'bg-gray-50 border-gray-200'
                      }`}
                    >
                      <span className="text-gray-500">{item.field}:</span>{' '}
                      <span className="font-medium text-gray-800">
                        {item.value}
                      </span>
                      <span className="ml-1 text-gray-400">
                        ({Math.round(item.confidence * 100)}%)
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

        {/* デフォルト状態（何も提案がない） */}
        {(!soapState || (!soapState.suggestions && !soapState.isLoading && !soapState.error)) &&
          (!summaryState || (summaryState.suggestions.length === 0 && !summaryState.isLoading && !summaryState.error)) && (
            <div className="rounded-lg bg-slm-green-light border border-slm-green/20 p-4">
              <p className="text-sm text-text-secondary">
                問診テキストを入力すると、SLMがSOAP形式の記録や
                家族歴・社会歴の要約を提案します。
              </p>
              <div className="mt-3 flex flex-col gap-1">
                <span className="text-xs text-text-muted">
                  <kbd className="px-1.5 py-0.5 bg-white rounded border border-gray-300 font-mono text-xs">
                    Tab
                  </kbd>{' '}
                  提案を採用
                </span>
                <span className="text-xs text-text-muted">
                  <kbd className="px-1.5 py-0.5 bg-white rounded border border-gray-300 font-mono text-xs">
                    Esc
                  </kbd>{' '}
                  提案を却下
                </span>
                <span className="text-xs text-text-muted">
                  <kbd className="px-1.5 py-0.5 bg-white rounded border border-gray-300 font-mono text-xs">
                    &rarr;
                  </kbd>{' '}
                  次の提案へ
                </span>
              </div>
            </div>
          )}
      </div>

      {/* メタ情報フッター */}
      <div className="p-3 border-t border-gray-200">
        <div className="flex items-center justify-between text-xs text-text-muted">
          <span>モデル: {slmModel}</span>
          <StatusBadge status={slmStatus} />
        </div>
      </div>
    </aside>
  );
}

function StatusDot({ status }: { status: SLMStatus }) {
  let color = 'bg-gray-300';
  if (status === 'connected') color = 'bg-emerald-400';
  else if (status === 'mock_mode') color = 'bg-amber-400';
  else if (status === 'error') color = 'bg-red-400';

  return (
    <div className="relative flex items-center">
      <div className={`w-2 h-2 rounded-full ${color}`} />
      {(status === 'connected' || status === 'loading') && (
        <div
          className={`absolute w-2 h-2 rounded-full ${color} animate-ping`}
        />
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: SLMStatus }) {
  if (status === 'connected') {
    return (
      <span className="px-2 py-0.5 bg-emerald-50 text-emerald-600 rounded-full text-xs font-medium">
        SLM接続中
      </span>
    );
  }
  if (status === 'mock_mode') {
    return (
      <span className="px-2 py-0.5 bg-slm-amber-light text-slm-amber rounded-full text-xs font-medium">
        モックモード
      </span>
    );
  }
  if (status === 'error') {
    return (
      <span className="px-2 py-0.5 bg-red-50 text-red-500 rounded-full text-xs font-medium">
        接続エラー
      </span>
    );
  }
  return (
    <span className="px-2 py-0.5 bg-gray-100 text-gray-400 rounded-full text-xs font-medium">
      確認中...
    </span>
  );
}

function MetaInfo({ meta }: { meta: SLMMeta | null }) {
  if (!meta) return null;
  return (
    <span className="text-[10px] text-gray-400">
      {meta.latency_ms ? `${meta.latency_ms}ms` : ''}
      {meta.is_mock ? ' (mock)' : ''}
    </span>
  );
}

function SOAPSkeleton() {
  return (
    <div className="space-y-2">
      {['S', 'O', 'A', 'P'].map((letter) => (
        <div key={letter} className="flex items-start gap-2">
          <span className="text-xs font-bold text-gray-300 w-4">
            {letter}
          </span>
          <div className="flex-1 space-y-1">
            <div className="animate-pulse bg-gray-200 rounded h-3 w-full" />
            <div className="animate-pulse bg-gray-200 rounded h-3 w-4/5" />
          </div>
        </div>
      ))}
    </div>
  );
}

interface SOAPPreviewProps {
  suggestions: SLMSoapSuggestion;
  acceptedFields: Set<keyof SLMSoapSuggestion>;
  dismissedFields: Set<keyof SLMSoapSuggestion>;
  activeField: keyof SLMSoapSuggestion | null;
}

function SOAPPreview({
  suggestions,
  acceptedFields,
  dismissedFields,
  activeField,
}: SOAPPreviewProps) {
  const fields: { key: keyof SLMSoapSuggestion; letter: string }[] = [
    { key: 'subjective', letter: 'S' },
    { key: 'objective', letter: 'O' },
    { key: 'assessment', letter: 'A' },
    { key: 'plan', letter: 'P' },
  ];

  return (
    <div className="space-y-2">
      {fields.map(({ key, letter }) => {
        const isActive = activeField === key;
        const isAccepted = acceptedFields.has(key);
        const isDismissed = dismissedFields.has(key);
        const text = suggestions[key];

        return (
          <div
            key={key}
            className={`flex items-start gap-2 p-2 rounded transition-colors ${
              isActive
                ? 'bg-primary-50 border border-primary-200'
                : isAccepted
                  ? 'bg-emerald-50 border border-emerald-100'
                  : isDismissed
                    ? 'opacity-40'
                    : 'bg-gray-50'
            }`}
          >
            <span
              className={`text-xs font-bold w-4 flex-shrink-0 ${
                isAccepted ? 'text-emerald-500' : 'text-gray-400'
              }`}
            >
              {letter}
            </span>
            <p
              className={`text-xs leading-relaxed line-clamp-3 ${
                isAccepted
                  ? 'text-gray-700'
                  : isDismissed
                    ? 'text-gray-400 line-through'
                    : 'text-gray-500'
              }`}
            >
              {text || '-'}
            </p>
          </div>
        );
      })}
    </div>
  );
}
