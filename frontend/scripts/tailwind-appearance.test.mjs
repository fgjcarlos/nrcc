#!/usr/bin/env node
/**
 * Computed-style regression check for the production Tailwind/daisyUI stylesheet.
 * Run with `pnpm test:appearance` from frontend.
 */
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, readdirSync, rmSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { chromium } from '@playwright/test';

const root = resolve(fileURLToPath(new URL('..', import.meta.url)));
const outDir = mkdtempSync(join(root, '.tailwind-appearance-'));
const vite = join(root, 'node_modules/vite/bin/vite.js');
let browser;
try {
  const build = spawnSync(process.execPath, [vite, 'build', '--outDir', outDir, '--emptyOutDir'], {
    cwd: root,
    encoding: 'utf8',
  });
  const buildOutput = `${build.stdout ?? ''}\n${build.stderr ?? ''}`;
  assert.equal(build.status, 0, `Vite production CSS build failed:\n${buildOutput}`);
  assert.doesNotMatch(buildOutput, /css-syntax-error|\[object Object\]/i, `Malformed CSS diagnostics:\n${buildOutput}`);

  const cssFile = readdirSync(join(outDir, 'assets')).find((file) => file.endsWith('.css'));
  assert.ok(cssFile, 'Vite emitted a production CSS asset');
  const css = readFileSync(join(outDir, 'assets', cssFile), 'utf8');
  browser = await chromium.launch({ headless: true });
  // Esquema eléctrico elevation: flat surface token, hairline border and at
  // most a 0 1px 2px shadow; no gradient, glass blur or wide glow.
  const surface = { corporateDark: 'rgb(23, 26, 31)', corporateLight: 'rgb(251, 250, 247)' };
  const maxBlur = (shadow) =>
    shadow === 'none' ? 0 : Math.max(0, ...[...shadow.matchAll(/(-?[\d.]+)px (-?[\d.]+)px ([\d.]+)px/g)].map((m) => Number(m[3])));
  const assertFlat = (computed, label) => {
    assert.equal(computed.backdropFilter, 'none', `${label} has no glass blur`);
    assert.equal(computed.backgroundImage, 'none', `${label} has no gradient`);
    assert.ok(maxBlur(computed.boxShadow) <= 2, `${label} shadow blur ≤ 2px, got ${computed.boxShadow}`);
  };

  for (const theme of ['corporateDark', 'corporateLight']) {
    for (const viewport of [{ width: 1440, height: 900 }, { width: 390, height: 844 }]) {
      const context = await browser.newContext({ viewport, reducedMotion: 'reduce' });
      try {
        const page = await context.newPage();
        await page.setContent(`<style>${css}</style><main data-theme="${theme}"><section id="panel" class="glass-panel p-5"><button id="primary" class="action-btn-primary">Primary</button><button id="secondary" class="action-btn-secondary">Secondary</button><button id="ghost" class="action-btn-ghost">Ghost</button><button id="danger" class="action-btn-danger">Danger</button></section><button id="disabled-primary" class="action-btn-primary" disabled>Disabled</button><button id="disabled-secondary" class="action-btn-secondary" disabled>Disabled</button><button id="disabled-ghost" class="action-btn-ghost" disabled>Disabled</button><button id="disabled-danger" class="action-btn-danger" disabled>Disabled</button><button id="busy" class="action-btn-primary" aria-busy="true">Saving</button><span id="spinner" class="loading loading-spinner"></span></main>`);
        const style = (id) => page.locator(`#${id}`).evaluate((el) => {
          const computed = getComputedStyle(el);
          return Object.fromEntries(['backdropFilter', 'backgroundColor', 'backgroundImage', 'boxShadow', 'cursor', 'border', 'borderRadius', 'padding', 'height', 'outlineStyle', 'transitionDuration', 'opacity', 'animationDuration'].map((key) => [key, computed[key]]));
        });
        const panel = await style('panel');
        assertFlat(panel, `${theme} ${viewport.width}px panel`);
        assert.equal(panel.backgroundColor, surface[theme], `${theme} panel uses the surface token`);
        assert.match(panel.border, /^1px solid /);
        assert.equal(panel.borderRadius, '4px');
        assert.equal(panel.padding, '20px');

        const hoverColors = [];
        for (const [id, padding] of [['primary', '8px 16px'], ['secondary', '8px 16px'], ['ghost', '8px 12px'], ['danger', '8px 16px']]) {
          const before = await style(id);
          assert.equal(before.cursor, 'pointer', `${theme} ${id} active cursor`);
          assertFlat(before, `${theme} ${id}`);
          assert.equal(before.backgroundColor, surface[theme], `${theme} ${id} uses the surface token`);
          assert.equal(before.borderRadius, '4px');
          assert.equal(before.padding, padding);
          assert.equal(before.height, '42px');
          assert.match(before.border, /^1px solid /);
          await page.locator(`#${id}`).hover();
          const hovered = await style(id);
          assert.notEqual(hovered.backgroundColor, 'rgba(0, 0, 0, 0)', `${theme} ${id} keeps its hover treatment`);
          hoverColors.push(hovered.backgroundColor);
        }
        assert.equal(new Set(hoverColors).size, 4, `${theme} preserves distinct action hover colors`);

        for (const id of ['primary', 'secondary', 'ghost', 'danger']) {
          await page.keyboard.press('Tab');
          assert.equal(await page.evaluate(() => document.activeElement.id), id);
          assert.notEqual((await style(id)).outlineStyle, 'none', `${id} keyboard focus remains visible`);
        }
        for (const id of ['primary', 'secondary', 'ghost', 'danger']) {
          const disabled = await style(`disabled-${id}`);
          assert.equal(await page.locator(`#disabled-${id}`).isDisabled(), true);
          assert.equal(disabled.cursor, 'default');
          assert.equal(disabled.opacity, '0.5');
        }
        assert.equal(await page.locator('#busy').getAttribute('aria-busy'), 'true');
        assert.equal(await page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches), true);
        assert.ok(Number.parseFloat((await style('spinner')).animationDuration) <= 0.01, 'reduced motion limits spinner animation');
      } finally {
        await context.close();
      }
    }
  }
  console.log(`Computed appearance passed in Chromium ${await browser.version()} (2 themes × 2 viewports, reduced motion).`);
} finally {
  try {
    if (browser) await browser.close();
  } finally {
    rmSync(outDir, { recursive: true, force: true });
  }
}
