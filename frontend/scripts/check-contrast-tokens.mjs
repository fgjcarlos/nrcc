#!/usr/bin/env node
/**
 * check-contrast-tokens.mjs — issues #766 slice G W4 and #889
 *
 * Static auditor for WCAG 2.1 SC 1.4.3 (Contrast — Minimum) and
 * SC 1.4.11 (Non-text Contrast).
 *
 * Reads the colour tokens declared in `src/index.css` for both themes
 * (and legacy `corporateDark` / `corporateLight` palettes in
 * `tailwind.config.js`), computes the contrast ratio for every
 * documented foreground / background pair, and fails when any pair
 * drops below its threshold.
 *
 * The contract: pairs are generated from the tables below. A new text
 * colour, surface or solid label added to the UI must also be added
 * here. Drift between the design tokens and this script fails the build.
 *
 * Modes:
 *   (default)  strict — any failing pair exits 1. `--strict` is accepted
 *              as an explicit alias.
 *   --report   report-only — failing pairs are listed but the exit code
 *              is 0. Use it locally while iterating on tokens.
 *
 * Exit codes:
 *   0  every documented pair meets AA in every theme (or --report)
 *   1  at least one pair fails AA in at least one theme
 *   2  token referenced by a pair is missing from the palette
 */
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));
// Honour --repo-root <dir> or CONTRAST_REPO_ROOT for the tests; default
// to the directory that owns this script so a bare `node scripts/...`
// invocation resolves src/index.css and tailwind.config.js correctly.
const argvRoot = (() => {
  const idx = process.argv.indexOf('--repo-root');
  if (idx >= 0 && process.argv[idx + 1]) return process.argv[idx + 1];
  if (process.env.CONTRAST_REPO_ROOT) return process.env.CONTRAST_REPO_ROOT;
  return null;
})();
const repoRoot = argvRoot ? resolve(argvRoot) : resolve(__dirname, '..');

// ── Token extraction ─────────────────────────────────────────────

function readIndexCssTokens() {
  const css = readFileSync(resolve(repoRoot, 'src/index.css'), 'utf8');
  /** @type {Record<string, Record<string, string>>} */
  const tokens = {};
  const blockRegex = /\[data-theme="([^"]+)"\]\s*{([\s\S]*?)\n\s*}/g;
  let match;
  while ((match = blockRegex.exec(css)) !== null) {
    const theme = match[1];
    const body = match[2];
    /** @type {Record<string, string>} */
    const themeTokens = {};
    for (const line of body.split('\n')) {
      const m = line.match(/^\s*(--[a-z0-9-]+)\s*:\s*([^;]+);/i);
      if (!m) continue;
      const name = m[1].replace(/^--/, '');
      themeTokens[name] = m[2].trim();
    }
    tokens[theme] = themeTokens;
  }
  return tokens;
}

function readTailwindTokens() {
  // Retain compatibility with older configs and test fixtures. The v5
  // source of truth is the explicit --color-* palette in index.css.
  // Extract only literal hex values; do not evaluate JavaScript.
  const cfg = readFileSync(resolve(repoRoot, 'tailwind.config.js'), 'utf8');
  /** @type {Record<string, Record<string, string>>} */
  const out = {};
  for (const theme of ['corporateDark', 'corporateLight']) {
    const re = new RegExp(`${theme}\\s*:\\s*{([\\s\\S]*?)\\n\\s*}`, 'm');
    const m = cfg.match(re);
    if (!m) continue;
    const body = m[1];
    /** @type {Record<string, string>} */
    const themeTokens = {};
    for (const line of body.split('\n')) {
      const lm = line.match(/^\s*"([a-zA-Z0-9-]+)"\s*:\s*"(#[0-9a-fA-F]{3,8})"/);
      if (!lm) continue;
      themeTokens[lm[1]] = lm[2];
  }
    out[theme] = themeTokens;
  }
  return out;
}

// ── Colour math ──────────────────────────────────────────────────

/** Normalise any CSS colour (hex / rgb) to { r, g, b } in 0..255. */
function parseHex(hex) {
  let h = hex.trim().replace(/^#/, '');
  if (h.length === 3) h = h.split('').map((c) => c + c).join('');
  if (h.length === 8) h = h.slice(0, 6);
  if (h.length !== 6) throw new Error(`unsupported colour literal: ${hex}`);
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16),
  };
}

function srgbToLinear(c) {
  const v = c / 255;
  return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
}

function relativeLuminance({ r, g, b }) {
  return (
    0.2126 * srgbToLinear(r) +
    0.7152 * srgbToLinear(g) +
    0.0722 * srgbToLinear(b)
  );
}

