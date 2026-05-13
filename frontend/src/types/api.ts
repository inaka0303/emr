// 患者
export interface Patient {
  id: number;
  mrn: string;
  name: string;
  name_kana: string;
  birth_date: string;
  gender: string;
  blood_type: string;
  phone: string;
  address: string;
  emergency_contact_name: string;
  emergency_contact_phone: string;
  created_at: string;
  updated_at: string;
}

// 受診
export interface Encounter {
  id: number;
  patient_id: number;
  encounter_date: string;
  encounter_type: '外来' | '入院' | '救急';
  department: string;
  attending_doctor: string;
  status: '進行中' | '完了';
  chief_complaint: string;
  created_at: string;
  updated_at: string;
}

// SOAP記録
export interface SOAPNote {
  id: number;
  encounter_id: number;
  author: string;
  subjective: string;
  objective: string;
  assessment: string;
  plan: string;
  is_slm_suggested: boolean;
  created_at: string;
  updated_at: string;
}

// 既往歴
export interface MedicalHistory {
  id: number;
  patient_id: number;
  condition: string;
  onset_date: string;
  status: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

// 家族歴
export interface FamilyHistory {
  id: number;
  patient_id: number;
  relation: string;
  condition: string;
  notes: string;
  is_slm_suggested: boolean;
  created_at: string;
  updated_at: string;
}

// 社会歴
export interface SocialHistory {
  id: number;
  patient_id: number;
  category: '喫煙' | '飲酒' | '職業' | '運動' | string;
  description: string;
  notes: string;
  is_slm_suggested: boolean;
  created_at: string;
  updated_at: string;
}

// 問診記録
export interface InterviewNote {
  id: number;
  encounter_id: number;
  raw_text: string;
  structured_data: Record<string, unknown> | null;
  created_at: string;
}

// SLM SOAP提案レスポンス
export interface SLMSoapSuggestion {
  subjective: string;
  objective: string;
  assessment: string;
  plan: string;
}

// SLM メタ情報
export interface SLMMeta {
  model: string;
  is_mock: boolean;
  latency_ms?: number;
  /** SOAPドラフトがDBキャッシュから返された場合 true */
  cached?: boolean;
}

// ページネーション用メタ情報
export interface PaginationMeta {
  total: number;
  page: number;
  limit: number;
}

// APIレスポンス共通
export interface ApiResponse<T> {
  data: T;
  error?: ApiError;
  meta?: PaginationMeta;
}

// 実験attempt
export interface ExperimentAttempt {
  attempt_id: string;
  subject_id: string;
  case_id: string;
  source_case_id: string;
  intervention: 'ai' | 'control';
  sequence_order: number;
  patient_id: number;
  encounter_id: number;
  status: 'ready' | 'in_progress' | 'finished' | 'abandoned';
  started_at?: string;
  ended_at?: string;
  duration_sec: number;
  interruption_sec: number;
  ai_wait_ms: number;
  ai_candidate_count: number;
  ai_accept_count: number;
  ai_edit_count: number;
  ai_reject_count: number;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface ExperimentFinishInput {
  duration_sec: number;
  interruption_sec: number;
  ai_wait_ms?: number;
  ai_candidate_count?: number;
  ai_accept_count: number;
  ai_edit_count: number;
  ai_reject_count: number;
  notes?: string;
}

// SLMレスポンス
export interface SLMResponse<T> {
  data: T;
  meta: SLMMeta;
}

// APIエラー
export interface ApiError {
  code: string;
  message: string;
}

// ナビゲーションタブ
export type MainTab = 'chart' | 'history' | 'patient-info';
export type MobileTab = 'patients' | 'chart' | 'settings';
