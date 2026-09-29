import { useEffect, useId, useMemo, useRef } from 'react';
import { createPortal } from 'react-dom';
import { ArrowRight, FileWarning, RotateCw, X } from 'lucide-react';
import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui/StatusChip';
import type {
  ConfigurationDiff,
  FieldDiff,
  RestartSeverity,
} from '../hooks/useConfigurationDiff';
import type { NodeRedConfigFormData } from '@/shared/types';
import { FIELD_LABELS } from '../lib/fieldLabels';

/**
 * Issue #766 slice F (W4) — reviewable safe-apply panel.
 *
 * Renders as a slide-in drawer from the right edge. The drawer opens
 * when the operator clicks the global Save button and renders every
 * field with `diff.pending === true`, grouped by restart severity
 * (hard → soft → none). The Apply button hands control back to the
 * caller; Cancel just dismisses without dispatching anything.
 *
 * Accessibility:
 *  - role="dialog" + aria-modal="true" + aria-labelledby to the title.
 *  - Esc cancels, Tab focus is trapped while open.
 *  - Initial focus lands on Cancel so destructive apply is not the
 *    default action.
 *
 * No data fetching lives here. The caller owns the save dispatch and
 * provides `isPending` to disable the apply button while the mutation
 * is in flight.
 */

export interface ReviewChangesPanelProps {
  isOpen: boolean;
  /** The diff summary; we read `fields` + `pendingCount`. */
  fields: ConfigurationDiff;
  pendingCount: number;
  /** Target description for the subtitle, e.g. "settings.js". */
  target: string;
  /** Disables Apply while the save mutation is in flight. */
  isPending?: boolean;
  /** Apply handler — the caller runs the existing save flow. */
  onApply: () => void;
  /** Cancel handler — closes the drawer without dispatching. */
  onCancel: () => void;
}

/** Group fields by restart severity in the canonical order. */
function groupFieldsByRestart(
  fields: ConfigurationDiff,
  pendingOnly: boolean
): Array<{ severity: RestartSeverity; entries: Array<{ key: keyof NodeRedConfigFormData; diff: FieldDiff }> }> {
  const buckets: Record<RestartSeverity, Array<{ key: keyof NodeRedConfigFormData; diff: FieldDiff }>> = {
    hard: [],
    soft: [],
    none: [],
  };
  for (const key of Object.keys(fields) as Array<keyof NodeRedConfigFormData>) {
    const f = fields[key];
    if (pendingOnly && !f.pending) continue;
    buckets[f.restart].push({ key, diff: f });
  }
  return (['hard', 'soft', 'none'] as const)
    .map((severity) => ({ severity, entries: buckets[severity] }))
    .filter((b) => b.entries.length > 0);
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  if (typeof value === 'number') return String(value);
  if (typeof value === 'string') return value.length > 0 ? value : '—';
  return JSON.stringify(value);
}

