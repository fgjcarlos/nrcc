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
import { toast } from 'sonner';
import { configService } from '@/features/configuration/services';

export interface ApplyError {
  status: string;
  detail?: string;
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
    return {
      status: 'failed',
      error: {
        status: 'REVISION_MISSING',
        detail: 'Refresh before retrying the transaction.',
      },
    };
  }
  try {
    await configService.applyConfig(patch, expectedRevision);
    return { status: 'applied' };
  } catch (error) {
    const code =
      error instanceof AxiosError ? error.response?.data?.error?.code : undefined;
    return {
      status: 'failed',
      error: {
        status: code ?? 'UNKNOWN',
        detail: humanizeApplyError(code),
      },
    };
  }
}

export function humanizeApplyError(code: string | undefined): string {
  switch (code) {
    case 'SETTINGS_REVISION_CONFLICT':
      return 'Settings changed elsewhere. Refresh, review the redacted preview, then retry.';
    case 'APPLY_IN_FLIGHT':
      return 'Another transaction is in progress. Wait for it to finish, then retry.';
    case 'SETTINGS_VALIDATION_FAILED':
      return 'The configuration did not validate. Review the fields and retry.';
    default:
      return 'The transaction did not complete. Any failed readiness check is rolled back; review runtime readiness, then retry.';
  }
}

export function reportApplyError(error: ApplyError) {
  toast.error(error.detail ?? 'Transaction needs attention');
}

export function reportApplySuccess(label: string) {
  toast.success(label);
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
