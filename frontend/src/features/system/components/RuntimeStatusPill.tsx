import { useId, useState } from 'react';
import { Link } from 'react-router-dom';
import { useT } from '@/i18n';
import { cn } from '@/shared/lib/utils';
import type { AppChromeData } from '../hooks/useAppChrome';

/**
 * RuntimeStatusPill: the header's single runtime status. It reads like
 * the Configuration page header ("Node-RED 5.0.9 · Editable") and links
 * to the host page. Secondary context (read-only reason, edge mode) sits
 * in a tooltip shown on hover and focus and dismissed with Escape
 * (WCAG 1.4.13).
 */
export function RuntimeStatusPill({ runtime, edgeMode }: AppChromeData) {
  const { t } = useT();
  const detailsId = useId();
  const [dismissed, setDismissed] = useState(false);

  const product = t('common:appChrome.nodeRedVersionLabel');
  let label: string;
  let state: string;
  let stateClass = 'text-ds-text-muted';
  if (runtime.state === 'known') {
    label = `${product} ${runtime.version ?? t('common:appChrome.nodeRedVersionUnknown')}`;
    state = runtime.editable ? t('common:appChrome.compatModeEditable') : t('common:appChrome.compatModeReadOnly');
    stateClass = runtime.editable ? 'text-ds-on-tint-success' : 'text-ds-on-tint-warning';
  } else {
    label = product;
    state = runtime.state === 'loading' ? t('common:appChrome.runtimeChecking') : t('common:appChrome.runtimeUnknown');
  }

  const details = [
    runtime.state === 'known' && !runtime.editable ? runtime.reason : undefined,
    edgeMode === undefined
      ? undefined
      : edgeMode
        ? t('common:appChrome.edgeModeEnabled')
        : t('common:appChrome.edgeModeDisabled'),
  ].filter(Boolean);

  return (
    <div className="group relative" onMouseLeave={() => setDismissed(false)} onBlur={() => setDismissed(false)}>
      <Link
        to="/bootstrap"
        data-testid="runtime-status"
        aria-busy={runtime.state === 'loading' || undefined}
        aria-describedby={details.length ? detailsId : undefined}
        onKeyDown={(event) => {
          if (event.key === 'Escape') setDismissed(true);
        }}
        className={cn(
          'inline-flex min-h-11 items-center gap-1.5 whitespace-nowrap rounded-field border bg-ds-bg-surface px-3 text-xs',
          'text-ds-text-primary transition-colors hover:bg-[color-mix(in_srgb,var(--color-primary)_8%,var(--ds-bg-surface))]',
          // Dashed edge = not confirmed (DESIGN.md Shapes).
          runtime.state === 'known' ? 'border-ds-border-strong' : 'border-dashed border-ds-border-strong',
        )}
      >
        <span className="font-mono font-semibold tabular-nums">{label}</span>
        {' · '}
        <span className={cn('font-semibold', stateClass)}>{state}</span>
      </Link>
      {details.length > 0 && (
        <div
          role="tooltip"
          id={detailsId}
          className={cn(
            'absolute right-0 top-full z-50 hidden pt-1',
            !dismissed && 'group-hover:block group-focus-within:block',
          )}
        >
          <div className="w-max max-w-72 rounded-box border border-ds-border-default bg-ds-bg-elevated px-3 py-2 text-xs text-ds-text-secondary shadow-[var(--ds-shadow-raised)]">
            {details.map((line) => (
              <p key={line}>{line}</p>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
