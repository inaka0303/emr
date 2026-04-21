import { useState, useCallback, useMemo, useEffect } from 'react';
import Sidebar from './Sidebar';
import BottomNav from './BottomNav';
import MainContent from './MainContent';
import PatientList from '../../pages/PatientList';
import SOAPEditor from '../chart/SOAPEditor';
import SOAPHistory from '../chart/SOAPHistory';
import PatientDetail from '../patient/PatientDetail';
import InterviewViewer from '../slm/InterviewViewer';
import AdmissionSummaryDrafter from '../slm/AdmissionSummaryDrafter';
import { usePatients } from '../../hooks/usePatients';
import { usePatientData } from '../../hooks/usePatientData';
import { useEncounterInterview } from '../../hooks/useEncounterInterview';
import { useSoapDraftCache, type SoapDraftEntry } from '../../hooks/useSoapDraftCache';
import type { Patient, MainTab, MobileTab, Encounter } from '../../types/api';

/**
 * 対象エンカウンターの選び方:
 *  1) status が '進行中' のものを優先
 *  2) それもなければ encounter_date が最新
 */
function selectActiveEncounter(encounters: Encounter[]): Encounter | null {
  if (encounters.length === 0) return null;
  const inProgress = encounters.find((e) => e.status === '進行中');
  if (inProgress) return inProgress;
  return [...encounters].sort(
    (a, b) => new Date(b.encounter_date).getTime() - new Date(a.encounter_date).getTime(),
  )[0];
}

