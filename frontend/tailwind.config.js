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

        // Brand (Signal Prime Red)
        'ds-brand-primary': 'var(--ds-brand-primary)',
        'ds-brand-hover': 'var(--ds-brand-hover)',
        'ds-brand-dim': 'var(--ds-brand-dim)',

        // Accent (Signal Prime Cyan)
        'ds-accent-primary': 'var(--ds-accent-primary)',
        'ds-accent-hover': 'var(--ds-accent-hover)',
        'ds-accent-dim': 'var(--ds-accent-dim)',

        // Backgrounds
        'ds-bg-void': 'var(--ds-bg-void)',
        'ds-bg-base': 'var(--ds-bg-base)',
        'ds-bg-surface': 'var(--ds-bg-surface)',
        'ds-bg-elevated': 'var(--ds-bg-elevated)',
        'ds-bg-overlay': 'var(--ds-bg-overlay)',

        // Borders (the --ds-border-* CSS tokens carry different values,
        // so the utilities use dedicated --ds-ui-border-* variables)
        'ds-border-subtle': 'var(--ds-ui-border-subtle)',
        'ds-border-default': 'var(--ds-ui-border-default)',
        'ds-border-strong': 'var(--ds-ui-border-strong)',

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
         sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
         mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'monospace'],
       },
      // ── Typography scale (issue #766 slice B) ─────────────────
      // Named tokens for the typography hierarchy. Use Tailwind
      // utilities like `text-ds-h1`, `text-ds-body` etc. so every page
      // renders the same hierarchy. Pair with `font-semibold` /
      // `font-medium` for explicit weight when needed.
      fontSize: {
        'ds-display': ['2.25rem', { lineHeight: '2.5rem', letterSpacing: '-0.01em', fontWeight: '700' }],
        'ds-h1':      ['1.875rem', { lineHeight: '2.25rem', letterSpacing: '-0.01em', fontWeight: '700' }],
        'ds-h2':      ['1.5rem',   { lineHeight: '2rem',    fontWeight: '600' }],
        'ds-h3':      ['1.25rem',  { lineHeight: '1.75rem', fontWeight: '600' }],
        'ds-body':    ['0.875rem', { lineHeight: '1.375rem', fontWeight: '400' }],
        'ds-caption': ['0.75rem',  { lineHeight: '1rem',     fontWeight: '500' }],
        'ds-eyebrow': ['0.68rem',  { lineHeight: '1rem',     letterSpacing: '0.24em', fontWeight: '600' }],
      },
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
         'glow': '0 24px 60px rgba(0, 0, 0, 0.42), 0 0 0 1px rgba(148, 163, 184, 0.08)',
         'glow-light': '0 24px 48px rgba(15, 23, 42, 0.08), 0 0 0 1px rgba(15, 23, 42, 0.04)',
         'glow-warning': '0 12px 30px rgba(224, 123, 32, 0.28), inset 0 1px rgba(255, 255, 255, 0.18)',
         // Focus ring used by the ds-focus-ring utility in index.css.
         // Two-stop ring: outer halo = surface colour, inner = accent.
         'ds-focus-ring': '0 0 0 2px var(--ds-bg-base, #07090d), 0 0 0 4px var(--ds-accent-primary, #0089b4)',
       },
    },
  },
}
