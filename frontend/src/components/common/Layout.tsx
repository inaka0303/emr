import { useState, useCallback } from 'react';
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
import { usePatients } from '../../hooks/usePatients';
import { usePatientData } from '../../hooks/usePatientData';
import { post } from '../../api/client';
import type { Patient, MainTab, MobileTab, SLMSoapSuggestion } from '../../types/api';

export default function Layout() {
  const [selectedPatient, setSelectedPatient] = useState<Patient | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeMainTab, setActiveMainTab] = useState<MainTab>('chart');
  const [activeMobileTab, setActiveMobileTab] = useState<MobileTab>('patients');

  // API hooks
  const { patients: filteredPatients, isLoading: patientsLoading, error: patientsError } = usePatients(searchQuery);
  const {
    encounters: patientEncounters,
    soapNotes,
    medicalHistories,
    familyHistories,
    socialHistories,
    isLoading: patientDataLoading,
    error: patientDataError,
  } = usePatientData(selectedPatient?.id ?? null);

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
        ? patientEncounters[0].id
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

  // API エラーバナー
  const apiError = patientsError || patientDataError;

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
                ? patientEncounters[0].id
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

  // ローディングスピナー
  const renderLoadingSpinner = () => (
    <div className="flex items-center justify-center py-12">
      <div className="flex flex-col items-center gap-3">
        <svg
          className="animate-spin h-8 w-8 text-primary-500"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
        <span className="text-sm text-text-muted">読み込み中...</span>
      </div>
    </div>
  );

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
        if (patientDataLoading && selectedPatient) return renderLoadingSpinner();
        return selectedPatient ? (
          <div className="space-y-6">
            <SOAPEditor
              patientName={selectedPatient.name}
              patientId={selectedPatient.id}
              encounterId={
                patientEncounters.length > 0
                  ? patientEncounters[0].id
                  : undefined
              }
            />
            <SOAPHistory
              encounters={patientEncounters}
              soapNotes={soapNotes}
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

    if (patientDataLoading) {
      return renderLoadingSpinner();
    }

    switch (activeMainTab) {
      case 'chart':
        return (
          <div className="space-y-6">
            <SOAPEditor
              patientName={selectedPatient.name}
              patientId={selectedPatient.id}
              encounterId={
                patientEncounters.length > 0
                  ? patientEncounters[0].id
                  : undefined
              }
            />
            <SOAPHistory
              encounters={patientEncounters}
              soapNotes={soapNotes}
            />
          </div>
        );
      case 'interview':
        return renderInterviewContent(false);
      case 'history':
        return (
          <SOAPHistory
            encounters={patientEncounters}
            soapNotes={soapNotes}
          />
        );
      case 'patient-info':
        return (
          <PatientDetail
            patient={selectedPatient}
            medicalHistories={medicalHistories}
            familyHistories={familyHistories}
            socialHistories={socialHistories}
          />
        );
    }
  };

  return (
    <div className="flex h-screen bg-surface">
      {/* API エラーバナー */}
      {apiError && (
        <div className="fixed top-0 left-0 right-0 z-50 bg-amber-50 border-b border-amber-200 px-4 py-2 text-center">
          <span className="text-xs text-amber-700">{apiError}</span>
        </div>
      )}

      {/* PC版: サイドバー */}
      <Sidebar
        patients={filteredPatients}
        selectedPatientId={selectedPatient?.id ?? null}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onSelectPatient={handleSelectPatient}
        isLoading={patientsLoading}
      />

      {/* メインコンテンツ */}
      <div className={`flex-1 flex ${apiError ? 'pt-8' : ''}`}>
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
