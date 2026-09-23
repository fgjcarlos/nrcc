import React, { useEffect } from 'react';
import { I18nextProvider, useTranslation } from 'react-i18next';
import i18n from './config';

// Slice 1 of issue #767 — typed wrapper around i18next's React
// adapter. Reads localStorage at mount (with EN fallback for invalid
// values) and persists every locale change back to localStorage.

export const I18N_STORAGE_KEY = 'nrcc.locale';
export const SUPPORTED_LOCALES = ['en', 'es'] as const;
export const DEFAULT_LOCALE: 'en' = 'en';

export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

function readStoredLocale(): SupportedLocale | null {
  if (typeof window === 'undefined') return null;
  const raw = window.localStorage.getItem(I18N_STORAGE_KEY);
  if (raw === 'en' || raw === 'es') return raw;
  return null;
}

interface I18nProviderProps {
  children: React.ReactNode;
}

export function I18nProvider({ children }: I18nProviderProps) {
  useEffect(() => {
    const stored = readStoredLocale();
    const target = stored ?? DEFAULT_LOCALE;
    if (i18n.language !== target) {
      void i18n.changeLanguage(target);
    }

    const onLanguageChanged = (lng: string) => {
      window.localStorage.setItem(I18N_STORAGE_KEY, lng);
    };
    i18n.on('languageChanged', onLanguageChanged);
    return () => {
      i18n.off('languageChanged', onLanguageChanged);
    };
  }, []);

  return <I18nextProvider i18n={i18n}>{children}</I18nextProvider>;
}

// useT — typed wrapper around react-i18next's useTranslation.
export function useT() {
  const { t, i18n: instance } = useTranslation();
  return { t, i18n: instance };
}
