import { useTranslation } from 'react-i18next';
import { SUPPORTED_LOCALES } from './constants';

// LocaleSwitcher — slice 1 of issue #767.
// Two-button group (EN | ES) rendered into the authenticated top bar.
// Click calls changeLanguage which triggers the persistence hook in
// I18nProvider.

export function LocaleSwitcher() {
  const { i18n } = useTranslation();
  const active = i18n.language;

  return (
    <div
      role="group"
      aria-label="Language selector"
      data-testid="locale-switcher"
      className="inline-flex items-center overflow-hidden rounded-xl border border-border/70 bg-base-300/45 text-xs"
    >
      {SUPPORTED_LOCALES.map((lng) => {
        const isActive = active === lng;
        return (
          <button
            key={lng}
            type="button"
            aria-pressed={isActive}
            aria-label={isActive ? `Active: ${lng.toUpperCase()}` : `Switch to ${lng.toUpperCase()}`}
            data-locale={lng}
            onClick={() => {
              void i18n.changeLanguage(lng);
            }}
            className={
              'px-2 py-1 font-semibold uppercase tracking-wider transition ' +
              (isActive
                ? 'bg-accent text-accent-content'
                : 'text-base-content/70 hover:bg-base-300')
            }
          >
            {lng}
          </button>
        );
      })}
    </div>
  );
}
