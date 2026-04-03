import { useState, useMemo } from 'react';
import type { Encounter, SOAPNote } from '../../types/api';

interface SOAPHistoryProps {
  encounters: Encounter[];
  soapNotes: SOAPNote[];
}

const ENCOUNTER_TYPE_STYLES: Record<string, string> = {
  '外来': 'bg-blue-50 text-blue-700',
  '入院': 'bg-amber-50 text-amber-700',
  '救急': 'bg-red-50 text-red-700',
};

const SOAP_SECTION_STYLES = {
  S: { label: '主観的所見', borderColor: 'border-blue-400', labelColor: 'text-blue-700 bg-blue-50' },
  O: { label: '客観的所見', borderColor: 'border-emerald-400', labelColor: 'text-emerald-700 bg-emerald-50' },
  A: { label: '評価', borderColor: 'border-amber-400', labelColor: 'text-amber-700 bg-amber-50' },
  P: { label: '計画', borderColor: 'border-purple-400', labelColor: 'text-purple-700 bg-purple-50' },
};

export default function SOAPHistory({ encounters, soapNotes }: SOAPHistoryProps) {
  const [expandedId, setExpandedId] = useState<number | null>(null);

  // encounter_idでSOAPNoteをマッピング（1:N対応）
  const notesByEncounterId = useMemo(() => {
    const map = new Map<number, SOAPNote[]>();
    for (const note of soapNotes) {
      const existing = map.get(note.encounter_id);
      if (existing) {
        existing.push(note);
      } else {
        map.set(note.encounter_id, [note]);
      }
    }
    return map;
  }, [soapNotes]);

  // 新しい順にソート
  const sortedEncounters = useMemo(() => {
    return [...encounters].sort(
      (a, b) => new Date(b.encounter_date).getTime() - new Date(a.encounter_date).getTime(),
    );
  }, [encounters]);

  if (sortedEncounters.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-8 text-center">
        <p className="text-sm text-gray-500">受診履歴はありません</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <h3 className="text-base font-semibold text-gray-800">カルテ履歴</h3>
      {sortedEncounters.map((encounter) => {
        const notes = notesByEncounterId.get(encounter.id);
        const isExpanded = expandedId === encounter.id;
        const hasSlmSuggested = notes?.some((n) => n.is_slm_suggested) ?? false;

        return (
          <div
            key={encounter.id}
            className="bg-white rounded-lg border border-gray-200 shadow-sm hover:shadow-md transition-shadow"
          >
            <button
              onClick={() => setExpandedId(isExpanded ? null : encounter.id)}
              className="w-full text-left px-4 py-3 focus:outline-none"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="text-sm font-medium text-gray-700">
                    {encounter.encounter_date}
                  </span>
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                      ENCOUNTER_TYPE_STYLES[encounter.encounter_type] ?? 'bg-gray-50 text-gray-700'
                    }`}
                  >
                    {encounter.encounter_type}
                  </span>
                  {hasSlmSuggested && (
                    <span className="text-xs px-2 py-0.5 rounded-full bg-violet-50 text-violet-700 font-medium">
                      SLM提案
                    </span>
                  )}
                </div>
                <svg
                  className={`w-4 h-4 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>
              <div className="flex items-center gap-2 mt-1.5">
                <span className="text-xs text-gray-500">{encounter.department}</span>
                <span className="text-xs text-gray-300">|</span>
                <span className="text-xs text-gray-500">{encounter.attending_doctor}</span>
              </div>
              <p className="text-sm text-gray-600 mt-1">{encounter.chief_complaint}</p>
            </button>

            {isExpanded && (
              <div className="px-4 pb-4 pt-1 border-t border-gray-100">
                {notes && notes.length > 0 ? (
                  notes.map((note) => (
                    <div key={note.id} className="space-y-3 mt-3">
                      {(
                        [
                          { key: 'S' as const, value: note.subjective },
                          { key: 'O' as const, value: note.objective },
                          { key: 'A' as const, value: note.assessment },
                          { key: 'P' as const, value: note.plan },
                        ] as const
                      ).map(({ key, value }) => {
                        const style = SOAP_SECTION_STYLES[key];
                        return (
                          <div key={key} className={`border-l-4 ${style.borderColor} pl-3`}>
                            <span
                              className={`inline-flex items-center justify-center w-5 h-5 rounded text-xs font-bold ${style.labelColor} mb-1`}
                            >
                              {key}
                            </span>
                            <p className="text-sm text-gray-700 whitespace-pre-wrap leading-relaxed">
                              {value}
                            </p>
                          </div>
                        );
                      })}
                      <div className="flex items-center gap-2 pt-2 text-xs text-gray-400">
                        <span>記録者: {note.author}</span>
                        <span>|</span>
                        <span>作成: {new Date(note.created_at).toLocaleDateString('ja-JP')}</span>
                      </div>
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-gray-400 py-3">SOAP記録なし</p>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
