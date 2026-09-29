import type { ReactNode } from 'react';
import { AlertCircle } from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';
import { useT } from '@/i18n';

/**
 * ErrorState — issue #766 slice G W1
 *
 * Reusable error surface used by every authenticated view when a query
 * fails. Renders an `aria-live="assertive"` alert region so screen
 * readers announce the failure as soon as it renders. A retry CTA is
 * optional; views whose retry is not safe (for example because the
 * error requires operator intervention) omit it.
 */

export interface ErrorStateProps {
  title: string;
  description?: string;
  icon?: ReactNode;
  onRetry?: () => void;
  retryLabel?: string;
}

export function ErrorState({
  title,
  description,
  icon,
  onRetry,
  retryLabel,
}: ErrorStateProps) {
  const { t } = useT();
  const label = retryLabel ?? t('common:tryAgain');

  return (
    <div
      role="alert"
      aria-live="assertive"
      className="flex flex-col items-center justify-center gap-3 rounded-xl border border-ds-danger/40 bg-ds-danger/8 px-6 py-12 text-center"
    >
      <span aria-hidden="true" className="text-ds-danger">
        {icon ?? <AlertCircle className="h-8 w-8" />}
      </span>
      <h3 className="text-base font-semibold text-base-content">{title}</h3>
      {description ? (
        <p className="max-w-sm text-sm text-base-content/70">{description}</p>
      ) : null}
      {onRetry ? (
        <Button type="button" variant="secondary" onClick={onRetry} className="mt-2">
          {label}
        </Button>
      ) : null}
    </div>
  );
}