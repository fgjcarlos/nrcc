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
      boxShadow: {
        'glow': '0 24px 60px rgba(0, 0, 0, 0.42), 0 0 0 1px rgba(148, 163, 184, 0.08)',
        'glow-light': '0 24px 48px rgba(15, 23, 42, 0.08), 0 0 0 1px rgba(15, 23, 42, 0.04)',
      },
    },
  },
  plugins: [daisyui],
  daisyui: {
    themes: [
      {
        // ══════════════════════════════════════════════════════════
        // DARK MODE — "Signal Console" (dark-first operations UI)
        // ══════════════════════════════════════════════════════════
        corporateDark: {
          "primary": "#ff3b30",
          "primary-content": "#ffffff",
          "primary-container": "#4f1110",
          "on-primary-container": "#fff1f0",

          "secondary": "#10242a",
          "secondary-content": "#dff7f3",

          "accent": "#38bdf8",
          "accent-content": "#041016",

          "base-100": "#06080a",
          "base-200": "#0c1116",
          "base-300": "#151c24",
          "base-content": "#eef4f8",

          "success": "#2dd4bf",
          "success-content": "#031312",
          "warning": "#fbbf24",
          "warning-content": "#1f1300",
          "error": "#ff3b30",
          "error-content": "#ffffff",
          "info": "#38bdf8",
          "info-content": "#041016",
        },
        // ══════════════════════════════════════════════════════════
        // LIGHT MODE — "Signal Light"
        // Light companion for the same dark-first palette
        // ══════════════════════════════════════════════════════════
        corporateLight: {
          "primary": "#d92d20",
          "primary-content": "#ffffff",
          "primary-container": "#fff1f0",
          "on-primary-container": "#4f1110",

          "secondary": "#e4f4f1",
          "secondary-content": "#0f3d3a",

          "accent": "#0369a1",
          "accent-content": "#ffffff",

          "base-100": "#f7f9fb",
          "base-200": "#edf2f6",
          "base-300": "#dce5ec",
          "base-content": "#111827",

          "success": "#0f766e",
          "success-content": "#ffffff",
          "warning": "#b45309",
          "warning-content": "#ffffff",
          "error": "#d92d20",
          "error-content": "#ffffff",
          "info": "#0284c7",
          "info-content": "#ffffff",
        },
      },
    ],
  },
}
