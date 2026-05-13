import { useState, useEffect, useCallback, useRef } from 'react';
import { get } from '../api/client';
import type { Patient, ApiResponse } from '../types/api';
import { MOCK_PATIENTS } from '../utils/mockData';

interface UsePatientsResult {
  patients: Patient[];
  isLoading: boolean;
  error: string | null;
  refetch: () => void;
}

interface UsePatientsOptions {
  allowMockFallback?: boolean;
}

export function usePatients(query: string, options: UsePatientsOptions = {}): UsePatientsResult {
  const { allowMockFallback = true } = options;
  const [patients, setPatients] = useState<Patient[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const fetchPatients = useCallback(async (searchQuery: string) => {
    // 前のリクエストをキャンセル
    if (abortRef.current) {
      abortRef.current.abort();
    }
    abortRef.current = new AbortController();

    setIsLoading(true);
    setError(null);

    try {
      const params = searchQuery.trim() ? `?q=${encodeURIComponent(searchQuery)}` : '';
      const response = await get<ApiResponse<Patient[]>>(`/patients${params}`);
      setPatients(response.data ?? []);
      setError(null);
    } catch (err) {
      // AbortError は無視
      if (err instanceof DOMException && err.name === 'AbortError') return;

      if (!allowMockFallback) {
        setError('APIに接続できません。実験モードではモックデータを表示しません。');
        setPatients([]);
        return;
      }

      console.warn('API接続失敗、モックデータにフォールバック:', err);
      setError('APIに接続できません。モックデータを表示しています。');

      // フォールバック: モックデータで検索
      if (!searchQuery.trim()) {
        setPatients(MOCK_PATIENTS);
      } else {
        const q = searchQuery.toLowerCase();
        setPatients(
          MOCK_PATIENTS.filter(
            (p) =>
              p.name.toLowerCase().includes(q) ||
              p.name_kana.toLowerCase().includes(q) ||
              p.mrn.toLowerCase().includes(q),
          ),
        );
      }
    } finally {
      setIsLoading(false);
    }
  }, [allowMockFallback]);

  useEffect(() => {
    // デバウンス: 300ms
    const timer = setTimeout(() => {
      fetchPatients(query);
    }, 300);
    return () => clearTimeout(timer);
  }, [query, fetchPatients]);

  const refetch = useCallback(() => {
    fetchPatients(query);
  }, [query, fetchPatients]);

  return { patients, isLoading, error, refetch };
}
