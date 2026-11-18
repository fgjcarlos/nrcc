#!/usr/bin/env node
/**
 * tailwind-ds-vars.test.mjs — issue #889 (T3, S8)
 *
 * Static guard for the theme-aware Tailwind `ds-*` palette.
 *
 * Run with `pnpm test:tailwind-vars` (or `node scripts/tailwind-ds-vars.test.mjs`).
 * Exits 0 when every assertion holds, 1 otherwise. Checks:
 *   1. every `'ds-…'` colour in tailwind.config.js is `var(--name)`
 *      (no hard-coded hex literal);
 *   2. every referenced `--name` is declared in both the
 *      `corporateDark` and `corporateLight` blocks of src/index.css;
 *   3. both themes resolve to the Esquema eléctrico palette.
 *
 * Both files are read as text; no JavaScript is evaluated.
 */
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');
const THEMES = ['corporateDark', 'corporateLight'];

/**
 * Tailwind key → expected hex per theme (Esquema eléctrico, ui-brand-overhaul
 * T3). Pins the palette so a token edit is a deliberate, reviewed change.
 */
const EXPECTED = {
  corporateDark: {
    'ds-brand-primary': '#7fa6ea',
    'ds-brand-dim': '#24375a',
    'ds-accent-primary': '#7fa6ea',
    'ds-accent-hover': '#9ab9ef',
    'ds-accent-dim': '#24375a',
    'ds-bg-void': '#111317',
    'ds-bg-base': '#0c0e11',
    'ds-bg-surface': '#171a1f',
    'ds-bg-elevated': '#1d2128',
    'ds-bg-overlay': '#1d2128',
    'ds-border-subtle': '#2a2f37',
    'ds-border-default': '#6b717c',
    'ds-border-strong': '#9a9ea6',
    'ds-text-primary': '#e9e7df',
    'ds-text-secondary': '#c3c1b8',
    'ds-text-tertiary': '#9a9ea6',
    'ds-success': '#8cc66b',
    'ds-warning': '#e8b04a',
    'ds-danger': '#f08a80',
    'ds-info': '#c99a6e',
  },
  corporateLight: {
    'ds-brand-primary': '#2b5fb4',
    'ds-brand-dim': '#dbe4f3',
    'ds-accent-primary': '#2b5fb4',
    'ds-accent-hover': '#234e94',
    'ds-accent-dim': '#dbe4f3',
    'ds-bg-void': '#f6f5f0',
    'ds-bg-base': '#eeede6',
    'ds-bg-surface': '#fbfaf7',
    'ds-bg-elevated': '#ffffff',
    'ds-bg-overlay': '#ffffff',
    'ds-border-subtle': '#d6d4cb',
    'ds-border-default': '#838891',
    'ds-border-strong': '#5c626c',
    'ds-text-primary': '#15171c',
    'ds-text-secondary': '#3d424b',
    'ds-text-tertiary': '#5c626c',
    'ds-success': '#2d6a22',
    'ds-warning': '#8a5a00',
    'ds-danger': '#b3261e',
    'ds-info': '#7a4a2a',
  },
};

let failures = 0;
function assert(cond, label, detail = '') {
  if (cond) {
    console.log(`  ✓ ${label}`);
  } else {
    console.error(`  ✗ ${label}${detail ? `\n    ${detail}` : ''}`);
    failures += 1;
  }
}

// ── Parse tailwind.config.js colours (text only) ─────────────────
const cfg = readFileSync(resolve(ROOT, 'tailwind.config.js'), 'utf8');
const colorsBlock = cfg.match(/colors\s*:\s*{([\s\S]*?)\n\s*}/);
/** @type {Record<string, string>} */
const twColors = {};
if (colorsBlock) {
  for (const line of colorsBlock[1].split('\n')) {
    const m = line.match(/^\s*'(ds-[a-z0-9-]+)'\s*:\s*'([^']+)'/);
    if (m) twColors[m[1]] = m[2].trim();
  }
}

// ── Parse src/index.css theme blocks (same rules as the auditor) ─
const css = readFileSync(resolve(ROOT, 'src/index.css'), 'utf8');
/** @type {Record<string, Record<string, string>>} */
const cssTokens = {};
const blockRegex = /\[data-theme="([^"]+)"\]\s*{([\s\S]*?)\n\s*}/g;
let block;
while ((block = blockRegex.exec(css)) !== null) {
  const vars = (cssTokens[block[1]] ??= {});
  for (const line of block[2].split('\n')) {
    const m = line.match(/^\s*(--[a-z0-9-]+)\s*:\s*([^;]+);/i);
    if (m) vars[m[1]] = m[2].trim().toLowerCase();
  }
}

// ── 1. Every ds-* colour is var(--name) ──────────────────────────
console.log('ds-* colours resolve through CSS variables');
assert(Object.keys(twColors).length > 0, 'tailwind.config.js declares ds-* colours');
/** @type {Record<string, string>} */
const varOf = {};
for (const [key, value] of Object.entries(twColors)) {
  const m = value.match(/^var\((--[a-z0-9-]+)\)$/);
  assert(Boolean(m), `${key} is var(--…)`, `found '${value}'`);
  if (m) varOf[key] = m[1];
}

// ── 2. Each var is declared in both themes ───────────────────────
console.log('\nreferenced variables are declared in both themes');
for (const theme of THEMES) {
  assert(Boolean(cssTokens[theme]), `src/index.css has a [data-theme="${theme}"] block`);
}
for (const [key, name] of Object.entries(varOf)) {
  for (const theme of THEMES) {
    assert(Boolean(cssTokens[theme]?.[name]), `${key} → ${name} declared in ${theme}`);
  }
}

// ── 3. Both themes resolve to the pinned palette ─────────────────
console.log('\nboth themes resolve to the Esquema eléctrico palette');
for (const theme of THEMES) {
  for (const [key, expected] of Object.entries(EXPECTED[theme])) {
    const name = varOf[key];
    const actual = name ? cssTokens[theme]?.[name] : undefined;
    assert(actual === expected, `${key} ${theme} = ${expected}`, `resolved ${name ?? '(no var)'} = ${actual ?? '(undeclared)'}`);
  }
}

if (failures > 0) {
  console.error(`\n✗ ${failures} test assertion(s) failed.`);
  process.exit(1);
}
console.log('\n✓ All tailwind ds-* variable assertions passed.');
process.exit(0);
