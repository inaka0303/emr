import { formatVitals, type PatientInfoContext } from '../../utils/patientContext';

interface PatientContextPanelProps {
  info: PatientInfoContext | null;
}

export default function PatientContextPanel({ info }: PatientContextPanelProps) {
  if (!info) return null;

  const demographics = [info.age ? `${info.age}歳` : '', info.gender ?? ''].filter(Boolean).join(' / ');
  const visit = [info.encounter_date, info.department, info.encounter_type].filter(Boolean).join(' / ');
  const vitals = formatVitals(info.reception_vitals);

  return (
    <section className="bg-white rounded-lg border border-gray-200 shadow-sm">
      <header className="px-4 py-3 border-b border-gray-100 flex items-center justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold text-gray-800">患者情報</h3>
          <p className="text-xs text-gray-500 mt-0.5">{visit}</p>
        </div>
        {demographics && (
          <span className="px-2 py-1 rounded bg-gray-50 border border-gray-200 text-sm font-medium text-gray-700">
            {demographics}
          </span>
        )}
      </header>

      <div className="p-4 grid gap-4 xl:grid-cols-[minmax(220px,0.9fr)_minmax(280px,1.1fr)_minmax(280px,1.1fr)]">
        <div className="space-y-3">
          <InfoBlock label="主訴" value={info.chief_complaint} />
          <InfoBlock label="副主訴" values={info.secondary_complaints} />
          <InfoBlock label="受付バイタル" value={vitals} />
        </div>

        <div className="space-y-3">
          <InfoBlock label="既往" values={info.comorbidities} />
          <InfoBlock label="持参薬" values={info.current_medications} />
          <InfoBlock label="アレルギー" values={info.allergies} />
        </div>

        <div className="space-y-3">
          <InfoBlock label="家族歴" values={info.family_history} />
          <InfoBlock label="社会歴" value={info.social_history} />
        </div>
      </div>
    </section>
  );
}

function InfoBlock({ label, value, values }: { label: string; value?: string; values?: string[] }) {
  const items = values?.filter((v) => v.trim() !== '') ?? [];
  const text = (value ?? '').trim();
  if (!text && items.length === 0) return null;

  return (
    <div>
      <div className="text-xs font-medium text-gray-500 mb-1">{label}</div>
      {items.length > 0 ? (
        <ul className="space-y-1">
          {items.map((item, idx) => (
            <li key={`${label}-${idx}`} className="text-sm text-gray-800 leading-relaxed">
              {item}
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-gray-800 leading-relaxed whitespace-pre-wrap">{text}</p>
      )}
    </div>
  );
}
