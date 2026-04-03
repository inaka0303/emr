import { useState, useCallback } from 'react';
import { suggestSOAP, suggestSummary } from '../api/slm';
import type {
  SLMSoapSuggestion,
  SLMSummarySuggestionItem,
  SLMMeta,
} from '../types/api';

export interface SuggestionState {
  suggestions: SLMSoapSuggestion | null;
  acceptedFields: Set<keyof SLMSoapSuggestion>;
  dismissedFields: Set<keyof SLMSoapSuggestion>;
  activeField: keyof SLMSoapSuggestion | null;
  isLoading: boolean;
  meta: SLMMeta | null;
  error: string | null;
}

const SOAP_FIELDS: (keyof SLMSoapSuggestion)[] = [
  'subjective',
  'objective',
  'assessment',
  'plan',
];

export function useSLMSuggestion() {
  const [state, setState] = useState<SuggestionState>({
    suggestions: null,
    acceptedFields: new Set(),
    dismissedFields: new Set(),
    activeField: null,
    isLoading: false,
    meta: null,
    error: null,
  });

  const requestSOAP = useCallback(async (interviewText: string) => {
    setState((prev) => ({
      ...prev,
      isLoading: true,
      error: null,
      suggestions: null,
      acceptedFields: new Set(),
      dismissedFields: new Set(),
      activeField: null,
      meta: null,
    }));

    try {
      const response = await suggestSOAP(interviewText);
      setState((prev) => ({
        ...prev,
        isLoading: false,
        suggestions: response.data,
        activeField: 'subjective',
        meta: response.meta,
      }));
    } catch (err) {
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error: err instanceof Error ? err.message : 'SOAP提案の取得に失敗しました',
      }));
    }
  }, []);

  const acceptField = useCallback((field: keyof SLMSoapSuggestion) => {
    setState((prev) => {
      const newAccepted = new Set(prev.acceptedFields);
      newAccepted.add(field);
      // 次の未処理フィールドにフォーカスを移動
      const currentIndex = SOAP_FIELDS.indexOf(field);
      let nextField: keyof SLMSoapSuggestion | null = null;
      for (let i = currentIndex + 1; i < SOAP_FIELDS.length; i++) {
        const f = SOAP_FIELDS[i];
        if (!newAccepted.has(f) && !prev.dismissedFields.has(f)) {
          nextField = f;
          break;
        }
      }
      return {
        ...prev,
        acceptedFields: newAccepted,
        activeField: nextField,
      };
    });
  }, []);

  const dismissField = useCallback((field: keyof SLMSoapSuggestion) => {
    setState((prev) => {
      const newDismissed = new Set(prev.dismissedFields);
      newDismissed.add(field);
      // 次の未処理フィールドにフォーカスを移動
      const currentIndex = SOAP_FIELDS.indexOf(field);
      let nextField: keyof SLMSoapSuggestion | null = null;
      for (let i = currentIndex + 1; i < SOAP_FIELDS.length; i++) {
        const f = SOAP_FIELDS[i];
        if (!prev.acceptedFields.has(f) && !newDismissed.has(f)) {
          nextField = f;
          break;
        }
      }
      return {
        ...prev,
        dismissedFields: newDismissed,
        activeField: nextField,
      };
    });
  }, []);

  const moveToNextField = useCallback(() => {
    setState((prev) => {
      if (!prev.activeField) return prev;
      const currentIndex = SOAP_FIELDS.indexOf(prev.activeField);
      for (let i = currentIndex + 1; i < SOAP_FIELDS.length; i++) {
        const f = SOAP_FIELDS[i];
        if (!prev.acceptedFields.has(f) && !prev.dismissedFields.has(f)) {
          return { ...prev, activeField: f };
        }
      }
      // ループバック
      for (let i = 0; i < currentIndex; i++) {
        const f = SOAP_FIELDS[i];
        if (!prev.acceptedFields.has(f) && !prev.dismissedFields.has(f)) {
          return { ...prev, activeField: f };
        }
      }
      return prev;
    });
  }, []);

  const setActiveField = useCallback((field: keyof SLMSoapSuggestion | null) => {
    setState((prev) => ({ ...prev, activeField: field }));
  }, []);

  const reset = useCallback(() => {
    setState({
      suggestions: null,
      acceptedFields: new Set(),
      dismissedFields: new Set(),
      activeField: null,
      isLoading: false,
      meta: null,
      error: null,
    });
  }, []);

  return {
    state,
    requestSOAP,
    acceptField,
    dismissField,
    moveToNextField,
    setActiveField,
    reset,
  };
}

export interface SummaryState {
  suggestions: SLMSummarySuggestionItem[];
  acceptedItems: Set<number>;
  dismissedItems: Set<number>;
  isLoading: boolean;
  meta: SLMMeta | null;
  error: string | null;
  category: 'family_history' | 'social_history' | null;
}

export function useSLMSummary() {
  const [state, setState] = useState<SummaryState>({
    suggestions: [],
    acceptedItems: new Set(),
    dismissedItems: new Set(),
    isLoading: false,
    meta: null,
    error: null,
    category: null,
  });

  const requestSummary = useCallback(
    async (
      interviewText: string,
      category: 'family_history' | 'social_history',
    ) => {
      setState((prev) => ({
        ...prev,
        isLoading: true,
        error: null,
        suggestions: [],
        acceptedItems: new Set(),
        dismissedItems: new Set(),
        meta: null,
        category,
      }));

      try {
        const response = await suggestSummary(interviewText, category);
        setState((prev) => ({
          ...prev,
          isLoading: false,
          suggestions: response.data.suggestions,
          meta: response.meta,
        }));
      } catch (err) {
        setState((prev) => ({
          ...prev,
          isLoading: false,
          error:
            err instanceof Error ? err.message : '要約提案の取得に失敗しました',
        }));
      }
    },
    [],
  );

  const acceptItem = useCallback((index: number) => {
    setState((prev) => {
      const newAccepted = new Set(prev.acceptedItems);
      newAccepted.add(index);
      return { ...prev, acceptedItems: newAccepted };
    });
  }, []);

  const dismissItem = useCallback((index: number) => {
    setState((prev) => {
      const newDismissed = new Set(prev.dismissedItems);
      newDismissed.add(index);
      return { ...prev, dismissedItems: newDismissed };
    });
  }, []);

  const acceptAll = useCallback(() => {
    setState((prev) => {
      const newAccepted = new Set(prev.acceptedItems);
      prev.suggestions.forEach((_, i) => {
        if (!prev.dismissedItems.has(i)) {
          newAccepted.add(i);
        }
      });
      return { ...prev, acceptedItems: newAccepted };
    });
  }, []);

  const reset = useCallback(() => {
    setState({
      suggestions: [],
      acceptedItems: new Set(),
      dismissedItems: new Set(),
      isLoading: false,
      meta: null,
      error: null,
      category: null,
    });
  }, []);

  return {
    state,
    requestSummary,
    acceptItem,
    dismissItem,
    acceptAll,
    reset,
  };
}
