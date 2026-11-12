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
 *   3. dark-theme values that must render identically to the previous
 *      hard-coded Tailwind hex still do.
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
 * Tailwind key → previous dark hex. These keys must keep rendering
 * exactly the same colour in the dark theme after the move to var().
 */
const DARK_UNCHANGED = {
  'ds-brand-primary': '#d4472b',
  'ds-brand-dim': '#8f2f1f',
  'ds-accent-primary': '#0089b4',
  'ds-accent-hover': '#22b4df',
  'ds-accent-dim': '#075a73',
  'ds-bg-void': '#07090d',
  'ds-bg-base': '#0f1419',
  'ds-bg-surface': '#141b23',
  'ds-bg-elevated': '#1a2332',
  'ds-bg-overlay': '#223044',
  'ds-border-subtle': '#263241',
  'ds-border-default': '#39485c',
  'ds-border-strong': '#4a5f7c',
  'ds-text-primary': '#f5f7fa',
  'ds-text-secondary': '#d8dee8',
  'ds-text-tertiary': '#9aa7b8',
  'ds-success': '#16a36a',
  'ds-warning': '#e8a811',
  'ds-danger': '#e54233',
  'ds-info': '#2c9edb',
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

// ── 3. Dark rendering unchanged for the pinned keys ──────────────
console.log('\ndark theme keeps the previous Tailwind colours');
for (const [key, expected] of Object.entries(DARK_UNCHANGED)) {
  const name = varOf[key];
  const actual = name ? cssTokens.corporateDark?.[name] : undefined;
  assert(actual === expected, `${key} dark = ${expected}`, `resolved ${name ?? '(no var)'} = ${actual ?? '(undeclared)'}`);
}

if (failures > 0) {
  console.error(`\n✗ ${failures} test assertion(s) failed.`);
  process.exit(1);
}
console.log('\n✓ All tailwind ds-* variable assertions passed.');
process.exit(0);
