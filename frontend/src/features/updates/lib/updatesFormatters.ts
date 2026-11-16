/**
 * Formatter functions for Updates feature
 */

type Translate = (key: string, options?: Record<string, unknown>) => string;

export function formatCheckedAt(checkedAtStr: string | undefined, t: Translate, locale?: string): string {
  const date = checkedAtStr ? new Date(checkedAtStr) : null;
  // Go marshals an unset time as 0001-01-01: that is "never checked", not a date.
  if (!date || Number.isNaN(date.getTime()) || date.getUTCFullYear() < 2000) return t('updates:never');

  const diffMins = Math.floor((Date.now() - date.getTime()) / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 1) return t('updates:justNow');
  if (diffMins < 60) return t('updates:minutesAgo', { count: diffMins });
  if (diffHours < 24) return t('updates:hoursAgo', { count: diffHours });
  if (diffDays < 7) return t('updates:daysAgo', { count: diffDays });

  return date.toLocaleDateString(locale);
}
