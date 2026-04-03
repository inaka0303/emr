import type { Patient } from '../../types/api';
import { calcAge } from '../../utils/dateUtils';

interface SidebarProps {
  patients: Patient[];
  selectedPatientId: number | null;
  searchQuery: string;
  onSearchChange: (query: string) => void;
  onSelectPatient: (patient: Patient) => void;
}

export default function Sidebar({
  patients,
  selectedPatientId,
  searchQuery,
  onSearchChange,
  onSelectPatient,
}: SidebarProps) {
  return (
    <aside className="hidden lg:flex lg:flex-col lg:w-sidebar lg:min-w-[280px] bg-white border-r border-gray-200 h-screen">
      {/* ヘッダー */}
      <div className="p-4 border-b border-gray-200">
        <h1 className="text-lg font-bold text-primary-600 mb-3">
          EHR Demo
        </h1>
        <div className="relative">
          <input
            type="text"
            placeholder="患者検索（名前/カナ/MRN）"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="w-full pl-9 pr-3 py-2 text-sm border border-gray-300 rounded-lg
              focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
              placeholder:text-text-muted"
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

      {/* 患者リスト */}
      <div className="flex-1 overflow-y-auto">
        {patients.length === 0 ? (
          <p className="p-4 text-sm text-text-muted text-center">
            患者が見つかりません
          </p>
        ) : (
          <ul className="divide-y divide-gray-100">
            {patients.map((patient) => (
              <li key={patient.id}>
                <button
                  onClick={() => onSelectPatient(patient)}
                  className={`w-full text-left px-4 py-3 hover:bg-primary-50 transition-colors ${
                    selectedPatientId === patient.id
                      ? 'bg-primary-50 border-l-4 border-primary-600'
                      : ''
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-sm text-text">
                      {patient.name}
                    </span>
                    <span className="text-xs text-text-muted">
                      {patient.gender} / {calcAge(patient.birth_date)}歳
                    </span>
                  </div>
                  <div className="flex items-center justify-between mt-1">
                    <span className="text-xs text-text-secondary">
                      {patient.name_kana}
                    </span>
                    <span className="text-xs text-text-muted font-mono">
                      {patient.mrn}
                    </span>
                  </div>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* フッター */}
      <div className="p-3 border-t border-gray-200 text-center">
        <span className="text-xs text-text-muted">
          {patients.length}名の患者
        </span>
      </div>
    </aside>
  );
}
