/**
 * Helpers used by LoadingBoundary call sites.
 *
 * Lives in a `.ts` file (no React) so the `.tsx` component file can keep
 * exporting components only — Vite fast-refresh stays happy.
 */

/**
 * Returns true when the data is a known-empty array. null / undefined
 * / non-array values return false so callers can keep waiting.
 */
export function useEmptyState<T>(data: T | null | undefined): boolean {
  return Array.isArray(data) && data.length === 0;
}

/**
 * Stringifies an unknown error for display in the ErrorState description.
 */
export function describeError(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'string') return error;
  return '';
}