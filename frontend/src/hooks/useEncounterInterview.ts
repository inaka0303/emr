import { useState, useEffect } from 'react';
import { get } from '../api/client';
import type { ApiResponse } from '../types/api';
import { formatPatientInfoForPrompt, parseStructuredData, type ExperimentStructuredData } from '../utils/patientContext';

export interface InterviewNote {
  id: number;
  encounter_id: number;
  /** 問診記録（患者から聞き取る: 主訴・現病歴・既往・家族歴・社会歴・アレルギー） */
  raw_text: string;
  /** お薬手帳（持参薬の正確な名前・用量） */
  medication_list: string;
  /** 診察所見（医師が視触聴診で取る客観的身体所見） */
  exam_findings: string;
  /** 検査結果（バイタル含む、採血・画像・心電図など） */
  lab_results: string;
  structured_data?: ExperimentStructuredData | string | null;
  created_at: string;
}

interface UseEncounterInterviewResult {
  notes: InterviewNote[];
  /** 全 note の raw_text を改行で連結したもの。SLMへの入力に使う。 */
  combinedText: string;
  isLoading: boolean;
  error: string | null;
}

export function useEncounterInterview(encounterId: number | null): UseEncounterInterviewResult {
  const [notes, setNotes] = useState<InterviewNote[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (encounterId == null) {
      setNotes([]);
      setError(null);
      return;
    }
    let cancelled = false;
    setIsLoading(true);
    setError(null);
    get<ApiResponse<InterviewNote[]>>(`/encounters/${encounterId}/interviews`)
      .then((res) => {
        if (cancelled) return;
        setNotes(res.data ?? []);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : '問診の取得に失敗しました');
        setNotes([]);
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [encounterId]);

  // SLM に渡す hybrid format（バックエンドの buildHybridInput と同形式）
  const combineOne = (n: InterviewNote): string => {
    const parts: string[] = [];
    const push = (header: string, body: string) => {
      const t = (body ?? '').trim();
      if (t) parts.push(`${header}\n${t}`);
    };
    const structured = parseStructuredData(n.structured_data);
    push('【患者情報】', formatPatientInfoForPrompt(structured?.patient_info));
    push('【問診記録】', n.raw_text);
    push('【お薬手帳より】', n.medication_list);
    push('【診察所見メモ】', n.exam_findings);
    push('【検査結果】', n.lab_results);
    return parts.join('\n\n');
  };
  const combinedText = notes.map(combineOne).filter(Boolean).join('\n\n');
  return { notes, combinedText, isLoading, error };
}
