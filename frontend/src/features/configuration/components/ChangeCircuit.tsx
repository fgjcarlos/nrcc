import { useId, type CSSProperties } from 'react';
import { useT } from '@/i18n';
import { cn } from '@/shared/lib';
import type { Circuit, CircuitStep, ContactState } from '../lib/changeCircuit';

/**
 * ChangeCircuit (T9) — the gated change sequence drawn as a single-line
 * schematic: six contacts on one hairline conductor. A contact closes
 * when its step passes; a failure opens it and, when the server rolled
 * back, draws the rollback branch from Apply back to Backup.
 *
 * Every state is carried by text and symbol shape, never by colour alone.
 * Below `lg` the same list renders as a compact horizontal strip.
 */
export interface ChangeCircuitProps {
  circuit: Circuit;
  pendingCount: number;
  restartCount: number;
  reviewDisabled: boolean;
  onReview: () => void;
  /** Shown only while there are pending edits and no apply is running. */
  onDiscard?: () => void;
  className?: string;
}

const STATE_TEXT: Record<ContactState, string> = {
  idle: 'text-ds-text-muted',
  blocked: 'text-ds-on-tint-warning',
  active: 'text-ds-on-tint-info',
  passed: 'text-ds-on-tint-success',
  failed: 'text-ds-on-tint-error',
  unknown: 'text-ds-text-muted',
};

const STATE_STROKE: Record<ContactState, string> = {
  idle: 'text-ds-text-muted',
  blocked: 'text-ds-on-tint-warning',
  active: 'text-ds-on-tint-info',
  passed: 'text-ds-text-primary',
  failed: 'text-ds-on-tint-error',
  unknown: 'text-ds-text-muted',
};

/**
 * IEC normally-open contact on the conductor. Two terminal stubs; the
 * blade pivots on the lower one. Open: the blade is angled off the line,
 * leaving a gap. Closed: the blade lies on the line and bridges it. The
 * surface-coloured rect cuts the shared conductor between the terminals.
 */
function ContactSymbol({ state, index }: { state: ContactState; index: number }) {
  const closed = state === 'passed';
  return (
    <svg
      viewBox="0 0 20 20"
      width="20"
      height="20"
      aria-hidden="true"
      className={cn('relative shrink-0 max-lg:-rotate-90', STATE_STROKE[state])}
    >
      <rect x="6" y="5" width="8" height="10" className="fill-ds-bg-surface" />
      <g stroke="currentColor" strokeWidth="1.5" strokeLinecap="square">
        <line x1="10" y1="0" x2="10" y2="5" />
        <line x1="10" y1="15" x2="10" y2="20" />
        <line
          x1="10"
          y1="15"
          x2="10"
          y2="5"
          strokeDasharray={state === 'unknown' ? '2 1.5' : undefined}
          className="circuit-blade"
          data-open={closed ? 'false' : 'true'}
          style={{ '--contact-i': index } as CSSProperties}
        />
      </g>
      {closed && (
        <rect
          x="13.5"
          y="8"
          width="4"
          height="4"
          className="fill-[var(--ds-verified-fill)] stroke-ds-text-primary"
          strokeWidth="1"
        />
      )}
    </svg>
  );
}

