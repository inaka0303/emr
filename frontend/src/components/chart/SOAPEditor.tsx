import { useState, useCallback } from 'react';
import { post } from '../../api/client';
import InlineCompletionTextarea from '../slm/InlineCompletionTextarea';

interface SOAPData {
  subjective: string;
  objective: string;
  assessment: string;
  plan: string;
}

interface SOAPEditorProps {
  patientName: string;
  encounterId?: number;
  patientId?: number;
}

interface SectionConfig {
  key: keyof SOAPData;
  label: string;
  letter: string;
  description: string;
  context: string;
  borderColor: string;
  labelColor: string;
}

const SECTIONS: SectionConfig[] = [
  {
    key: 'subjective',
    label: '主観的所見',
    letter: 'S',
    description: '患者の訴え、自覚症状',
    context: 'soap_subjective',
    borderColor: 'border-blue-400',
    labelColor: 'text-blue-700 bg-blue-50',
  },
  {
    key: 'objective',
    label: '客観的所見',
    letter: 'O',
    description: '検査結果、バイタル、身体所見',
    context: 'soap_objective',
    borderColor: 'border-emerald-400',
    labelColor: 'text-emerald-700 bg-emerald-50',
  },
  {
    key: 'assessment',
    label: '評価',
    letter: 'A',
    description: '診断、アセスメント',
    context: 'soap_assessment',
    borderColor: 'border-amber-400',
    labelColor: 'text-amber-700 bg-amber-50',
  },
  {
    key: 'plan',
    label: '計画',
    letter: 'P',
    description: '治療計画、処方、次回予約',
    context: 'soap_plan',
    borderColor: 'border-purple-400',
    labelColor: 'text-purple-700 bg-purple-50',
  },
];

export default function SOAPEditor({ patientName, encounterId, patientId }: SOAPEditorProps) {
  const [data, setData] = useState<SOAPData>({
    subjective: '',
    objective: '',
    assessment: '',
    plan: '',
  });
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');

  const handleChange = useCallback((key: keyof SOAPData, value: string) => {
    setData((prev) => ({ ...prev, [key]: value }));
  }, []);

  const handleSave = useCallback(async () => {
    if (!encounterId) {
      setSaveStatus('error');
      setErrorMessage('受診を選択してください');
      return;
    }

    setSaveStatus('saving');
    setErrorMessage('');
    try {
      await post(`/encounters/${encounterId}/soap`, data);
      setSaveStatus('success');
      setData({ subjective: '', objective: '', assessment: '', plan: '' });
      setTimeout(() => setSaveStatus('idle'), 3000);
    } catch (err) {
      setSaveStatus('error');
      setErrorMessage(err instanceof Error ? err.message : '保存に失敗しました');
    }
  }, [data, encounterId]);

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
      <div className="px-4 py-3 border-b border-gray-100">
        <h3 className="text-base font-semibold text-gray-800">
          新規カルテ記録
        </h3>
        <p className="text-xs text-gray-500 mt-0.5">
          {patientName} - SOAP形式
          <span className="ml-2 text-violet-500">SLMインライン補完有効</span>
        </p>
      </div>

      <div className="p-4 space-y-4">
        {SECTIONS.map((section) => (
          <div key={section.key} className={`border-l-4 ${section.borderColor} pl-3`}>
            <div className="flex items-center gap-2 mb-1.5">
              <span
                className={`inline-flex items-center justify-center w-6 h-6 rounded text-xs font-bold ${section.labelColor}`}
              >
                {section.letter}
              </span>
              <span className="text-sm font-medium text-gray-700">
                {section.label}
              </span>
              <span className="text-xs text-gray-400">
                {section.description}
              </span>
            </div>
            <InlineCompletionTextarea
              value={data[section.key]}
              onChange={(val) => handleChange(section.key, val)}
              context={section.context}
              patientId={patientId}
              placeholder={`${section.label}を入力...`}
              rows={3}
            />
          </div>
        ))}
      </div>

      <div className="px-4 py-3 border-t border-gray-100 flex items-center justify-between">
        <div className="text-sm">
          {saveStatus === 'success' && (
            <span className="text-emerald-600">保存しました</span>
          )}
          {saveStatus === 'error' && (
            <span className="text-red-600">{errorMessage}</span>
          )}
        </div>
        <button
          onClick={handleSave}
          disabled={saveStatus === 'saving'}
          className="px-5 py-2 bg-primary-600 text-white text-sm font-medium rounded-lg
            hover:bg-primary-700 transition-colors focus:outline-none focus:ring-2
            focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50"
        >
          {saveStatus === 'saving' ? '保存中...' : '保存'}
        </button>
      </div>
    </div>
  );
}
