import daisyui from 'daisyui'

/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class', '[data-theme]'],
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Primary (Node Amber)
        'ds-primary': '#e07b20',
        'ds-primary-dim': '#b85e0f',

        // Accent (Teal)
        'ds-accent': '#00c8b4',
        'ds-accent-dim': '#009987',

        // Backgrounds
        'ds-bg-void': '#050810',
        'ds-bg-base': '#080d18',
        'ds-bg-surface': '#0d1525',
        'ds-bg-elevated': '#121d32',
        'ds-bg-overlay': '#1a2844',

        // Borders
        'ds-border-subtle': '#1c2d4a',
        'ds-border-default': '#243860',
        'ds-border-strong': '#2d4878',

        // Text
        'ds-text-primary': '#e8f0ff',
        'ds-text-secondary': '#8fa8cc',
        'ds-text-muted': '#4d6a8a',

        // Semantic
        'ds-success': '#22c55e',
        'ds-warning': '#f59e0b',
        'ds-error': '#f43f5e',
        'ds-info': '#38bdf8',
      },
      fontFamily: {
        sans: ['Outfit', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
      boxShadow: {
        'glow': '0 24px 60px rgba(0, 0, 0, 0.42), 0 0 0 1px rgba(148, 163, 184, 0.08)',
        'glow-light': '0 24px 48px rgba(15, 23, 42, 0.08), 0 0 0 1px rgba(15, 23, 42, 0.04)',
        'glow-amber': '0 12px 30px rgba(224, 123, 32, 0.28), inset 0 1px rgba(255, 255, 255, 0.18)',
      },
    },
  },
  plugins: [daisyui],
  daisyui: {
    themes: [
      {
        // ══════════════════════════════════════════════════════════
        // DARK MODE — "Deep Signal" (dark-first operations UI)
        // ══════════════════════════════════════════════════════════
        corporateDark: {
          "primary": "#e07b20",
          "primary-content": "#ffffff",
          "primary-container": "#3d1f00",
          "on-primary-container": "#ffe8d5",

          "secondary": "#0d1525",
          "secondary-content": "#8fa8cc",

          "accent": "#00c8b4",
          "accent-content": "#001a17",

          "base-100": "#050810",
          "base-200": "#080d18",
          "base-300": "#0d1525",
          "base-content": "#e8f0ff",

          "success": "#22c55e",
          "success-content": "#052e16",
          "warning": "#f59e0b",
          "warning-content": "#1c0a00",
          "error": "#f43f5e",
          "error-content": "#ffffff",
          "info": "#38bdf8",
          "info-content": "#041016",
        },
        // ══════════════════════════════════════════════════════════
        // LIGHT MODE — "Deep Signal Light"
        // ══════════════════════════════════════════════════════════
        corporateLight: {
          "primary": "#b85e0f",
          "primary-content": "#ffffff",
          "primary-container": "#ffe8d5",
          "on-primary-container": "#3d1f00",

          "secondary": "#e8f0ff",
          "secondary-content": "#1c2d4a",

          "accent": "#009987",
          "accent-content": "#ffffff",

          "base-100": "#f4f6fb",
          "base-200": "#e8edf5",
          "base-300": "#d6dfed",
          "base-content": "#0d1525",

          "success": "#16a34a",
          "success-content": "#ffffff",
          "warning": "#d97706",
          "warning-content": "#ffffff",
          "error": "#e11d48",
          "error-content": "#ffffff",
          "info": "#0284c7",
          "info-content": "#ffffff",
        },
      },
    ],
  },
}
