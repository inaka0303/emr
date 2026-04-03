import { useState, useMemo } from 'react';
import type { Patient, MedicalHistory, FamilyHistory, SocialHistory } from '../../types/api';
import { calcAge } from '../../utils/dateUtils';
import MedicalHistoryList from './MedicalHistoryList';
import FamilyHistoryList from './FamilyHistoryList';
import SocialHistoryList from './SocialHistoryList';

interface PatientDetailProps {
  patient: Patient;
  medicalHistories: MedicalHistory[];
  familyHistories: FamilyHistory[];
  socialHistories: SocialHistory[];
}

type SubTab = 'basic' | 'medical' | 'family' | 'social';

interface SubTabItem {
  id: SubTab;
  label: string;
}

const SUB_TABS: SubTabItem[] = [
  { id: 'basic', label: '基本情報' },
  { id: 'medical', label: '既往歴' },
  { id: 'family', label: '家族歴' },
  { id: 'social', label: '社会歴' },
];

export default function PatientDetail({
  patient,
  medicalHistories,
  familyHistories,
  socialHistories,
}: PatientDetailProps) {
  const [activeSubTab, setActiveSubTab] = useState<SubTab>('basic');

  const patientMedical = useMemo(
    () => medicalHistories.filter((h) => h.patient_id === patient.id),
    [medicalHistories, patient.id],
  );

  const patientFamily = useMemo(
    () => familyHistories.filter((h) => h.patient_id === patient.id),
    [familyHistories, patient.id],
  );

  const patientSocial = useMemo(
    () => socialHistories.filter((h) => h.patient_id === patient.id),
    [socialHistories, patient.id],
  );

  return (
    <div>
      <h2 className="text-lg font-bold text-gray-800 mb-4">
        患者情報 - {patient.name}
      </h2>

      {/* サブタブ */}
      <div className="flex gap-1 mb-4 bg-gray-100 rounded-lg p-1">
        {SUB_TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveSubTab(tab.id)}
            className={`flex-1 px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeSubTab === tab.id
                ? 'bg-white text-primary-600 shadow-sm'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* コンテンツ */}
      {activeSubTab === 'basic' && (
        <div className="bg-white rounded-lg border border-gray-200 divide-y divide-gray-100">
          <InfoRow label="MRN" value={patient.mrn} />
          <InfoRow label="氏名" value={patient.name} />
          <InfoRow label="フリガナ" value={patient.name_kana} />
          <InfoRow label="生年月日" value={`${patient.birth_date}（${calcAge(patient.birth_date)}歳）`} />
          <InfoRow label="性別" value={patient.gender} />
          <InfoRow label="血液型" value={patient.blood_type} />
          <InfoRow label="電話番号" value={patient.phone} />
          <InfoRow label="住所" value={patient.address} />
          <InfoRow
            label="緊急連絡先"
            value={patient.emergency_contact_name
              ? `${patient.emergency_contact_name}（${patient.emergency_contact_phone}）`
              : '-'}
          />
        </div>
      )}

      {activeSubTab === 'medical' && (
        <MedicalHistoryList histories={patientMedical} />
      )}

      {activeSubTab === 'family' && (
        <FamilyHistoryList histories={patientFamily} />
      )}

      {activeSubTab === 'social' && (
        <SocialHistoryList histories={patientSocial} />
      )}
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center px-4 py-3">
      <span className="text-sm text-gray-500 w-28 flex-shrink-0">{label}</span>
      <span className="text-sm text-gray-800">{value}</span>
    </div>
  );
}
