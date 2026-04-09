import { useEffect } from 'react'
import { useThemeStore } from '../../hooks/useThemeStore'

interface ThemeProviderProps {
  children: React.ReactNode
}

/**
 * ThemeProvider component that syncs the Zustand theme state to the DOM.
 * On mount and when theme changes, sets the `data-theme` attribute on the document root.
 * This allows both Tailwind dark: variants and DaisyUI theme switching to work.
 */
export function ThemeProvider({ children }: ThemeProviderProps) {
  const theme = useThemeStore((state) => state.theme)

  useEffect(() => {
    // Sync theme to document element whenever it changes
    document.documentElement.setAttribute('data-theme', theme)
  }, [theme])

  return <>{children}</>
}
