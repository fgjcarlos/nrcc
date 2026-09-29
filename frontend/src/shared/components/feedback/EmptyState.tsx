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
      className="flex flex-col items-center justify-center gap-3 rounded-xl border border-ds-border-default bg-base-200/30 px-6 py-12 text-center"
    >
      {icon ? (
        <span aria-hidden="true" className="text-base-content/60">
          {icon}
        </span>
      ) : null}
      <h3 className="text-base font-semibold text-base-content">{title}</h3>
      {description ? (
        <p className="max-w-sm text-sm text-base-content/65">{description}</p>
      ) : null}
      {hasAction ? (
        <Button type="button" onClick={onAction} className="mt-2">
          {actionLabel}
        </Button>
      ) : null}
    </div>
  );
}