export function ChangeCircuit({
  circuit,
  pendingCount,
  restartCount,
  reviewDisabled,
  onReview,
  onDiscard,
  className,
}: ChangeCircuitProps) {
  const { t } = useT();
  const titleId = useId();

  const stepName = (s: CircuitStep) => t(`configuration:circuit.steps.${s.id}`);
  const stateLabel = (s: CircuitStep) => t(`configuration:circuit.state.${s.state}`);
  const proofText = (s: CircuitStep) =>
    s.proof ? t(`configuration:circuit.proof.${s.proof.key}`, { count: s.proof.count }) : '';

  // The frontier is the step the operator should look at next.
  const frontier =
    circuit.steps.find((s) => s.state === 'failed') ??
    circuit.steps.find((s) => s.state !== 'passed') ??
    circuit.steps[circuit.steps.length - 1];
  const frontierIndex = circuit.steps.indexOf(frontier);
  const announcement = [`${stepName(frontier)}: ${stateLabel(frontier)}`, proofText(frontier)]
    .filter(Boolean)
    .join('. ');

  const total = [
    t('configuration:circuit.total', { count: pendingCount }),
    restartCount > 0 ? t('configuration:circuit.restartCost', { count: restartCount }) : null,
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <aside
      aria-labelledby={titleId}
      className={cn(
        'sticky top-0 z-10 min-w-0 self-start border-b border-ds-border-subtle bg-ds-bg-surface py-2',
        'lg:top-4 lg:border-b-0 lg:border-l lg:py-0 lg:pl-5',
        className,
      )}
      data-testid="change-circuit"
    >
      <h2 id={titleId} className="mb-3 text-sm font-semibold text-ds-text-primary max-lg:sr-only">
        {t('configuration:circuit.title')}
      </h2>

      {/* Below lg: a short summary over the six symbols; the list items
          keep the full state text for assistive tech. */}
      <p className="mb-1 truncate text-xs text-ds-text-secondary lg:hidden" aria-hidden="true">
        {t('configuration:circuit.summary', {
          index: frontierIndex + 1,
          total: circuit.steps.length,
          step: stepName(frontier),
          state: stateLabel(frontier),
        })}
      </p>

      {/* One continuous conductor (the ::before hairline) runs through all six contacts. */}
      <ol
        aria-labelledby={titleId}
        className={cn(
          'relative flex min-w-0 justify-between lg:flex-col lg:justify-start',
          "before:pointer-events-none before:absolute before:bg-ds-border-default before:content-['']",
          'max-lg:before:inset-x-0 max-lg:before:top-1/2 max-lg:before:h-px',
          'lg:before:inset-y-0 lg:before:left-[11.5px] lg:before:w-px',
        )}
        data-testid="change-circuit-steps"
      >
        {circuit.steps.map((s, index) => {
          const last = index === circuit.steps.length - 1;
          const branch = circuit.rollback && (s.id === 'backup' || s.id === 'apply');
          return (
            <li key={s.id} className="relative flex min-w-0 justify-center lg:justify-start lg:gap-3" data-state={s.state}>
              <span className="flex w-6 shrink-0 justify-center">
                <ContactSymbol state={s.state} index={index} />
              </span>
              {branch && (
                <span
                  aria-hidden="true"
                  className={cn(
                    'absolute left-0 w-3 border-dashed border-ds-danger max-lg:hidden',
                    s.id === 'backup' ? 'bottom-0 top-2.5 border-l border-t' : 'top-0 h-2.5 border-b border-l',
                  )}
                />
              )}

              <div className={cn('min-w-0 max-lg:sr-only lg:flex-1', !last && 'lg:pb-3')}>
                <div className="flex items-baseline gap-2">
                  <span className="font-mono text-xs text-ds-text-muted">{`K${index + 1}`}</span>
                  <span className="text-sm font-semibold text-ds-text-primary">{stepName(s)}</span>
                  <span className={cn('ml-auto text-xs font-semibold', STATE_TEXT[s.state])}>
                    {stateLabel(s)}
                  </span>
                </div>
                {s.proof && <p className="text-xs text-ds-text-secondary">{proofText(s)}</p>}
                {s.code && <p className="break-all font-mono text-xs text-ds-text-secondary">{s.code}</p>}
              </div>
            </li>
          );
        })}
      </ol>

      {circuit.rollback && (
        <p
          className={cn(
            'mt-2 flex items-baseline gap-2 border-t border-dashed border-ds-danger pt-2 text-xs',
            circuit.rollback === 'failed' ? 'font-semibold text-ds-on-tint-error' : 'text-ds-text-primary',
          )}
          data-testid="change-circuit-rollback"
        >
          <span className="font-mono text-ds-on-tint-error">RB</span>
          {t(`configuration:circuit.rollback.${circuit.rollback}`)}
        </p>
      )}

      <div className="mt-2 flex flex-col gap-2 border-t border-ds-border-subtle pt-2 lg:mt-3 lg:pt-3">
        <p className="text-xs tabular-nums text-ds-text-secondary lg:text-right" data-testid="change-circuit-total">
          {total}
        </p>
        <button
          type="button"
          onClick={onReview}
          disabled={reviewDisabled}
          className="action-btn-primary min-h-9 w-full justify-center text-sm"
        >
          {pendingCount > 0
            ? t('configuration:circuit.review', { count: pendingCount })
            : t('configuration:circuit.reviewNone')}
        </button>
        {onDiscard && (
          <button
            type="button"
            onClick={onDiscard}
            className="action-btn-ghost min-h-9 w-full text-sm text-ds-text-secondary hover:text-ds-text-primary"
          >
            {t('configuration:discardChanges')}
          </button>
        )}
      </div>

      <p className="sr-only" role="status" aria-live="polite" data-testid="change-circuit-live">
        {announcement}
      </p>
    </aside>
  );
}
