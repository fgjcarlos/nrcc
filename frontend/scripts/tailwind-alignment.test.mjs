#!/usr/bin/env node
/**
 * Build-level regression check for the Tailwind/daisyUI CSS pipeline.
 * Run with `node scripts/tailwind-alignment.test.mjs` from frontend.
 */
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, readdirSync, rmSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = resolve(fileURLToPath(new URL('..', import.meta.url)));
const outDir = mkdtempSync(join(root, '.tailwind-alignment-build-'));
const vite = resolve(root, 'node_modules/vite/bin/vite.js');
try {
  const result = spawnSync(process.execPath, [vite, 'build', '--outDir', outDir, '--emptyOutDir'], {
    cwd: root,
    encoding: 'utf8',
  });
  const output = `${result.stdout ?? ''}\n${result.stderr ?? ''}`;
  assert.equal(result.status, 0, `Vite CSS build failed:\n${output}`);
  assert.doesNotMatch(output, /css-syntax-error|\[object Object\]/i, `CSS diagnostics detected:\n${output}`);

  const cssFile = readdirSync(join(outDir, 'assets')).find((file) => file.endsWith('.css'));
  assert.ok(cssFile, 'Vite build emitted a CSS asset');
  const css = readFileSync(join(outDir, 'assets', cssFile), 'utf8');
  for (const [label, pattern] of [
    ['dark corporate theme tokens', /data-theme=["']?corporateDark["']?/],
    ['light corporate theme tokens', /data-theme=["']?corporateLight["']?/],
    // Esquema eléctrico: neutral-conductor blue is the action colour.
    ['dark theme palette', /--color-primary:#7fa6ea/],
    ['light theme palette', /--color-primary:#2b5fb4/],
    ['glass panel styling', /\.glass-panel/],
    ['primary button utility', /\.btn-primary/],
    ['input control styling', /\.input\{/],
    ['select control styling', /\.select\{/],
    ['toggle control styling', /\.toggle\{/],
    ['loading indicator styling', /\.loading\{/],
    ['disabled control state', /:disabled/],
    ['visible focus state', /:focus-visible/],
    // One button spec: the action classes carry the 36px DESIGN.md target.
    ['action button minimum height', /\.action-btn-primary[^{]*\{[^}]*min-height:2\.25rem/],
    // Legacy glow utilities now resolve to the flat raised elevation.
    ['flattened shadow utility', /\.shadow-glow[,{][^}]*--tw-shadow:var\(--ds-shadow-raised\)/],
    ['DaisyUI box radius token', /--radius-box:4px/],
    ['DaisyUI field radius token', /--radius-field:4px/],
    ['DaisyUI selector radius token', /--radius-selector:2px/],
    ['self-hosted Overpass face', /@font-face\{font-family:Overpass Variable/],
    ['self-hosted Overpass Mono face', /@font-face\{font-family:Overpass Mono/],
    ['body set in Overpass', /font-family:Overpass Variable,system-ui,sans-serif/],
    ['DaisyUI field size token', /--size-field:\.25rem/],
    // #965: semantic text reads the readable on-tint token; fills keep --color-*.
    // Tailwind only emits used utilities, so check every semantic text class the
    // build actually contains (text-primary is always used by the shell).
    ...['primary', 'error', 'success', 'warning', 'info', 'accent']
      .filter((colour) => colour === 'primary' || css.includes(`.text-${colour}{`))
      .map((colour) => [
        `text-${colour} on --ds-on-tint-${colour}`,
        new RegExp(`\\.text-${colour}\\{color:var\\(--ds-on-tint-${colour}\\)\\}`),
      ]),
    ['bg-primary keeps the solid fill', /\.bg-primary[,{][^}]*background-color:var\(--color-primary\)/],
  ]) {
    assert.match(css, pattern, `Built CSS is missing ${label}`);
  }
  // Offline (PRODUCT.md) and the retired Signal Prime world: no font CDN,
  // no Inter/JetBrains faces, no old brand hex in the stylesheet.
  for (const [label, pattern] of [
    ['font CDN request', /fonts\.(googleapis|gstatic)\.com/],
    ['legacy font family', /font-family:[^;}]*(Inter|JetBrains)/],
    ['legacy Signal Prime hex', /#(d4472b|cf452a|bd3f27|0089b4|007aa0|e54233)\b/i],
    // DESIGN.md has three type sizes; the retired ds-* scale must not return.
    ['retired ds-* type scale', /\.text-ds-(display|h1|h2|h3|body|caption|eyebrow)\{/],
    // Text controls get their boundary without !important (L19 cleanup).
    ['!important on the text control rule', /\[data-theme\] (input|textarea|select)[^{]*\{[^}]*!important/],
  ]) {
    assert.doesNotMatch(css, pattern, `Built CSS still contains ${label}`);
  }
  const html = readFileSync(join(outDir, 'index.html'), 'utf8');
  assert.doesNotMatch(html, /fonts\.(googleapis|gstatic)\.com/, 'index.html still requests a font CDN');
  console.log('Tailwind/daisyUI production CSS contract passed.');
} finally {
  rmSync(outDir, { recursive: true, force: true });
}
