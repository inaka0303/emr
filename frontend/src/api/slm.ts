import { post } from './client';
import type { SLMResponse } from '../types/api';

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