export default function Layout() {
  const [selectedPatient, setSelectedPatient] = useState<Patient | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeMainTab, setActiveMainTab] = useState<MainTab>('chart');
  const [activeMobileTab, setActiveMobileTab] = useState<MobileTab>('patients');
  // encounterId set: 問診入力モードの患者で医師が「確定」を押したもの
  const [finalizedEncounters, setFinalizedEncounters] = useState<Set<number>>(new Set());

  // ページロード時に、デモ用テスト患者 (MRN-0021, 新規太郎) の interview と SOAP ドラフトをリセット
  // これにより、ブラウザリロード毎に問診入力テストを何度でも繰り返せる
  useEffect(() => {
    fetch('/api/test-patient/reset', { method: 'POST' }).catch(() => {
      /* ignore: demo-only endpoint */
    });
  }, []);

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

  const activeEncounter = useMemo(() => selectActiveEncounter(patientEncounters), [patientEncounters]);
  const activeEncounterId = activeEncounter?.id ?? null;
  const { notes: interviewNotes, combinedText: interviewText, isLoading: interviewLoading, error: interviewError } =
    useEncounterInterview(activeEncounterId);

  // activeEncounterにSOAPが既に存在するかチェック（auto-draft抑止用）
  const hasExistingSOAPForActive = useMemo(() => {
    if (!activeEncounter) return false;
    return soapNotes.some((n) => n.encounter_id === activeEncounter.id);
  }, [soapNotes, activeEncounter]);

  // SOAPドラフトキャッシュ: 患者/受診選択と同時に先行生成し、カルテ表示で即反映
  // バックエンドの DB キャッシュと二段構え: バックエンドは encounter_id 単位で永続化済。
  const soapDraftCache = useSoapDraftCache();
  const { prefetch: prefetchSoapDraft, invalidate: invalidateSoapDraft } = soapDraftCache;
  useEffect(() => {
    if (!activeEncounterId) return;
    if (hasExistingSOAPForActive) return;
    // 問診が未入力の encounter は SOAP 生成しない（バックエンドも400を返す）
    if (interviewNotes.length === 0) return;
    prefetchSoapDraft(activeEncounterId);
    // prefetchSoapDraft は useCallback で安定しているため、soapDraftCache 全体を依存に含めない
    // （含めると親の再レンダー毎に fetch が再発火する）
  }, [activeEncounterId, hasExistingSOAPForActive, prefetchSoapDraft, interviewNotes.length]);
  const soapDraftEntry = soapDraftCache.get(activeEncounterId);

  // 問診が入力／更新された時: DB キャッシュを invalidate し、新しい問診で再生成
  const handleInterviewUpdated = useCallback(
    (encId: number) => {
      invalidateSoapDraft(encId);
      // force=true で DB キャッシュも無視して再生成
      prefetchSoapDraft(encId, true);
    },
    [invalidateSoapDraft, prefetchSoapDraft],
  );

  // 「確定」ボタン押下: SOAPエリアの表示を解禁する（裏で生成された結果がそのまま見える）
  const handleFinalize = useCallback(
    (encId: number) => {
      setFinalizedEncounters((prev) => {
        if (prev.has(encId)) return prev;
        const next = new Set(prev);
        next.add(encId);
        return next;
      });
      // 念のため prefetch を促す（既に走っていれば no-op）
      prefetchSoapDraft(encId);
    },
    [prefetchSoapDraft],
  );

  const handleSelectPatient = useCallback((patient: Patient) => {
    setSelectedPatient(patient);
    setActiveMobileTab('chart');
  }, []);

  const apiError = patientsError || patientDataError;

  // カルテ画面（問診 + SOAP + (入院時サマリ) + 履歴）
  const renderChartView = () => {
    if (!selectedPatient) {
      return (
        <div className="flex items-center justify-center h-full">
          <p className="text-text-muted">左のリストから患者を選択してください</p>
        </div>
      );
    }
    if (patientDataLoading) return <LoadingSpinner />;

    if (!activeEncounter) {
      return (
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center">
          <p className="text-sm text-gray-500">この患者には受診記録がありません</p>
        </div>
      );
    }

    return (
      <div className="space-y-6">
        {/* 受診ヘッダ */}
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm px-4 py-3">
          <div className="flex items-center justify-between flex-wrap gap-2">
            <div>
              <div className="text-xs text-gray-400">対象受診</div>
              <div className="text-sm font-medium text-gray-800">
                {activeEncounter.encounter_date} / {activeEncounter.encounter_type} / {activeEncounter.department} /{' '}
                Dr. {activeEncounter.attending_doctor}
              </div>
              <div className="text-xs text-gray-600 mt-0.5">主訴: {activeEncounter.chief_complaint}</div>
            </div>
            <span
              className={`px-2 py-0.5 text-xs rounded-full ${
                activeEncounter.status === '進行中'
                  ? 'bg-amber-50 text-amber-700 border border-amber-200'
                  : 'bg-gray-50 text-gray-600 border border-gray-200'
              }`}
            >
              {activeEncounter.status}
            </span>
          </div>
        </div>

        {/* 問診（左）+ SOAP（右）の2カラム */}
        <div className="grid gap-6 lg:grid-cols-[minmax(280px,360px)_1fr]">
          <div className="min-h-[400px]">
            <InterviewViewer
              encounterId={activeEncounter.id}
              notes={interviewNotes}
              isLoading={interviewLoading}
              error={interviewError}
              onInterviewUpdated={handleInterviewUpdated}
              onFinalize={handleFinalize}
              finalized={finalizedEncounters.has(activeEncounter.id)}
            />
          </div>
          <div className="space-y-6">
            {interviewNotes.length > 0 || finalizedEncounters.has(activeEncounter.id) ? (
              <SOAPEditor
                patientName={selectedPatient.name}
                patientId={selectedPatient.id}
                encounterId={activeEncounter.id}
                interviewText={interviewText}
                hasExistingSOAP={hasExistingSOAPForActive}
                draftEntry={soapDraftEntry}
              />
            ) : (
              <SoapPlaceholder draftEntry={soapDraftEntry} />
            )}
            {activeEncounter.encounter_type === '入院' && interviewText && (
              <AdmissionSummaryDrafter encounterId={activeEncounter.id} interviewText={interviewText} />
            )}
          </div>
        </div>

        {/* 過去SOAP履歴 */}
        <SOAPHistory encounters={patientEncounters} soapNotes={soapNotes} />
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
        return renderChartView();
      case 'settings':
        return (
          <div>
            <h2 className="text-lg font-bold mb-4">設定</h2>
            <div className="bg-white rounded-lg p-4 border border-gray-200">
              <div className="flex items-center justify-between py-2">
                <span className="text-sm">SLM API URL</span>
                <span className="text-xs text-text-muted font-mono">localhost:8081</span>
              </div>
              <div className="flex items-center justify-between py-2 border-t border-gray-100">
                <span className="text-sm">モデル</span>
                <span className="text-xs text-text-muted">qwen3.5-4b-medical</span>
              </div>
            </div>
          </div>
        );
    }
  };

  // PC版コンテンツ
  const renderDesktopContent = () => {
    switch (activeMainTab) {
      case 'chart':
        return renderChartView();
      case 'history':
        if (!selectedPatient) {
          return <p className="text-text-muted">左のリストから患者を選択してください</p>;
        }
        return <SOAPHistory encounters={patientEncounters} soapNotes={soapNotes} />;
      case 'patient-info':
        if (!selectedPatient) {
          return <p className="text-text-muted">左のリストから患者を選択してください</p>;
        }
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
      {apiError && (
        <div className="fixed top-0 left-0 right-0 z-50 bg-amber-50 border-b border-amber-200 px-4 py-2 text-center">
          <span className="text-xs text-amber-700">{apiError}</span>
        </div>
      )}

      <Sidebar
        patients={filteredPatients}
        selectedPatientId={selectedPatient?.id ?? null}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onSelectPatient={handleSelectPatient}
        isLoading={patientsLoading}
      />

      <div className={`flex-1 flex ${apiError ? 'pt-8' : ''}`}>
        <div className="lg:hidden flex-1 overflow-y-auto p-4 pb-20">
          {renderMobileContent()}
        </div>

        <div className="hidden lg:flex lg:flex-1">
          <MainContent activeTab={activeMainTab} onTabChange={setActiveMainTab}>
            {renderDesktopContent()}
          </MainContent>
        </div>
      </div>

      <BottomNav activeTab={activeMobileTab} onTabChange={setActiveMobileTab} />
    </div>
  );
}

function SoapPlaceholder({ draftEntry }: { draftEntry: SoapDraftEntry | null }) {
  let hint: React.ReactNode = '問診を入力後「確定」ボタンでSOAPドラフトが表示されます。';
  if (draftEntry?.isLoading) {
    hint = (
      <>
        バックグラウンドで SOAP をスタンバイ生成中... <span className="text-violet-500">確定時に即表示されます</span>
      </>
    );
  } else if (draftEntry?.done && !draftEntry?.error) {
    hint = <span className="text-emerald-600">✓ SOAPドラフト生成完了（確定ボタンで表示）</span>;
  } else if (draftEntry?.error) {
    hint = <span className="text-red-600">生成エラー: {draftEntry.error}</span>;
  }
  return (
    <div className="bg-gray-50 rounded-lg border border-dashed border-gray-300 p-8 text-center">
      <svg className="w-10 h-10 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
          d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      <p className="text-sm text-gray-700">{hint}</p>
    </div>
  );
}

function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center py-12">
      <div className="flex flex-col items-center gap-3">
        <svg className="animate-spin h-8 w-8 text-primary-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        <span className="text-sm text-text-muted">読み込み中...</span>
      </div>
    </div>
  );
}
