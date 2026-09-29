import type { ReactNode } from 'react';
import { useT } from '@/i18n';
import { Skeleton } from './Skeleton';
import { EmptyState } from './EmptyState';
import { ErrorState } from './ErrorState';
import { describeError } from './loadingBoundaryHelpers';

/**
 * LoadingBoundary — issue #766 slice G W1
 *
 * View-level wrapper that renders one of four states based on a
 * discriminated union. It does not own fetching; the parent passes a
 * `state` value computed from a React Query (or any) result. The
 * boundary decides which sub-component (Skeleton, EmptyState,
 * ErrorState, or children) to render.
 *
 * Priority order is fixed: pending → error → empty → children. This
 * mirrors `StateContainer` so the codebase has one mental model.
 */

export type LoadingBoundaryState =
  | { status: 'pending' }
  | {
      status: 'error';
      error: unknown;
      errorTitle?: string;
      errorDescription?: string;
      onRetry?: () => void;
    }
  | {
      status: 'empty';
      emptyTitle: string;
      emptyDescription?: string;
      emptyIcon?: ReactNode;
      emptyActionLabel?: string;
      onEmptyAction?: () => void;
    }
  | { status: 'success' };

export interface LoadingBoundaryProps {
  state: LoadingBoundaryState;
  children: ReactNode;
  loadingFallback?: ReactNode;
}

export function LoadingBoundary({
  state,
  children,
  loadingFallback,
}: LoadingBoundaryProps) {
  const { t } = useT();

  if (state.status === 'pending') {
    return loadingFallback ? <>{loadingFallback}</> : <Skeleton variant="rect" aria-label={t('common:loading')} />;
  }

  if (state.status === 'error') {
    return (
      <ErrorState
        title={state.errorTitle ?? t('common:errorOccurred')}
        description={state.errorDescription || describeError(state.error)}
        onRetry={state.onRetry}
      />
    );
  }

  if (state.status === 'empty') {
    return (
      <EmptyState
        title={state.emptyTitle}
        description={state.emptyDescription}
        icon={state.emptyIcon}
        actionLabel={state.emptyActionLabel}
        onAction={state.onEmptyAction}
      />
    );
  }

  return <>{children}</>;
}