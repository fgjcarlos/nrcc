import { create } from 'zustand'

export type Theme = 'dark' | 'light'

interface ThemeStore {
  theme: Theme
  setTheme: (theme: Theme) => void
}

/**
 * Zustand store for theme management.
 * Persists to localStorage to maintain user preference across sessions.
 * Uses DaisyUI standard theme names: 'dark' and 'light'.
 */
export const useThemeStore = create<ThemeStore>((set) => {
  // Synchronously read from localStorage on store creation
  const storedTheme = localStorage.getItem('theme')
  const initialTheme: Theme =
    storedTheme === 'dark' || storedTheme === 'light'
      ? (storedTheme as Theme)
      : 'dark' // Default to dark-first approach

  return {
    theme: initialTheme,
    setTheme: (theme: Theme) => {
      set({ theme })
      localStorage.setItem('theme', theme)
    },
  }
})
