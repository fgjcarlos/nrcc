/**
 * SecurityCenter helpers — issue #766 slice E
 *
 * Extracted from SecurityCenter so the four new boundary cards
 * (AdminAuth, HttpNodeAuth, HttpStaticAuth) share this code without
 * re-importing the combined widget. Each boundary card owns its own
 * state for its surface but reuses the same payload shape, error
 * handling, and "credentials must be bcrypt hashes" guard.
 */

import { AxiosError } from 'axios';
import { configService } from '@/features/configuration/services';

export interface ApplyError {
  status: string;
}

export interface ApplyResult {
  status: 'applied' | 'failed';
  error?: ApplyError;
}

/**
 * Apply a partial config payload to /api/config/apply with a
 * revision-conflict guard. Returns a structured result rather than
 * throwing so callers can render their own per-surface status.
 */
export async function applySecurityPatch(
  patch: Record<string, unknown>,
  expectedRevision: string | undefined,
): Promise<ApplyResult> {
  if (!expectedRevision) {
    return { status: 'failed', error: { status: 'REVISION_MISSING' } };
  }
  try {
    await configService.applyConfig(patch, expectedRevision);
    return { status: 'applied' };
  } catch (error) {
    const code =
      error instanceof AxiosError ? error.response?.data?.error?.code : undefined;
    return { status: 'failed', error: { status: code ?? 'UNKNOWN' } };
  }
}

/** Catalog key (security namespace) for the operator-facing apply error. */
export function applyErrorKey(code: string | undefined): string {
  switch (code) {
    case 'SETTINGS_REVISION_CONFLICT':
      return 'security:applyErrors.revisionConflict';
    case 'APPLY_IN_FLIGHT':
      return 'security:applyErrors.inFlight';
    case 'SETTINGS_VALIDATION_FAILED':
      return 'security:applyErrors.validationFailed';
    case 'REVISION_MISSING':
      return 'security:applyErrors.revisionMissing';
    default:
      return 'security:applyErrors.generic';
  }
}

/** True when the value looks like a bcrypt hash (or empty for legacy). */
export function isBcryptOrEmpty(value: string): boolean {
  if (!value) return true;
  return /^\$2[aby]\$/.test(value);
}

/**
 * Returns the subset of rawSettingsContent that *does not* contain
 * the legacy `httpNodeAuth` / `httpStaticAuth` aliases. Slice E keeps
 * this guard because legacy aliases still trigger the migration
 * confirmation dialog until the operator explicitly acknowledges them.
 */
export function detectLegacyAliases(source: string): string[] {
  return ['nodeHttpAuth', 'staticAuth'].filter((name) =>
    new RegExp(`\\b${name}\\s*:`).test(source),
  );
}
