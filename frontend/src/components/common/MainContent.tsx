import type { MainTab } from '../../types/api';

interface MainContentProps {
  activeTab: MainTab;
  onTabChange: (tab: MainTab) => void;
  children: React.ReactNode;
}

interface TabItem {
  id: MainTab;
  label: string;
}

const tabs: TabItem[] = [
  { id: 'chart', label: 'カルテ' },
  { id: 'history', label: '履歴' },
  { id: 'patient-info', label: '患者情報' },
];

export default function MainContent({
  activeTab,
  onTabChange,
  children,
}: MainContentProps) {
  return (
    <main className="flex-1 flex flex-col min-w-0 h-screen overflow-hidden">
      {/* タブバー (PC版) */}
      <div className="hidden lg:block border-b border-gray-200 bg-white">
        <div className="flex">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              className={`px-6 py-3 text-sm font-medium transition-colors border-b-2 ${
                activeTab === tab.id
                  ? 'text-primary-600 border-primary-600'
                  : 'text-text-secondary border-transparent hover:text-text hover:border-gray-300'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {/* コンテンツエリア */}
      <div className="flex-1 overflow-y-auto p-4 lg:p-6 pb-20 lg:pb-6">
        {children}
      </div>
    </main>
  );
}
