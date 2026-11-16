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
        // Theme-aware palette: every ds-* colour resolves through a CSS
        // variable declared in both the corporateDark and corporateLight
        // blocks of src/index.css (issue #889). Guarded by
        // scripts/tailwind-ds-vars.test.mjs.

        // Brand: neutral-conductor blue, action and selection (Esquema eléctrico)
        'ds-brand-primary': 'var(--ds-brand-primary)',
        'ds-brand-hover': 'var(--ds-brand-hover)',
        'ds-brand-dim': 'var(--ds-brand-dim)',

        // Accent: same conductor blue (focus and selection)
        'ds-accent-primary': 'var(--ds-accent-primary)',
        'ds-accent-hover': 'var(--ds-accent-hover)',
        'ds-accent-dim': 'var(--ds-accent-dim)',

        // Backgrounds
        'ds-bg-void': 'var(--ds-bg-void)',
        'ds-bg-base': 'var(--ds-bg-base)',
        'ds-bg-surface': 'var(--ds-bg-surface)',
        'ds-bg-elevated': 'var(--ds-bg-elevated)',
        'ds-bg-overlay': 'var(--ds-bg-overlay)',

        // Borders read the same-named CSS tokens: subtle and default are the
        // hairline (decorative), strong is the control border (>=3:1).
        'ds-border-subtle': 'var(--ds-border-subtle)',
        'ds-border-default': 'var(--ds-border-default)',
        'ds-border-strong': 'var(--ds-border-strong)',

        // Text
        'ds-text-primary': 'var(--ds-text-primary)',
        'ds-text-secondary': 'var(--ds-text-secondary)',
        'ds-text-tertiary': 'var(--ds-text-tertiary)',
        'ds-text-muted': 'var(--ds-text-muted)',

        // Semantic Signals
        'ds-success': 'var(--ds-success)',
        'ds-warning': 'var(--ds-warning)',
        'ds-danger': 'var(--ds-danger)',
        'ds-info': 'var(--ds-info)',

        // On-tint text: theme-aware labels on translucent semantic tints
        'ds-on-tint-primary': 'var(--ds-on-tint-primary)',
        'ds-on-tint-success': 'var(--ds-on-tint-success)',
        'ds-on-tint-info': 'var(--ds-on-tint-info)',
        'ds-on-tint-warning': 'var(--ds-on-tint-warning)',
        'ds-on-tint-error': 'var(--ds-on-tint-error)',
      },
      fontFamily: {
        // Self-hosted through @fontsource (src/main.tsx); no CDN at runtime.
        sans: ['Overpass Variable', 'system-ui', 'sans-serif'],
        mono: ['Overpass Mono', 'ui-monospace', 'monospace'],
      },
      // Type sizes: DESIGN.md allows only 20, 14 and 12px, which are the
      // stock text-xl, text-sm and text-xs utilities.
      // ── Density scale (issue #766 slice B) ────────────────────
      // Semantic vertical-rhythm tokens. `compact` for data-dense
      // list rows; `comfortable` for forms and cards (default);
      // `spacious` for hero sections and dialog bodies.
      spacing: {
        'ds-density-compact':      '0.25rem',
        'ds-density-comfortable':  '0.5rem',
        'ds-density-spacious':     '0.75rem',
      },
      boxShadow: {
        // The legacy glow names stay because components still use them;
        // they now resolve to the flat raised elevation (max 0 1px 2px).
        'glow': 'var(--ds-shadow-raised)',
        'glow-light': 'var(--ds-shadow-raised)',
        'glow-warning': 'var(--ds-shadow-raised)',
        // Focus ring used by the ds-focus-ring utility in index.css.
        // Two-stop ring: outer halo = surface colour, inner = accent.
        'ds-focus-ring': '0 0 0 2px var(--ds-bg-base), 0 0 0 4px var(--ds-accent-primary)',
      },
    },
  },
}
