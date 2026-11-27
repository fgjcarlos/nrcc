import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

// i18n instance — slice 1 of issue #767.
// EN is the source of truth; ES is the first maintained translation.
// Catalogs are bundled statically and registered through `resources`.

import enCommon from '@/locales/en/common.json';
import enAuth from '@/locales/en/auth.json';
import enSetup from '@/locales/en/setup.json';
import enBackups from '@/locales/en/backups.json';
import enBootstrap from '@/locales/en/bootstrap.json';
import enConfig from '@/locales/en/configuration.json';
import enDashboard from '@/locales/en/dashboard.json';
import enEnvVars from '@/locales/en/env-vars.json';
import enLibraries from '@/locales/en/libraries.json';
import enSecurity from '@/locales/en/security.json';
import enUpdates from '@/locales/en/updates.json';
import enLogs from '@/locales/en/logs.json';

import esCommon from '@/locales/es/common.json';
import esAuth from '@/locales/es/auth.json';
import esSetup from '@/locales/es/setup.json';
import esBackups from '@/locales/es/backups.json';
import esBootstrap from '@/locales/es/bootstrap.json';
import esConfig from '@/locales/es/configuration.json';
import esDashboard from '@/locales/es/dashboard.json';
import esEnvVars from '@/locales/es/env-vars.json';
import esLibraries from '@/locales/es/libraries.json';
import esSecurity from '@/locales/es/security.json';
import esUpdates from '@/locales/es/updates.json';
import esLogs from '@/locales/es/logs.json';

const resources = {
  en: {
    common: enCommon,
    auth: enAuth,
    setup: enSetup,
    backups: enBackups,
    bootstrap: enBootstrap,
    configuration: enConfig,
    dashboard: enDashboard,
    'env-vars': enEnvVars,
    libraries: enLibraries,
    security: enSecurity,
    updates: enUpdates,
    logs: enLogs,
  },
  es: {
    common: esCommon,
    auth: esAuth,
    setup: esSetup,
    backups: esBackups,
    bootstrap: esBootstrap,
    configuration: esConfig,
    dashboard: esDashboard,
    'env-vars': esEnvVars,
    libraries: esLibraries,
    security: esSecurity,
    updates: esUpdates,
    logs: esLogs,
  },
};

// Use a global flag so HMR re-loads don't double-register middleware.
// SAFETY: globalThis is augmented with this flag only inside this module;
//             the `as unknown as { ... }` narrows the Global type to the
//             intentionally injected shape without polluting other modules.
const w = globalThis as unknown as { __nrccI18nInitOnce?: boolean };
if (!w.__nrccI18nInitOnce) {
  void i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      fallbackLng: 'en',
      defaultNS: 'common',
      ns: [
        'common', 'auth', 'setup', 'backups', 'bootstrap', 'configuration',
        'dashboard', 'env-vars', 'libraries', 'security', 'updates', 'logs',
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
