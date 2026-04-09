import { create } from 'zustand'

export type Theme = 'nrcc_dark' | 'nrcc_light'

interface ThemeStore {
  theme: Theme
  setTheme: (theme: Theme) => void
}

/**
 * Zustand store for theme management.
 * Persists to localStorage to maintain user preference across sessions.
 */
export const useThemeStore = create<ThemeStore>((set) => {
  // Synchronously read from localStorage on store creation
  const storedTheme = localStorage.getItem('theme')
  const initialTheme: Theme =
    storedTheme === 'nrcc_dark' || storedTheme === 'nrcc_light'
      ? (storedTheme as Theme)
      : 'nrcc_dark' // Default to dark-first

  return {
    theme: initialTheme,
    setTheme: (theme: Theme) => {
      set({ theme })
      localStorage.setItem('theme', theme)
    },
  }
})
