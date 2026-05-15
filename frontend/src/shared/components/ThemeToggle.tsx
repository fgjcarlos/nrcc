import { Sun, Moon, Monitor } from 'lucide-react';
import { useTheme } from '@/shared/hooks';

export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();

  const getIcon = () => {
    if (theme === 'light') return <Sun className="w-5 h-5" />;
    if (theme === 'dark') return <Moon className="w-5 h-5" />;
    return <Monitor className="w-5 h-5" />;
  };

  const getLabel = () => {
    if (theme === 'light') return 'Claro';
    if (theme === 'dark') return 'Oscuro';
    return 'Sistema';
  };

  return (
    <button
      onClick={toggleTheme}
      className="theme-toggle-shell inline-flex h-11 items-center gap-2 rounded-xl border px-3 text-sm font-medium text-base-content/80 transition-colors hover:text-base-content focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/50"
      title={`Tema: ${getLabel()}`}
      aria-label={`Tema: ${getLabel()}`}
    >
      <span className="grid h-7 w-7 place-items-center rounded-lg bg-accent/10 text-accent">
        {getIcon()}
      </span>
      <span className="hidden sm:inline">{getLabel()}</span>
    </button>
  );
}
