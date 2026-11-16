import type { ReactNode } from 'react';
import { Button } from '@/shared/components/ui/Button';

/**
 * EmptyState — issue #766 slice G W1
 *
 * Reusable empty-state surface used by every authenticated view when a
 * query succeeds with zero rows. Renders an `aria-live="polite"` status
 * region with an optional icon, description, and a single primary CTA
 * so screen readers announce the absence without operators having to
 * scan the page for it.
 */

export interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({
  title,
  description,
  icon,
  actionLabel,
  onAction,
}: EmptyStateProps) {
  const hasAction = Boolean(actionLabel && onAction);

  return (
    <div
      role="status"
      aria-live="polite"
      className="flex flex-col items-center justify-center gap-2 rounded-box border border-dashed border-ds-border-default px-6 py-10 text-center"
    >
      {icon ? (
        <span aria-hidden="true" className="text-ds-text-muted">
          {icon}
        </span>
      ) : null}
      <h3 className="text-sm font-semibold text-ds-text-primary">{title}</h3>
      {description ? (
        <p className="max-w-sm text-sm text-ds-text-secondary">{description}</p>
      ) : null}
      {hasAction ? (
        <Button type="button" onClick={onAction} className="mt-2">
          {actionLabel}
        </Button>
      ) : null}
    </div>
  );
}