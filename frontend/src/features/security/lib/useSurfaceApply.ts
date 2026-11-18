import { useMemo, useState } from 'react';
import { deriveCircuit, type ApplyRun } from '@/features/configuration/lib/changeCircuit';
import type { ApplyResult } from '../components/securityCenterHelpers';

/**
 * Review drawer + change circuit state for one Security surface (P1 #1).
 * `draft` is the surface's edit state object: the Diff contact closes when
 * the drawer was opened for exactly this draft, as on Configuration.
 */
export function useSurfaceApply(pendingCount: number, draft: unknown, editable: boolean) {
  const [reviewOpen, setReviewOpen] = useState(false);
  const [reviewed, setReviewed] = useState<unknown>(null);
  const [run, setRun] = useState<ApplyRun>({ status: 'idle' });

  const circuit = useMemo(
    () =>
      deriveCircuit({ pendingCount, invalidFields: [], reviewed: reviewed === draft, readOnly: !editable, run }),
    [pendingCount, reviewed, draft, editable, run],
  );

  return {
    circuit,
    reviewOpen,
    openReview: () => {
      setReviewed(draft);
      setReviewOpen(true);
    },
    closeReview: () => setReviewOpen(false),
    /** A new edit starts a new circuit; the last outcome no longer applies. */
    edited: () => setRun((current) => (current.status === 'running' ? current : { status: 'idle' })),
    started: (count: number) => setRun({ status: 'running', count }),
    /** Every settled outcome closes the drawer; the circuit carries it. */
    settled: (count: number, result: ApplyResult) => {
      setReviewOpen(false);
      const code = result.error?.status;
      setRun(
        result.status === 'applied'
          ? { status: 'succeeded', count }
          : // No request left the browser; the inline error says why.
            code === 'REVISION_MISSING'
            ? { status: 'idle' }
            : { status: 'failed', count, code: code === 'UNKNOWN' ? undefined : code },
      );
    },
  };
}
