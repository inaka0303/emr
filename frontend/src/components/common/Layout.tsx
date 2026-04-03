import { useState, useMemo, useCallback } from 'react';
import Sidebar from './Sidebar';
import BottomNav from './BottomNav';
import MainContent from './MainContent';
import SuggestionPanel from './SuggestionPanel';
import PatientList from '../../pages/PatientList';
import SOAPEditor from '../chart/SOAPEditor';
import SOAPHistory from '../chart/SOAPHistory';
import PatientDetail from '../patient/PatientDetail';
import InterviewInput from '../slm/InterviewInput';
import SOAPSuggest from '../slm/SOAPSuggest';
import SummarySuggest from '../slm/SummarySuggest';
import {
  useSLMSuggestion,
  useSLMSummary,
} from '../../hooks/useSLMSuggestion';
import { post } from '../../api/client';
import type { Patient, MainTab, MobileTab, SLMSoapSuggestion } from '../../types/api';
import {
  MOCK_PATIENTS,
  MOCK_ENCOUNTERS,
  MOCK_SOAP_NOTES,
  MOCK_MEDICAL_HISTORY,
  MOCK_FAMILY_HISTORY,
  MOCK_SOCIAL_HISTORY,
} from '../../utils/mockData';

export default function Layout() {
  const [selectedPatient, setSelectedPatient] = useState<Patient | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeMainTab, setActiveMainTab] = useState<MainTab>('chart');
  const [activeMobileTab, setActiveMobileTab] = useState<MobileTab>('patients');

  // SOAP フィールド値（SOAPSuggest と共有）
  const [soapFieldValues, setSoapFieldValues] = useState<
    Record<keyof SLMSoapSuggestion, string>
  >({
    subjective: '',
    objective: '',
    assessment: '',
    plan: '',
  });
  const [saveStatus, setSaveStatus] = useState<
    'idle' | 'saving' | 'success' | 'error'
  >('idle');
  const [saveError, setSaveError] = useState('');

  // SLM hooks
  const soapSuggestion = useSLMSuggestion();
  const summary = useSLMSummary();

  const filteredPatients = useMemo(() => {
    if (!searchQuery.trim()) return MOCK_PATIENTS;
    const q = searchQuery.toLowerCase();
    return MOCK_PATIENTS.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.name_kana.toLowerCase().includes(q) ||
        p.mrn.toLowerCase().includes(q),
    );
  }, [searchQuery]);

  const patientEncounters = useMemo(() => {
    if (!selectedPatient) return [];
    return MOCK_ENCOUNTERS.filter((e) => e.patient_id === selectedPatient.id);
  }, [selectedPatient]);

  const handleSelectPatient = useCallback((patient: Patient) => {
    setSelectedPatient(patient);
    setActiveMobileTab('chart');
  }, []);

  const handleSoapFieldChange = useCallback(
    (field: keyof SLMSoapSuggestion, value: string) => {
      setSoapFieldValues((prev) => ({ ...prev, [field]: value }));
    },
    [],
  );

  const handleSoapSave = useCallback(async () => {
    const encounterId =
      patientEncounters.length > 0
        ? patientEncounters[patientEncounters.length - 1].id
        : undefined;
    if (!encounterId) {
      setSaveStatus('error');
      setSaveError('受診を選択してください');
      return;
    }
    setSaveStatus('saving');
    setSaveError('');
    try {
      await post(`/encounters/${encounterId}/soap`, soapFieldValues);
      setSaveStatus('success');
      // React 18 自動 batching: 同一ハンドラ内の setState は一括処理される
      // reset() を先に呼び、提案状態をクリアしてからフィールド値をリセット
      soapSuggestion.reset();
      setSoapFieldValues({
        subjective: '',
        objective: '',
        assessment: '',
        plan: '',
      });
      setTimeout(() => setSaveStatus('idle'), 3000);
    } catch (err) {
      setSaveStatus('error');
      setSaveError(
        err instanceof Error ? err.message : '保存に失敗しました',
      );
    }
  }, [soapFieldValues, patientEncounters, soapSuggestion]);

  const handleRequestSOAP = useCallback(
    (text: string) => {
      // フィールドをリセットしてからリクエスト
      setSoapFieldValues({
        subjective: '',
        objective: '',
        assessment: '',
        plan: '',
      });
      soapSuggestion.requestSOAP(text);
    },
    [soapSuggestion],
  );

  const handleRequestFamilyHistory = useCallback(
    (text: string) => {
      summary.requestSummary(text, 'family_history');
    },
    [summary],
  );

  const handleRequestSocialHistory = useCallback(
    (text: string) => {
      summary.requestSummary(text, 'social_history');
    },
    [summary],
  );

  const showSuggestionPanel = activeMainTab === 'interview';

  // 問診コンテンツ（PC/スマホ共通）
  const renderInterviewContent = (isMobile: boolean) => {
    if (!selectedPatient) {
      return (
        <p className="text-sm text-text-muted text-center mt-8">
          患者を選択してください
        </p>
      );
    }

    return (
      <div className="space-y-6">
        <InterviewInput
          onRequestSOAP={handleRequestSOAP}
          onRequestFamilyHistory={handleRequestFamilyHistory}
          onRequestSocialHistory={handleRequestSocialHistory}
          isLoadingSOAP={soapSuggestion.state.isLoading}
          isLoadingSummary={summary.state.isLoading}
          patientName={selectedPatient.name}
        />

        {/* SOAP提案エディタ（提案あり or ローディング中に表示） */}
        {(soapSuggestion.state.suggestions ||
          soapSuggestion.state.isLoading ||
          soapSuggestion.state.error) && (
          <SOAPSuggest
            suggestionState={soapSuggestion.state}
            fieldValues={soapFieldValues}
            onFieldChange={handleSoapFieldChange}
            onAccept={soapSuggestion.acceptField}
            onDismiss={soapSuggestion.dismissField}
            onMoveNext={soapSuggestion.moveToNextField}
            onSetActiveField={soapSuggestion.setActiveField}
            onSave={handleSoapSave}
            saveStatus={saveStatus}
            errorMessage={saveError}
            patientName={selectedPatient.name}
            encounterId={
              patientEncounters.length > 0
                ? patientEncounters[patientEncounters.length - 1].id
                : undefined
            }
          />
        )}

        {/* スマホ版では要約提案もインラインで表示 */}
        {isMobile && (
          <SummarySuggest
            summaryState={summary.state}
            onAcceptItem={summary.acceptItem}
            onDismissItem={summary.dismissItem}
            onAcceptAll={summary.acceptAll}
          />
        )}
      </div>
    );
  };

  // スマホ版コンテンツ
  const renderMobileContent = () => {
    switch (activeMobileTab) {
      case 'patients':
        return (
          <PatientList
            patients={filteredPatients}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            onSelectPatient={handleSelectPatient}
          />
        );
      case 'chart':
        return selectedPatient ? (
          <div className="space-y-6">
            <SOAPEditor
              patientName={selectedPatient.name}
              encounterId={
                patientEncounters.length > 0
                  ? patientEncounters[patientEncounters.length - 1].id
                  : undefined
              }
            />
            <SOAPHistory
              encounters={patientEncounters}
              soapNotes={MOCK_SOAP_NOTES}
            />
          </div>
        ) : (
          <p className="text-sm text-text-muted text-center mt-8">
            患者を選択してください
          </p>
        );
      case 'interview':
        return renderInterviewContent(true);
      case 'settings':
        return (
          <div>
            <h2 className="text-lg font-bold mb-4">設定</h2>
            <div className="bg-white rounded-lg p-4 border border-gray-200">
              <div className="flex items-center justify-between py-2">
                <span className="text-sm">SLM API URL</span>
                <span className="text-xs text-text-muted font-mono">
                  localhost:8000
                </span>
              </div>
              <div className="flex items-center justify-between py-2 border-t border-gray-100">
                <span className="text-sm">モデル</span>
                <span className="text-xs text-text-muted">
                  qwen3.5-0.8b-medical
                </span>
              </div>
            </div>
          </div>
        );
    }
  };

  // PC版コンテンツ
  const renderDesktopContent = () => {
    if (!selectedPatient) {
      return (
        <div className="flex items-center justify-center h-full">
          <div className="text-center">
            <svg
              className="w-16 h-16 mx-auto text-text-muted mb-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
              />
            </svg>
            <p className="text-text-muted">
              左のリストから患者を選択してください
            </p>
          </div>
        </div>
      );
    }

    switch (activeMainTab) {
      case 'chart':
        return (
          <div className="space-y-6">
            <SOAPEditor
              patientName={selectedPatient.name}
              encounterId={
                patientEncounters.length > 0
                  ? patientEncounters[patientEncounters.length - 1].id
                  : undefined
              }
            />
            <SOAPHistory
              encounters={patientEncounters}
              soapNotes={MOCK_SOAP_NOTES}
            />
          </div>
        );
      case 'interview':
        return renderInterviewContent(false);
      case 'history':
        return (
          <SOAPHistory
            encounters={patientEncounters}
            soapNotes={MOCK_SOAP_NOTES}
          />
        );
      case 'patient-info':
        return (
          <PatientDetail
            patient={selectedPatient}
            medicalHistories={MOCK_MEDICAL_HISTORY}
            familyHistories={MOCK_FAMILY_HISTORY}
            socialHistories={MOCK_SOCIAL_HISTORY}
          />
        );
    }
  };

  return (
    <div className="flex h-screen bg-surface">
      {/* PC版: サイドバー */}
      <Sidebar
        patients={filteredPatients}
        selectedPatientId={selectedPatient?.id ?? null}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onSelectPatient={handleSelectPatient}
      />

      {/* メインコンテンツ */}
      <div className="flex-1 flex">
        {/* スマホ版 */}
        <div className="lg:hidden flex-1 overflow-y-auto p-4 pb-20">
          {renderMobileContent()}
        </div>

        {/* PC版 */}
        <div className="hidden lg:flex lg:flex-1">
          <MainContent
            activeTab={activeMainTab}
            onTabChange={setActiveMainTab}
          >
            {renderDesktopContent()}
          </MainContent>

          <SuggestionPanel
            visible={showSuggestionPanel}
            soapState={soapSuggestion.state}
            summaryState={summary.state}
          />
        </div>
      </div>

      {/* スマホ版: ボトムナビ */}
      <BottomNav
        activeTab={activeMobileTab}
        onTabChange={setActiveMobileTab}
      />
    </div>
  );
}