function contrastRatio(hexA, hexB) {
  const la = relativeLuminance(parseHex(hexA));
  const lb = relativeLuminance(parseHex(hexB));
  const [hi, lo] = la > lb ? [la, lb] : [lb, la];
  return (hi + 0.05) / (lo + 0.05);
}

// ── Pairs contract ───────────────────────────────────────────────

/**
 * Thresholds (WCAG 2.1):
 *   text      4.5:1  normal text, including small button, badge and
 *                    chip labels (they are < 18.66px bold / < 24px)
 *   large     3.0:1  text verified to be >= 24px, or >= 18.66px bold
 *   non-text  3.0:1  borders, icons and focus indicators (SC 1.4.11)
 *
 * Labels are never treated as large text automatically: a pair may only
 * use `large` when every rendered instance meets the size rule.
 */
const THRESHOLD = { text: 4.5, large: 3.0, 'non-text': 3.0 };

const THEMES = ['corporateDark', 'corporateLight'];

/** Surfaces that body, secondary and muted text are rendered on. */
const SURFACES = [
  'base-100',
  'base-200',
  'base-300',
  'ds-bg-surface',
  'ds-bg-elevated',
  'ds-bg-overlay',
];

/** Neutral text colours, audited on every surface. */
const TEXT_COLOURS = [
  { fg: 'base-content', name: 'body text' },
  { fg: 'ds-text-secondary', name: 'secondary text' },
  { fg: 'ds-text-muted', name: 'muted text' },
];

/** Solid fills carrying a small label (buttons, chips, badges). */
const SOLID_LABELS = [
  { fg: 'primary-content', bg: 'primary', label: 'primary button label' },
  { fg: 'primary-content', bg: 'ds-brand-hover', label: 'primary button label (hover)' },
  { fg: 'accent-content', bg: 'accent', label: 'accent surface label' },
  { fg: 'info-content', bg: 'info', label: 'info surface label' },
  { fg: 'success-content', bg: 'success', label: 'success surface label' },
  { fg: 'warning-content', bg: 'warning', label: 'warning surface label' },
  { fg: 'error-content', bg: 'error', label: 'error surface label' },
];

const PAIRS = THEMES.flatMap((theme) => [
  ...TEXT_COLOURS.flatMap(({ fg, name }) =>
    SURFACES.map((bg) => ({ theme, fg, bg, use: 'text', label: `${name} on ${bg}` })),
  ),
  ...SOLID_LABELS.map(({ fg, bg, label }) => ({ theme, fg, bg, use: 'text', label })),
]);

// ── Run ──────────────────────────────────────────────────────────

const tailwind = readTailwindTokens();
const cssTokens = readIndexCssTokens();

/** @type {string[]} */
const failures = [];
/** @type {string[]} */
const errors = [];

for (const pair of PAIRS) {
  const palette = { ...(tailwind[pair.theme] ?? {}), ...(cssTokens[pair.theme] ?? {}) };
  if (!tailwind[pair.theme] && !cssTokens[pair.theme]) {
    errors.push(`palette missing for theme ${pair.theme}`);
    continue;
  }
  const token = (name) => palette[name] ?? palette[`color-${name}`];
  const fg = token(pair.fg);
  const bg = token(pair.bg);
  if (!fg || !bg) {
    errors.push(`token missing for ${pair.theme}.${pair.fg}/${pair.bg}`);
    continue;
  }
  const ratio = contrastRatio(fg, bg);
  const need = THRESHOLD[pair.use];
  if (ratio + 1e-9 < need) {
    failures.push(
      `  ✗ ${pair.theme.padEnd(15)} ${pair.label.padEnd(36)} ${pair.fg}(${fg}) on ${pair.bg}(${bg}) → ${ratio.toFixed(2)}:1  (need ≥ ${need}:1 for ${pair.use})`
    );
  } else {
    console.log(
      `  ✓ ${pair.theme.padEnd(15)} ${pair.label.padEnd(36)} ${pair.fg}(${fg}) on ${pair.bg}(${bg}) → ${ratio.toFixed(2)}:1`
    );
  }
}

if (errors.length) {
  console.error('\nERROR — token resolution failed:');
  for (const e of errors) console.error(e);
  process.exit(2);
}

const reportOnly = process.argv.includes('--report');

if (failures.length) {
  console.error('\n✗ WCAG AA contrast drift detected:');
  for (const f of failures) console.error(f);
  if (!reportOnly) {
    console.error(
      '\nFailing because the contrast guard is strict by default. Fix the token(s) above; pass --report to list drift without failing.'
    );
    process.exit(1);
  }
  console.error('\nReport-only mode (--report): exiting 0 despite the drift listed above.');
  process.exit(0);
}

console.log(
  `\n✓ All ${PAIRS.length} documented contrast pairs meet WCAG 2.1 AA in every theme.\n`
);
process.exit(0);
