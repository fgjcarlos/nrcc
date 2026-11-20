import type { ReactNode } from 'react';
import { Button } from '@/shared/components/ui/Button';

/**
 * EmptyState — issue #766 slice G W1
 *
 * Reusable empty-state surface used by every authenticated view when a
 * query succeeds with zero rows. Renders an `aria-live="polite"` status
 * region with an optional icon, description, and a single primary CTA
 * so screen readers announce the absence without operators having to
 * scan the page for it. It lies flat on the page sheet between two
 * hairline rules rather than as a second card inside it.
 */

export interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  actionLabel?: string;
  onAction?: () => void;
  /** Heading level for the title; pick the next level below the enclosing heading. */
  headingLevel?: 2 | 3;
}

export function EmptyState({
  title,
  description,
  icon,
  actionLabel,
  onAction,
  headingLevel = 3,
}: EmptyStateProps) {
  const hasAction = Boolean(actionLabel && onAction);
  const Heading = headingLevel === 2 ? 'h2' : 'h3';

  return (
    <div
      role="status"
      aria-live="polite"
      className="flex flex-col items-center justify-center gap-2 border-y border-ds-border-subtle px-6 py-10 text-center"
    >
      {icon ? (
        <span aria-hidden="true" className="text-ds-text-muted">
          {icon}
        </span>
      ) : null}
      <Heading className="text-sm font-semibold text-ds-text-primary">{title}</Heading>
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