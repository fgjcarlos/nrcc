import type { RuntimeStatus } from '@/shared/types';
import type { BackupObservability } from '@/features/backups/services/backupService';

/**
 * Overview summary (T8): one sentence and at most one next action, derived
 * only from data the page already fetches. Anything not yet received is
 * "not reported", never a guessed state.
 */

export interface SecurityPosture {
  adminAuth: boolean;
  httpNodeAuth: boolean;
  httpStaticAuth: boolean;
  requireHttps: boolean;
}

export type SecurityVerdict = 'exposed' | 'partial' | 'lockedDown';

/**
 * Without adminAuth anyone who reaches the editor can deploy flows, so that
 * alone is the critical "exposed" verdict. The other surfaces are often left
 * off on purpose and only make the posture partial.
 */
export function securityVerdict(posture: SecurityPosture): SecurityVerdict {
  if (!posture.adminAuth) return 'exposed';
  return posture.httpNodeAuth && posture.httpStaticAuth && posture.requireHttps ? 'lockedDown' : 'partial';
}

/** Reads the four surfaces from /config using the keys the backend reports. */
export function postureFromConfig(config: Record<string, unknown> | undefined): SecurityPosture | undefined {
  if (!config) return undefined;
  return {
    adminAuth: Boolean(config.adminAuth),
    httpNodeAuth: Boolean(config.httpNodeAuth),
    httpStaticAuth: Boolean(config.httpStaticAuth),
    requireHttps: Boolean(config.requireHttps),
  };
}

export type BackupState = { kind: 'age'; createdAt: string } | { kind: 'none' } | { kind: 'unknown' };

export function backupState(backups: BackupObservability | undefined): BackupState {
  if (!backups) return { kind: 'unknown' };
  const createdAt = backups.latestBackup?.createdAt;
  if (!createdAt || Number.isNaN(new Date(createdAt).getTime())) return backups.latestBackup ? { kind: 'unknown' } : { kind: 'none' };
  return { kind: 'age', createdAt };
}

const KNOWN_RUNTIME: ReadonlyArray<RuntimeStatus> = ['running', 'stopped', 'error', 'detected'];

export function runtimeKey(status: string | undefined): RuntimeStatus {
  return KNOWN_RUNTIME.includes(status as RuntimeStatus) ? (status as RuntimeStatus) : 'unknown';
}

export type SummaryAction = { to: '/security'; labelKey: 'reviewSecurity' } | { to: '/backups'; labelKey: 'createBackup' };

/** The most urgent condition wins: exposure, then a missing backup; otherwise no action. */
export function summaryAction(verdict: SecurityVerdict | undefined, backup: BackupState): SummaryAction | null {
  if (verdict === 'exposed') return { to: '/security', labelKey: 'reviewSecurity' };
  if (backup.kind === 'none') return { to: '/backups', labelKey: 'createBackup' };
  return null;
}

/** "3 hours ago" in the active locale; future timestamps (clock skew) read as now. */
export function formatAge(iso: string, locale: string, now = Date.now()): string {
  const seconds = Math.max(0, Math.round((now - new Date(iso).getTime()) / 1000));
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });
  if (seconds < 3600) return rtf.format(-Math.floor(seconds / 60), 'minute');
  if (seconds < 172_800) return rtf.format(-Math.floor(seconds / 3600), 'hour');
  return rtf.format(-Math.floor(seconds / 86_400), 'day');
}
