import { useState, useEffect, useCallback } from 'react';
import { get } from '../api/client';
import type {
  Encounter,
  SOAPNote,
  MedicalHistory,
  FamilyHistory,
  SocialHistory,
  ApiResponse,
} from '../types/api';
import {
  MOCK_ENCOUNTERS,
  MOCK_SOAP_NOTES,
  MOCK_MEDICAL_HISTORY,
  MOCK_FAMILY_HISTORY,
  MOCK_SOCIAL_HISTORY,
} from '../utils/mockData';

interface RefetchOptions {
  silent?: boolean;
}

interface UsePatientDataResult {
  encounters: Encounter[];
  soapNotes: SOAPNote[];
  medicalHistories: MedicalHistory[];
  familyHistories: FamilyHistory[];
  socialHistories: SocialHistory[];
  isLoading: boolean;
  error: string | null;
  refetch: (options?: RefetchOptions) => void;
}

interface UsePatientDataOptions {
  allowMockFallback?: boolean;
}

export function usePatientData(patientId: number | null, options: UsePatientDataOptions = {}): UsePatientDataResult {
  const { allowMockFallback = true } = options;
  const [encounters, setEncounters] = useState<Encounter[]>([]);
  const [soapNotes, setSoapNotes] = useState<SOAPNote[]>([]);
  const [medicalHistories, setMedicalHistories] = useState<MedicalHistory[]>([]);
  const [familyHistories, setFamilyHistories] = useState<FamilyHistory[]>([]);
  const [socialHistories, setSocialHistories] = useState<SocialHistory[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async (pid: number, fetchOptions: RefetchOptions = {}) => {
    if (!fetchOptions.silent) {
      setIsLoading(true);
    }
    setError(null);

    try {
      // 並列リクエスト: encounters, medical/family/social history
      const [encountersRes, medicalRes, familyRes, socialRes] = await Promise.all([
        get<ApiResponse<Encounter[]>>(`/patients/${pid}/encounters`),
        get<ApiResponse<MedicalHistory[]>>(`/patients/${pid}/medical-history`),
        get<ApiResponse<FamilyHistory[]>>(`/patients/${pid}/family-history`),
        get<ApiResponse<SocialHistory[]>>(`/patients/${pid}/social-history`),
      ]);

      const fetchedEncounters = encountersRes.data ?? [];
      setEncounters(fetchedEncounters);
      setMedicalHistories(medicalRes.data ?? []);
      setFamilyHistories(familyRes.data ?? []);
      setSocialHistories(socialRes.data ?? []);

      // 各 encounter の SOAP ノートを取得
      if (fetchedEncounters.length > 0) {
        const soapResults = await Promise.all(
          fetchedEncounters.map((enc) =>
            get<ApiResponse<SOAPNote[]>>(`/encounters/${enc.id}/soap`).catch(() => ({
              data: [] as SOAPNote[],
            })),
          ),
        );
        const allSoapNotes = soapResults.flatMap((res) => res.data ?? []);
        setSoapNotes(allSoapNotes);
      } else {
        setSoapNotes([]);
      }

      setError(null);
    } catch (err) {
      if (!allowMockFallback) {
        setError('APIに接続できません。実験モードではモックデータを表示しません。');
        setEncounters([]);
        setSoapNotes([]);
        setMedicalHistories([]);
        setFamilyHistories([]);
        setSocialHistories([]);
        return;
      }

      console.warn('API接続失敗、モックデータにフォールバック:', err);
      setError('APIに接続できません。モックデータを表示しています。');

      // フォールバック: モックデータからフィルタ
      setEncounters(MOCK_ENCOUNTERS.filter((e) => e.patient_id === pid));
      setSoapNotes(
        MOCK_SOAP_NOTES.filter((s) =>
          MOCK_ENCOUNTERS.some(
            (e) => e.patient_id === pid && e.id === s.encounter_id,
          ),
        ),
      );
      setMedicalHistories(MOCK_MEDICAL_HISTORY.filter((h) => h.patient_id === pid));
      setFamilyHistories(MOCK_FAMILY_HISTORY.filter((h) => h.patient_id === pid));
      setSocialHistories(MOCK_SOCIAL_HISTORY.filter((h) => h.patient_id === pid));
    } finally {
      if (!fetchOptions.silent) {
        setIsLoading(false);
      }
    }
  }, [allowMockFallback]);

  useEffect(() => {
    if (patientId === null) {
      setEncounters([]);
      setSoapNotes([]);
      setMedicalHistories([]);
      setFamilyHistories([]);
      setSocialHistories([]);
      setError(null);
      return;
    }
    fetchData(patientId);
  }, [patientId, fetchData]);

  const refetch = useCallback((refetchOptions: RefetchOptions = {}) => {
    if (patientId !== null) {
      fetchData(patientId, refetchOptions);
    }
  }, [patientId, fetchData]);

  return {
    encounters,
    soapNotes,
    medicalHistories,
    familyHistories,
    socialHistories,
    isLoading,
    error,
    refetch,
  };
}
