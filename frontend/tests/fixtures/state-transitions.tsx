import { StrictMode, type ComponentProps } from 'react';
import { createRoot } from 'react-dom/client';
import SOAPEditor from '../../src/components/chart/SOAPEditor';
import AdmissionSummaryDrafter from '../../src/components/slm/AdmissionSummaryDrafter';
import { useEncounterInterview } from '../../src/hooks/useEncounterInterview';

export type Fixture =
  | { kind: 'soap'; props: ComponentProps<typeof SOAPEditor> }
  | { kind: 'admission'; props: ComponentProps<typeof AdmissionSummaryDrafter> }
  | { kind: 'interview'; encounterId: number | null };

declare global {
  interface Window {
    renderFixture: (fixture: Fixture) => void;
  }
}

export function InterviewResult({ encounterId }: { encounterId: number | null }) {
  const result = useEncounterInterview(encounterId);
  return <output data-testid="interview">{JSON.stringify(result)}</output>;
}

const root = createRoot(document.getElementById('root')!);
window.renderFixture = (fixture) => {
  root.render(
    <StrictMode>
      {fixture.kind === 'soap' ? <SOAPEditor {...fixture.props} />
        : fixture.kind === 'admission' ? <AdmissionSummaryDrafter {...fixture.props} />
          : <InterviewResult encounterId={fixture.encounterId} />}
    </StrictMode>,
  );
};
