import { formatBytes } from '@/shared/lib';
import type { BackupSummary } from '../services';

/**
 * Validates if a string is a valid ISO date
 */
export function isValidDate(value: string | null | undefined): boolean {
  return Boolean(value) && !Number.isNaN(new Date(value as string).getTime());
}

/**
 * Formats a backup creation date to locale string
 * Returns 'Fecha no disponible' if the date is invalid
 */
export function formatBackupDate(value: string | null | undefined): string {
  if (!isValidDate(value)) {
    return 'Fecha no disponible';
  }

  return new Date(value as string).toLocaleString();
}

/**
 * Formats a numeric count, returning '--' for non-finite values
 */
export function formatCount(value: number | null | undefined): string {
  return Number.isFinite(value) ? String(value) : '--';
}

/**
 * Formats a backup size using formatBytes utility
 * Returns '--' for negative or non-finite values
 */
export function formatBackupSize(value: number | null | undefined): string {
  return Number.isFinite(value) && (value as number) >= 0 ? formatBytes(value as number) : '--';
}

/**
 * Extracts error message from various error formats
 * Checks Error.message, response.data.error.message, then returns fallback
 */
export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  if (typeof error === 'object' && error !== null && 'response' in error) {
    const response = (error as { response?: { data?: { error?: { message?: string } } } }).response;
    const message = response?.data?.error?.message;
    if (message) {
      return message;
    }
  }

  return fallback;
}

/**
 * Generates a backup file label combining date and ID prefix
 * Format: 'backup-{ISO-date-safe}-{id-prefix}.zip'
 */
export function getBackupFileLabel(backup: BackupSummary): string {
  const date = isValidDate(backup.createdAt)
    ? new Date(backup.createdAt).toISOString().slice(0, 19).replace(/[:T]/g, '-')
    : 'backup';
  return `backup-${date}-${backup.id.slice(0, 8)}.zip`;
}

/**
 * Generates a backup summary string with type, date, and size
 * Format: '{type} · {date} · {size}'
 */
export function getBackupSummary(backup: BackupSummary, typeLabels: Record<BackupSummary['type'], string>): string {
  return [typeLabels[backup.type], formatBackupDate(backup.createdAt), formatBackupSize(backup.totalSize)].join(' · ');
}

/**
 * Gets the display name for a backup (name or ID)
 * Returns backup.name if non-empty after trim, otherwise backup.id
 */
export function getBackupDisplayName(backup: BackupSummary): string {
  return backup.name?.trim() || backup.id;
}
