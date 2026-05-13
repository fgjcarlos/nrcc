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
      className="gap-2 px-2 py-4 border rounded-lg btn btn-ghost btn-sm border-border bg-base-300/30 w-[120px]"
      title={`Tema: ${getLabel()}`}
      aria-label={`Tema: ${getLabel()}`}
    >
      {getIcon()}
      <span className="hidden sm:inline">{getLabel()}</span>
    </button>
  );
}
