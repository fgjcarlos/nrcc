import { Sun, Moon, Monitor } from 'lucide-react';
import { useT } from '@/i18n';
import { useTheme } from '@/shared/hooks';

export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();
  const { t } = useT();

  const Icon = theme === 'light' ? Sun : theme === 'dark' ? Moon : Monitor;
  const label =
    theme === 'light' ? t('common:themeLight') : theme === 'dark' ? t('common:themeDark') : t('common:themeSystem');
  const name = t('common:themeToggleLabel', { theme: label });

  return (
    <button
      type="button"
      onClick={toggleTheme}
      className="theme-toggle-shell inline-flex h-11 min-w-11 items-center justify-center gap-2 rounded-field border px-3 text-sm font-medium text-ds-text-primary transition-colors"
      title={name}
      aria-label={name}
    >
      <Icon className="h-5 w-5 text-ds-text-secondary" aria-hidden="true" />
      <span className="hidden sm:inline">{label}</span>
    </button>
  );
}
