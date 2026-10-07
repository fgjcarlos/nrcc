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
    ['dark theme palette', /--color-primary:#d4472b/],
    ['light theme palette', /--color-primary:#bd3f27/],
    ['glass panel styling', /\.glass-panel/],
    ['primary button utility', /\.btn-primary/],
    ['input control styling', /\.input\{/],
    ['select control styling', /\.select\{/],
    ['toggle control styling', /\.toggle\{/],
    ['loading indicator styling', /\.loading\{/],
    ['disabled control state', /:disabled/],
    ['visible focus state', /:focus-visible/],
    ['custom typography utility', /\.text-ds-h1\{/],
    ['custom shadow utility', /\.shadow-glow\{/],
    ['DaisyUI field radius token', /--radius-field:\.25rem/],
    ['DaisyUI field size token', /--size-field:\.25rem/],
  ]) {
    assert.match(css, pattern, `Built CSS is missing ${label}`);
  }
  console.log('Tailwind/daisyUI production CSS contract passed.');
} finally {
  rmSync(outDir, { recursive: true, force: true });
}
