import { useThemeStore } from '../../hooks/useThemeStore'

/**
 * ThemeToggle component - a button to switch between dark and light themes.
 * Displays a sun icon when in dark mode (click to switch to light)
 * and a moon icon when in light mode (click to switch to dark).
 */
export function ThemeToggle() {
  const { theme, setTheme } = useThemeStore()

  const handleToggle = () => {
    const newTheme = theme === 'nrcc_dark' ? 'nrcc_light' : 'nrcc_dark'
    setTheme(newTheme)
  }

  const label =
    theme === 'nrcc_dark'
      ? 'Switch to light mode'
      : 'Switch to dark mode'

  return (
    <button
      onClick={handleToggle}
      className="btn btn-ghost btn-circle btn-sm"
      aria-label={label}
      title={label}
    >
      {theme === 'nrcc_dark' ? (
        // Sun icon (when in dark mode)
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <circle cx="12" cy="12" r="5" />
          <line x1="12" y1="1" x2="12" y2="3" />
          <line x1="12" y1="21" x2="12" y2="23" />
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
          <line x1="1" y1="12" x2="3" y2="12" />
          <line x1="21" y1="12" x2="23" y2="12" />
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
        </svg>
      ) : (
        // Moon icon (when in light mode)
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
      )}
    </button>
  )
}
