import { ReactNode } from 'react';
import { Loader2, AlertCircle } from 'lucide-react';

/**
 * Presentational component for standardized loading/error/empty state branching
 * Renders children only when no state condition is active
 */
export interface StateContainerProps {
  isLoading: boolean;
  isError: boolean;
  isEmpty: boolean;
  loadingSlot?: ReactNode;
  errorSlot?: ReactNode;
  emptySlot?: ReactNode;
  children: ReactNode;
}

export function StateContainer({
  isLoading,
  isError,
  isEmpty,
  loadingSlot,
  errorSlot,
  emptySlot,
  children,
}: StateContainerProps) {
  // Priority order: loading → error → empty → children
  if (isLoading) {
    return (
      loadingSlot || (
        <div className="flex flex-col items-center justify-center gap-3 py-12">
          <Loader2 className="w-8 h-8 text-primary animate-spin" />
          <p className="text-base-content/60">{loadingSlot || 'Loading...'}</p>
        </div>
      )
    );
  }

  if (isError) {
    return (
      errorSlot || (
        <div className="p-6 rounded-lg border border-error/20 bg-error/8">
          <div className="flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-error flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-base-content font-medium">An error occurred</p>
              <p className="text-base-content/60 text-sm mt-1">Please try again later</p>
            </div>
          </div>
        </div>
      )
    );
  }

  if (isEmpty) {
    return (
      emptySlot || (
        <div className="flex flex-col items-center justify-center gap-2 py-12">
          <p className="text-base-content/60">No items yet</p>
        </div>
      )
    );
  }

  return <>{children}</>;
}
