/** @type {import('tailwindcss').Config} */
import daisyui from 'daisyui'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  // Enable dark mode based on data-theme attribute selector
  darkMode: ['selector', '[data-theme="dark"]'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"IBM Plex Sans"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
    },
  },
  plugins: [
    daisyui({
      // Use DaisyUI's built-in light and dark themes.
      // The actual NRCC color values are injected via CSS variable overrides in styles.css
      // for selectors [data-theme="light"] and [data-theme="dark"].
      // This architecture keeps DaisyUI's CSS generation clean and allows robust color customization.
      themes: ['light', 'dark'],
      base: true,
      styled: true,
      utils: true,
      logs: false,
    }),
  ],
}

