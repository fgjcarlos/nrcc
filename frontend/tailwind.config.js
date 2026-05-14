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
        // Brand (Signal Prime Red)
        'ds-brand-primary': '#d4472b',
        'ds-brand-hover': '#f05a3d',
        'ds-brand-dim': '#8f2f1f',

        // Accent (Signal Prime Cyan)
        'ds-accent-primary': '#0089b4',
        'ds-accent-hover': '#22b4df',
        'ds-accent-dim': '#075a73',

        // Backgrounds (Dark Mode)
        'ds-bg-void': '#07090d',
        'ds-bg-base': '#0f1419',
        'ds-bg-surface': '#141b23',
        'ds-bg-elevated': '#1a2332',
        'ds-bg-overlay': '#223044',

        // Borders (Dark Mode)
        'ds-border-subtle': '#263241',
        'ds-border-default': '#39485c',
        'ds-border-strong': '#4a5f7c',

        // Text (Dark Mode)
        'ds-text-primary': '#f5f7fa',
        'ds-text-secondary': '#d8dee8',
        'ds-text-tertiary': '#9aa7b8',
        'ds-text-muted': '#6f7d90',

        // Semantic Signals
        'ds-success': '#16a36a',
        'ds-warning': '#e8a811',
        'ds-danger': '#e54233',
        'ds-info': '#2c9edb',
      },
      fontFamily: {
         sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
         mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'monospace'],
       },
      boxShadow: {
         'glow': '0 24px 60px rgba(0, 0, 0, 0.42), 0 0 0 1px rgba(148, 163, 184, 0.08)',
         'glow-light': '0 24px 48px rgba(15, 23, 42, 0.08), 0 0 0 1px rgba(15, 23, 42, 0.04)',
         'glow-warning': '0 12px 30px rgba(224, 123, 32, 0.28), inset 0 1px rgba(255, 255, 255, 0.18)',
       },
    },
  },
  plugins: [daisyui],
  daisyui: {
    themes: [
      {
        // ══════════════════════════════════════════════════════════
        // DARK MODE — "Signal Prime Dark" (dark-first operations UI)
        // ══════════════════════════════════════════════════════════
        corporateDark: {
          "primary": "#d4472b",
          "primary-content": "#ffffff",

          "secondary": "#141b23",
          "secondary-content": "#d8dee8",

          "accent": "#0089b4",
          "accent-content": "#f5f7fa",

          "neutral": "#1a2332",
          "neutral-content": "#f5f7fa",

          "base-100": "#07090d",
          "base-200": "#0f1419",
          "base-300": "#141b23",
          "base-content": "#f5f7fa",

          "info": "#2c9edb",
          "info-content": "#ffffff",
          "success": "#16a36a",
          "success-content": "#ffffff",
          "warning": "#e8a811",
          "warning-content": "#07090d",
          "error": "#e54233",
          "error-content": "#ffffff",
        },
        // ══════════════════════════════════════════════════════════
        // LIGHT MODE — "Signal Prime Light"
        // ══════════════════════════════════════════════════════════
        corporateLight: {
          "primary": "#bd3f27",
          "primary-content": "#ffffff",

          "secondary": "#eef3f7",
          "secondary-content": "#263241",

          "accent": "#007aa0",
          "accent-content": "#ffffff",

          "neutral": "#536173",
          "neutral-content": "#f6f7f9",

          "base-100": "#f6f7f9",
          "base-200": "#edf1f5",
          "base-300": "#ffffff",
          "base-content": "#10151c",

          "info": "#1679b7",
          "info-content": "#ffffff",
          "success": "#0f8f5f",
          "success-content": "#ffffff",
          "warning": "#c98905",
          "warning-content": "#10151c",
          "error": "#c93429",
          "error-content": "#ffffff",
        },
      },
    ],
  },
}
