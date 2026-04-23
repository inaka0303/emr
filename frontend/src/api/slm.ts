import { post, get } from './client';
import type {
  SLMResponse,
  SLMSoapSuggestion,
} from '../types/api';

/**
 * encounter_id 単位のSOAPドラフト取得。
 * 初回はSLMで生成しDBに保存、以降は即座にキャッシュから返す。
 * force=true で再生成（キャッシュ上書き）。
 */
export async function fetchSoapDraft(
  encounterId: number,
  force = false,
): Promise<SLMResponse<SLMSoapSuggestion>> {
  return post<SLMResponse<SLMSoapSuggestion>>(`/encounters/${encounterId}/soap-draft`, { force });
}

/** SOAPドラフトキャッシュを削除 */
export async function invalidateSoapDraft(encounterId: number): Promise<void> {
  await fetch(`/api/encounters/${encounterId}/soap-draft`, { method: 'DELETE' });
}

/** 旧API（ad-hoc用）。encounter に紐づかない自由入力問診から生成。 */
export async function suggestSOAP(
  interviewText: string,
): Promise<SLMResponse<SLMSoapSuggestion>> {
  return post<SLMResponse<SLMSoapSuggestion>>('/slm/suggest/soap', {
    interview_text: interviewText,
  });
}

export interface AdmissionSummaryData {
  text: string;
  /**
   * どの推論サーバーで生成されたか。
   * "9B": 専用9Bサーバー（本番、最高品質）
   * "4B-fallback": 9B未接続時の4B admission LoRA（品質低下、UI で警告表示）
   * undefined: モック応答
   */
  server?: '9B' | '4B-fallback';
}

export async function suggestAdmissionSummary(
  interviewText: string,
  encounterId?: number,
): Promise<SLMResponse<AdmissionSummaryData>> {
  return post<SLMResponse<AdmissionSummaryData>>('/slm/suggest/admission', {
    interview_text: interviewText,
    ...(encounterId ? { encounter_id: encounterId } : {}),
  });
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
