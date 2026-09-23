import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

// i18n instance — slice 1 of issue #767.
// EN is the source of truth; ES is the first maintained translation.
// Catalogs are bundled statically and registered through `resources`.

import enCommon from '@/locales/en/common.json';
import enAuth from '@/locales/en/auth.json';
import enBackups from '@/locales/en/backups.json';
import enConfig from '@/locales/en/configuration.json';
import enDashboard from '@/locales/en/dashboard.json';
import enEnvVars from '@/locales/en/env-vars.json';
import enFiles from '@/locales/en/files.json';
import enFlows from '@/locales/en/flows.json';
import enLibraries from '@/locales/en/libraries.json';
import enUpdates from '@/locales/en/updates.json';

import esCommon from '@/locales/es/common.json';
import esAuth from '@/locales/es/auth.json';
import esBackups from '@/locales/es/backups.json';
import esConfig from '@/locales/es/configuration.json';
import esDashboard from '@/locales/es/dashboard.json';
import esEnvVars from '@/locales/es/env-vars.json';
import esFiles from '@/locales/es/files.json';
import esFlows from '@/locales/es/flows.json';
import esLibraries from '@/locales/es/libraries.json';
import esUpdates from '@/locales/es/updates.json';

const resources = {
  en: {
    common: enCommon,
    auth: enAuth,
    backups: enBackups,
    configuration: enConfig,
    dashboard: enDashboard,
    'env-vars': enEnvVars,
    files: enFiles,
    flows: enFlows,
    libraries: enLibraries,
    updates: enUpdates,
  },
  es: {
    common: esCommon,
    auth: esAuth,
    backups: esBackups,
    configuration: esConfig,
    dashboard: esDashboard,
    'env-vars': esEnvVars,
    files: esFiles,
    flows: esFlows,
    libraries: esLibraries,
    updates: esUpdates,
  },
};

// Use a global flag so HMR re-loads don't double-register middleware.
const w = globalThis as unknown as { __nrccI18nInitOnce?: boolean };
if (!w.__nrccI18nInitOnce) {
  void i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      fallbackLng: 'en',
      defaultNS: 'common',
      ns: [
        'common', 'auth', 'backups', 'configuration', 'dashboard',
        'env-vars', 'files', 'flows', 'libraries', 'updates',
      ],
      resources,
      detection: {
        order: ['localStorage'],
        lookupLocalStorage: 'nrcc.locale',
        caches: ['localStorage'],
      },
      interpolation: { escapeValue: false },
      returnEmptyString: false,
      saveMissing: false,
    });
  w.__nrccI18nInitOnce = true;
}

export default i18n;
