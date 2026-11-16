import { useTranslation } from 'react-i18next';
import { SUPPORTED_LOCALES } from './constants';

// LocaleSwitcher — slice 1 of issue #767.
// Two-button group (EN | ES) rendered into the authenticated top bar.
// Click calls changeLanguage which triggers the persistence hook in
// I18nProvider.

export function LocaleSwitcher() {
  const { i18n, t } = useTranslation();
  const active = i18n.language;

  return (
    <div
      role="group"
      aria-label={t('common:languageSelector')}
      data-testid="locale-switcher"
      // Outline, not border: it paints over the buttons so they keep the full 44px target.
      className="inline-flex h-11 items-stretch overflow-hidden rounded-field outline outline-1 -outline-offset-1 outline-ds-border-strong bg-ds-bg-surface text-xs"
    >
      {SUPPORTED_LOCALES.map((lng) => {
        const isActive = active === lng;
        const code = lng.toUpperCase();
        return (
          <button
            key={lng}
            type="button"
            aria-pressed={isActive}
            aria-label={isActive ? t('common:locale.active', { code }) : t('common:locale.switchTo', { code })}
            data-locale={lng}
            onClick={() => {
              void i18n.changeLanguage(lng);
            }}
            className={
              'min-w-11 px-2 font-mono font-semibold uppercase transition-colors ' +
              (isActive
                ? 'bg-ds-accent-dim text-ds-on-tint-primary shadow-[inset_0_-2px_0_var(--ds-accent-primary)]'
                : 'text-ds-text-secondary hover:bg-base-200 hover:text-ds-text-primary')
            }
          >
            {lng}
          </button>
        );
      })}
    </div>
  );
}
