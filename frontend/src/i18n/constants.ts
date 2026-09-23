// Constants for the i18n foundation. Lives in its own file so
// `frontend/src/i18n/provider.tsx` only exports React components
// (preserves the fast-refresh / HMR contract).

export const I18N_STORAGE_KEY = 'nrcc.locale' as const;
export const SUPPORTED_LOCALES = ['en', 'es'] as const;
export const DEFAULT_LOCALE = 'en' as const;
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];
