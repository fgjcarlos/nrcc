/**
 * Compile-time feature flags for capabilities whose backend is not yet shipped.
 *
 * Keep these honest: a flag is `false` only while the corresponding server
 * endpoint is a stub. Flipping a flag to `true` must coincide with a real
 * backend, never to "unhide" a button that still 501s.
 */
export const FEATURES = {
  /**
   * AI pattern detection across flows.
   *
   * The endpoints (`POST /api/ai/analyze/patterns`,
   * `GET /api/ai/patterns/{id}/readme`, `GET /api/ai/patterns/{id}/download`)
   * are documented in docs/openapi.yaml as `x-status: stub`; the Go handlers in
   * internal/handler/ai.go return 501 NOT_IMPLEMENTED. Until that backend
   * exists, the UI shows a "coming soon" state instead of letting users click
   * into a generic error toast. See issue #295.
   */
  patternDetection: false,
} as const;
