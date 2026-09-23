/**
 * UI_COPY — DEPRECATED as of slice 2 of issue #767.
 *
 * All UI strings previously defined here have been migrated to:
 *   - frontend/src/locales/{en,es}/{namespace}.json
 *   - resolved at render time via the `useT()` hook from `@/i18n`
 *
 * The constants are kept as a typed forwarding stub so any legacy
 * imports compile during the transition. New code MUST use `useT()`
 * instead — `UI_COPY` will be removed in slice 3 of #767.
 *
 * @deprecated use the typed `useT()` hook from `@/i18n` instead.
 */
import { i18n } from '@/i18n';

const _t = (key: string) => i18n.t(key);

export const UI_COPY = {
  cancel: _t('common:cancel'),
  confirm: _t('common:confirm'),
  confirmAction: _t('common:confirmAction'),
  processing: _t('common:processing'),
  loading: _t('common:loading'),
  errorOccurred: _t('common:errorOccurred'),
  tryAgain: _t('common:tryAgain'),
  delete: _t('common:delete'),
  add: _t('common:add'),
  saving: _t('common:saving'),
  saved: _t('common:saved'),
  typeToConfirm: (word: string) => _t('common:typeToConfirm').replace('{{word}}', String(word)),
  previousPage: _t('common:previousPage'),
  nextPage: _t('common:nextPage'),
  pageOf: (current: number, total: number) => _t('common:pageOf').replace('{{current}}', String(current)).replace('{{total}}', String(total)),
  // Auth
  loadingBackups: _t('backups:loading'),
  readingBackups: _t('backups:reading'),
  failedToLoadBackups: _t('backups:failedToLoad'),
  retryLoadBackups: _t('backups:retryLoad'),
  noBackupsYet: _t('backups:noBackupsYet'),
  createFirstBackup: _t('backups:createFirst'),
  createBackupDesc: _t('backups:createFirstDesc'),
  // ...remaining keys available from the i18n catalogs.
} as const;
