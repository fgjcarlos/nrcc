import React, { useEffect } from 'react';
import { I18nextProvider } from 'react-i18next';
import { I18N_STORAGE_KEY, DEFAULT_LOCALE, SUPPORTED_LOCALES } from './constants';
import { readStoredLocale } from './helpers';
import i18n from './config';

// Slice 1 of issue #767 -- typed wrapper around i18next's React
// adapter. Reads localStorage at mount (with EN fallback for invalid
// values), persists every locale change back to localStorage, and
// mirrors the active locale to document.documentElement.lang so the
// browser, accessibility tooling, and e2e selectors all see the same
// language attribute.

interface I18nProviderProps {
  children: React.ReactNode;
}

// normalizeLocale trims any region tag (e.g. "es-ES" -> "es") and
// falls back to DEFAULT_LOCALE when the value is not one of the
// supported locales. Keeps document.documentElement.lang in sync with
// the canonical short tag used by the catalogs.
function normalizeLocale(lng: string | undefined): string {
  if (!lng) return DEFAULT_LOCALE;
  const base = lng.toLowerCase().split(/[-_]/)[0];
  return (SUPPORTED_LOCALES as readonly string[]).includes(base)
    ? base
    : DEFAULT_LOCALE;
}

export function I18nProvider({ children }: I18nProviderProps) {
  useEffect(() => {
    const stored = readStoredLocale();
    const target = stored ?? DEFAULT_LOCALE;
    if (i18n.language !== target) {
      void i18n.changeLanguage(target);
    }

    // Mirror the active locale to <html lang> on mount (covers the
    // case where i18n was already initialized at the canonical
    // locale before this provider mounted) and on every change so the
    // attribute tracks i18n.language instead of getting out of sync.
    document.documentElement.lang = normalizeLocale(i18n.language);

    const onLanguageChanged = (lng: string) => {
      window.localStorage.setItem(I18N_STORAGE_KEY, lng);
      document.documentElement.lang = normalizeLocale(lng);
    };
    i18n.on('languageChanged', onLanguageChanged);
    return () => {
      i18n.off('languageChanged', onLanguageChanged);
    };
  }, []);

  return <I18nextProvider i18n={i18n}>{children}</I18nextProvider>;
}


