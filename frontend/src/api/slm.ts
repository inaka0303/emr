import { post, get } from './client';
import type {
  SLMResponse,
  SLMSoapSuggestion,
  SLMSummarySuggestionItem,
} from '../types/api';

export async function suggestSOAP(
  interviewText: string,
): Promise<SLMResponse<SLMSoapSuggestion>> {
  return post<SLMResponse<SLMSoapSuggestion>>('/slm/suggest/soap', {
    interview_text: interviewText,
  });
}

export async function suggestSummary(
  interviewText: string,
  category: 'family_history' | 'social_history',
): Promise<SLMResponse<{ suggestions: SLMSummarySuggestionItem[] }>> {
  return post<SLMResponse<{ suggestions: SLMSummarySuggestionItem[] }>>(
    '/slm/suggest/summary',
    { interview_text: interviewText, category },
  );
}

export interface SLMHealthResponse {
  data: {
    status: 'connected' | 'mock_mode';
    api_url: string;
    model: string;
  };
}

export async function getSLMHealth(): Promise<SLMHealthResponse> {
  return get<SLMHealthResponse>('/slm/health');
}
