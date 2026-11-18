import { ReactNode } from 'react';
import { Loader2 } from 'lucide-react';
import { useT } from '@/i18n';
import { EmptyState } from './feedback/EmptyState';
import { ErrorState } from './feedback/ErrorState';

/**
 * Presentational component for standardized loading/error/empty state branching
 * Renders children only when no state condition is active
 */
export interface StateContainerProps {
  isLoading: boolean;
  isError: boolean;
  isEmpty: boolean;
  /** Visible text of the default loading status (defaults to common:loading). */
  loadingLabel?: string;
  loadingSlot?: ReactNode;
  errorSlot?: ReactNode;
  emptySlot?: ReactNode;
  children: ReactNode;
}

export function StateContainer({
  isLoading,
  isError,
  isEmpty,
  loadingLabel,
  loadingSlot,
  errorSlot,
  emptySlot,
  children,
}: StateContainerProps) {
  const { t } = useT();

  // Priority order: loading → error → empty → children
  if (isLoading) {
    return (
      loadingSlot || (
        <div
          role="status"
          aria-busy="true"
          className="flex items-center justify-center gap-2 py-10 text-sm text-ds-text-secondary"
        >
          <Loader2 className="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
          {loadingLabel ?? t('common:loading')}
        </div>
      )
    );
  }

  if (isError) {
    return errorSlot || <ErrorState title={t('common:errorOccurred')} description={t('common:tryAgainLater')} />;
  }

  if (isEmpty) {
    return emptySlot || <EmptyState title={t('common:noItemsYet')} />;
  }

  return <>{children}</>;
}
