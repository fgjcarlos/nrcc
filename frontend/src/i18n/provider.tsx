import React, { useEffect } from 'react';
import { I18nextProvider } from 'react-i18next';
import { I18N_STORAGE_KEY, DEFAULT_LOCALE } from './constants';
import { readStoredLocale } from './helpers';
import i18n from './config';

// Slice 1 of issue #767 -- typed wrapper around i18next's React
// adapter. Reads localStorage at mount (with EN fallback for invalid
// values) and persists every locale change back to localStorage.

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