export function ReviewChangesPanel({
  isOpen,
  fields,
  pendingCount,
  target,
  isPending = false,
  onApply,
  onCancel,
}: ReviewChangesPanelProps) {
  const { t } = useT();
  const titleId = useId();
  const panelRef = useRef<HTMLDivElement>(null);
  const cancelButtonRef = useRef<HTMLButtonElement>(null);
  const previouslyFocusedRef = useRef<HTMLElement | null>(null);

  const groups = useMemo(
    () => groupFieldsByRestart(fields, true),
    [fields]
  );

  // Move focus into the drawer when it opens; restore on close.
  useEffect(() => {
    if (!isOpen) return;
    previouslyFocusedRef.current = document.activeElement as HTMLElement | null;
    const focusTimer = window.setTimeout(() => {
      cancelButtonRef.current?.focus();
    }, 0);
    return () => {
      window.clearTimeout(focusTimer);
      const previouslyFocused = previouslyFocusedRef.current;
      previouslyFocusedRef.current = null;
      if (previouslyFocused?.isConnected) previouslyFocused.focus();
    };
  }, [isOpen]);

  // Esc cancels + focus trap while open.
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isPending) {
        e.preventDefault();
        onCancel();
        return;
      }
      if (e.key !== 'Tab') return;
      const panel = panelRef.current;
      if (!panel) return;
      const focusable = Array.from(
        panel.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], [tabindex]:not([tabindex="-1"])'
        )
      );
      if (focusable.length === 0) {
        e.preventDefault();
        panel.focus();
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const active = document.activeElement;
      if (e.shiftKey && (active === first || !panel.contains(active))) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && (active === last || !panel.contains(active))) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, isPending, onCancel]);

  if (!isOpen) return null;

  const noChanges = pendingCount === 0 || groups.every((g) => g.entries.length === 0);

  return createPortal(
    <div
      className="fixed inset-0 z-[100] flex justify-end sm:items-stretch items-end"
      data-testid="review-changes-panel-portal"
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 modal-overlay"
        aria-hidden="true"
        onClick={isPending ? undefined : onCancel}
      />

      {/* Drawer */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className="relative surface-panel flex h-full w-full max-w-xl flex-col border-l border-border shadow-glow sm:rounded-none rounded-t-2xl sm:mt-0 mt-auto max-h-[85vh] sm:max-h-full"
        data-testid="review-changes-panel"
      >
        {/* Header */}
        <header className="flex items-start justify-between border-b ghost-divider modal-inner p-6">
          <div className="min-w-0">
            <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">
              {t('configuration:reviewPanel.title')}
            </p>
            <h2 id={titleId} className="mt-1 text-lg font-semibold text-base-content">
              {t('configuration:reviewPanel.subtitle', { target })}
            </h2>
          </div>
          <button
            type="button"
            onClick={onCancel}
            disabled={isPending}
            aria-label={t('configuration:reviewPanel.cancel')}
            className="text-body-secondary transition-colors hover:text-base-content disabled:opacity-50"
          >
            <X className="h-5 w-5" />
          </button>
        </header>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5">
          {noChanges ? (
            <p
              className="text-sm text-base-content/70"
              data-testid="review-changes-panel-empty"
            >
              {t('configuration:reviewPanel.noChanges')}
            </p>
          ) : (
            <div className="space-y-6">
              {groups.map((group) => (
                <section
                  key={group.severity}
                  data-testid={`review-changes-panel-group-${group.severity}`}
                >
                  <h3 className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-base-content/65">
                    {group.severity === 'hard' ? (
                      <RotateCw className="h-3.5 w-3.5" aria-hidden="true" />
                    ) : group.severity === 'soft' ? (
                      <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
                    ) : (
                      <FileWarning className="h-3.5 w-3.5" aria-hidden="true" />
                    )}
                    {group.severity === 'hard'
                      ? t('configuration:reviewPanel.groupHardRestart', { count: group.entries.length })
                      : group.severity === 'soft'
                      ? t('configuration:reviewPanel.groupSoftRestart', { count: group.entries.length })
                      : t('configuration:reviewPanel.groupNone', { count: group.entries.length })}
                  </h3>
                  <ul className="space-y-2">
                    {group.entries.map(({ key, diff }) => (
                      <li
                        key={key}
                        className="rounded-xl border border-border bg-base-100/60 p-3"
                        data-testid={`review-changes-panel-row-${key}`}
                      >
                        <div className="flex items-start justify-between gap-2">
                          <span className="text-sm font-medium text-base-content">
                            {FIELD_LABELS[key] ?? key}
                          </span>
                          <StatusChip
                            variant={diff.validation.state === 'invalid' ? 'warning' : 'neutral'}
                            size="sm"
                          >
                            {diff.validation.state === 'invalid'
                              ? t('configuration:fieldStatus.validation.invalid')
                              : t('configuration:fieldStatus.validation.pending')}
                          </StatusChip>
                        </div>
                        <div className="mt-2 grid grid-cols-[max-content_1fr] gap-x-3 gap-y-1 text-xs">
                          <span className="text-base-content/50">
                            {t('configuration:reviewPanel.configuredLabel')}
                          </span>
                          <span className="font-mono text-base-content/80 break-all">
                            {formatValue(diff.configuredValue)}
                          </span>
                          <span className="text-base-content/50">
                            {t('configuration:reviewPanel.newLabel')}
                          </span>
                          <span className="font-mono text-base-content break-all">
                            {formatValue(diff.formValue)}
                          </span>
                        </div>
                      </li>
                    ))}
                  </ul>
                </section>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <footer className="flex justify-end gap-3 border-t ghost-divider modal-inner px-6 py-4">
          <button
            ref={cancelButtonRef}
            type="button"
            onClick={onCancel}
            disabled={isPending}
            className="action-btn-secondary"
          >
            {t('configuration:reviewPanel.cancel')}
          </button>
          <button
            type="button"
            onClick={onApply}
            disabled={isPending || noChanges}
            className="action-btn-primary"
            data-testid="review-changes-panel-apply"
          >
            {isPending
              ? t('configuration:reviewPanel.applying')
              : t('configuration:reviewPanel.apply')}
          </button>
        </footer>
      </div>
    </div>,
    document.body
  );
}
