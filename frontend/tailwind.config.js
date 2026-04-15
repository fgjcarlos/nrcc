/** @type {import('tailwindcss').Config} */
import daisyui from 'daisyui'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  // Enable dark mode based on data-theme attribute selector
  darkMode: ['selector', '[data-theme="dark"]'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Berkeley Mono"', '"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', '"Liberation Mono"', '"Courier New"', 'monospace'],
        mono: ['"Berkeley Mono"', '"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', '"Liberation Mono"', '"Courier New"', 'monospace'],
      },
      borderRadius: {
        DEFAULT: '4px',
        none: '0',
        sm: '2px',
        input: '6px',
        full: '9999px',
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

