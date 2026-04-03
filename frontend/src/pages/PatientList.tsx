import type { Patient } from '../types/api';
import { calcAge } from '../utils/dateUtils';

interface PatientListProps {
  patients: Patient[];
  searchQuery: string;
  onSearchChange: (query: string) => void;
  onSelectPatient: (patient: Patient) => void;
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')}`;
}

export default function PatientList({
  patients,
  searchQuery,
  onSearchChange,
  onSelectPatient,
}: PatientListProps) {
  return (
    <div>
      {/* ヘッダー */}
      <div className="mb-4">
        <h1 className="text-xl font-bold text-text mb-3">患者一覧</h1>
        <div className="relative">
          <input
            type="text"
            placeholder="患者検索（名前/カナ/MRN）"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="w-full pl-9 pr-3 py-2.5 text-sm border border-gray-300 rounded-lg
              focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
              placeholder:text-text-muted bg-white"
          />
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
        </div>
      </div>

      {/* 結果数 */}
      <p className="text-xs text-text-muted mb-3">{patients.length}名の患者</p>

      {/* 患者カード一覧 */}
      {patients.length === 0 ? (
        <div className="text-center py-12">
          <svg className="w-12 h-12 mx-auto text-text-muted mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <p className="text-sm text-text-muted">該当する患者が見つかりません</p>
        </div>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2">
          {patients.map((patient) => (
            <button
              key={patient.id}
              onClick={() => onSelectPatient(patient)}
              className="bg-white rounded-lg border border-gray-200 p-4 text-left
                hover:border-primary-300 hover:shadow-sm transition-all active:scale-[0.98]"
            >
              <div className="flex items-start justify-between mb-2">
                <div>
                  <span className="font-semibold text-text text-sm">
                    {patient.name}
                  </span>
                  <span className="text-xs text-text-muted ml-2">
                    {patient.name_kana}
                  </span>
                </div>
                <span className="text-xs font-mono text-text-muted bg-gray-100 px-2 py-0.5 rounded">
                  {patient.mrn}
                </span>
              </div>

              <div className="flex items-center gap-3 text-xs text-text-secondary">
                <span>{patient.gender}</span>
                <span>{calcAge(patient.birth_date)}歳</span>
                <span>{patient.blood_type}</span>
              </div>

              <div className="mt-2 pt-2 border-t border-gray-100 flex items-center justify-between">
                <span className="text-xs text-text-muted">
                  最終更新: {formatDate(patient.updated_at)}
                </span>
                <svg className="w-4 h-4 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
