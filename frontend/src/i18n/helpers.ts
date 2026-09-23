// Storage read/write helpers for the i18n foundation.

import type { SupportedLocale } from './constants';

export function readStoredLocale(): SupportedLocale | null {
  if (typeof window === 'undefined') return null;
  const raw = window.localStorage.getItem('nrcc.locale');
  if (raw === 'en' || raw === 'es') return raw;
  return null;
}
