export interface PatientInfoContext {
  age?: number;
  gender?: string;
  blood_type?: string | null;
  encounter_date?: string;
  encounter_type?: string;
  department?: string;
  chief_complaint?: string;
  secondary_complaints?: string[];
  comorbidities?: string[];
  current_medications?: string[];
  allergies?: string[];
  family_history?: string[];
  social_history?: string;
  reception_vitals?: Record<string, string | number | null | undefined>;
}

export interface ExperimentStructuredData {
  patient_info?: PatientInfoContext;
  experiment?: {
    attempt_id?: string;
    subject_id?: string;
    case_id?: string;
    source_case_id?: string;
    intervention?: string;
    sequence_order?: number;
  };
}

export function parseStructuredData(value: unknown): ExperimentStructuredData | null {
  if (!value) return null;
  if (typeof value === 'object') return value as ExperimentStructuredData;
  if (typeof value !== 'string') return null;
  try {
    return JSON.parse(value) as ExperimentStructuredData;
  } catch {
    return null;
  }
}

export function formatVitals(vitals?: PatientInfoContext['reception_vitals']): string {
  if (!vitals) return '';
  const value = (key: string) => {
    const v = vitals[key];
    return v == null || v === '' ? '' : String(v);
  };
  const parts: string[] = [];
  const sys = value('BP_sys');
  const dia = value('BP_dia');
  if (sys && dia) parts.push(`BP ${sys}/${dia} mmHg`);
  const ordered = [
    ['HR', 'HR', '/min'],
    ['SpO2', 'SpO2', '%'],
    ['RR', 'RR', '/min'],
    ['BT', 'BT', '℃'],
  ] as const;
  for (const [key, label, unit] of ordered) {
    const v = value(key);
    if (v) parts.push(`${label} ${v}${unit}`);
  }
  return parts.join(', ');
}

export function formatPatientInfoForPrompt(info?: PatientInfoContext): string {
  if (!info) return '';
  const lines: string[] = [];
  const add = (label: string, value?: string) => {
    const text = (value ?? '').trim();
    if (text) lines.push(`${label}: ${text}`);
  };
  if (info.age || info.gender) add('年齢/性別', `${info.age ?? ''}歳 ${info.gender ?? ''}`.trim());
  add('受診', [info.encounter_date, info.department, info.encounter_type].filter(Boolean).join(' / '));
  add('主訴', info.chief_complaint);
  add('副主訴', (info.secondary_complaints ?? []).join('、'));
  add('既往', (info.comorbidities ?? []).join('、'));
  add('持参薬', (info.current_medications ?? []).join('、'));
  add('アレルギー', (info.allergies ?? []).join('、'));
  add('家族歴', (info.family_history ?? []).join('、'));
  add('社会歴', info.social_history);
  add('受付バイタル', formatVitals(info.reception_vitals));
  return lines.join('\n');
}
