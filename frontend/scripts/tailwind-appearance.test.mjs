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
  const ground = { corporateDark: 'rgb(17, 19, 23)', corporateLight: 'rgb(246, 245, 240)' };
  const sunken = { corporateDark: 'rgb(12, 14, 17)', corporateLight: 'rgb(238, 237, 230)' };
  const raised = { corporateDark: 'rgb(29, 33, 40)', corporateLight: 'rgb(255, 255, 255)' };
  const controlBorder = { corporateDark: 'rgb(107, 113, 124)', corporateLight: 'rgb(131, 136, 145)' };
  const primary = { corporateDark: 'rgb(127, 166, 234)', corporateLight: 'rgb(43, 95, 180)' };
  const accent = primary;
  const critical = { corporateDark: 'rgb(240, 138, 128)', corporateLight: 'rgb(179, 38, 30)' };
  // WCAG relative-luminance contrast between two opaque rgb() strings.
  const luminance = (rgb) => {
    const [r, g, b] = rgb.match(/\d+(\.\d+)?/g).slice(0, 3).map((v) => {
      const c = Number(v) / 255;
      return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
    });
    return 0.2126 * r + 0.7152 * g + 0.0722 * b;
  };
  const contrast = (a, b) => {
    const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
    return (hi + 0.05) / (lo + 0.05);
  };
  assert.ok(Math.abs(contrast('rgb(0, 0, 0)', 'rgb(255, 255, 255)') - 21) < 0.01, 'contrast helper self-check');
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
        await page.setContent(`<style>${css}</style><main data-theme="${theme}" class="bg-base-100"><section id="panel" class="glass-panel p-5"><button id="primary" class="action-btn-primary">Primary</button><button id="secondary" class="action-btn-secondary">Secondary</button><button id="ghost" class="action-btn-ghost">Ghost</button><button id="danger" class="action-btn-danger">Danger</button><button id="solid" class="action-btn-solid">Solid</button><button id="daisy-primary" class="btn btn-primary">Solid</button><button id="daisy-secondary" class="btn btn-secondary">Secondary</button><button id="daisy-ghost" class="btn btn-ghost">Ghost</button><button id="daisy-error" class="btn btn-error">Danger</button><button id="daisy-xs" class="btn btn-ghost btn-xs">Small</button></section><button id="disabled-primary" class="action-btn-primary" disabled>Disabled</button><button id="disabled-secondary" class="action-btn-secondary" disabled>Disabled</button><button id="disabled-ghost" class="action-btn-ghost" disabled>Disabled</button><button id="disabled-danger" class="action-btn-danger" disabled>Disabled</button><button id="disabled-solid" class="action-btn-solid" disabled>Disabled</button><button id="disabled-daisy" class="btn btn-primary" disabled>Disabled</button><button id="busy" class="action-btn-primary" aria-busy="true">Saving</button><span id="spinner" class="loading loading-spinner"></span><section id="fields" class="bg-ds-bg-surface p-5"><input id="field" class="block w-full px-3 py-2 text-sm" aria-label="Shared input"><input id="glass-field" class="glass-panel h-12 w-full rounded-xl border-0 px-3" aria-label="Glass input"><select id="select-field" aria-label="Select"><option>One</option></select><textarea id="textarea-field" aria-label="Textarea"></textarea><input id="invalid-field" aria-invalid="true" aria-label="Invalid"><input id="disabled-field" disabled aria-label="Disabled"><textarea id="readonly-field" readonly aria-label="Read-only"></textarea><input id="bare-field" class="input-bare border-b border-border" aria-label="Bare"><input id="check-field" type="checkbox" class="checkbox" aria-label="Check"></section></main>`);
        const style = (id) => page.locator(`#${id}`).evaluate((el) => {
          const computed = getComputedStyle(el);
          return Object.fromEntries(['backdropFilter', 'backgroundColor', 'backgroundImage', 'boxShadow', 'cursor', 'border', 'borderColor', 'borderTopWidth', 'borderTopStyle', 'borderRadius', 'padding', 'height', 'minHeight', 'display', 'gap', 'fontSize', 'color', 'outlineStyle', 'outlineColor', 'transitionDuration', 'opacity', 'animationDuration'].map((key) => [key, computed[key]]));
        });
        const panel = await style('panel');
        assertFlat(panel, `${theme} ${viewport.width}px panel`);
        assert.equal(panel.backgroundColor, surface[theme], `${theme} panel uses the surface token`);
        assert.match(panel.border, /^1px solid /);
        assert.equal(panel.borderRadius, '4px');
        assert.equal(panel.padding, '20px');

        // One button spec (DESIGN.md Buttons): 4px field radius, inline-flex,
        // 8px gap, 14px label, 36px minimum target, 1px edge.
        const transparent = 'rgba(0, 0, 0, 0)';
        const restGround = { primary: surface[theme], secondary: surface[theme], ghost: transparent, danger: surface[theme] };
        const hoverColors = [];
        for (const [id, padding] of [['primary', '8px 16px'], ['secondary', '8px 16px'], ['ghost', '8px 12px'], ['danger', '8px 16px'], ['solid', '8px 16px']]) {
          const before = await style(id);
          assert.equal(before.cursor, 'pointer', `${theme} ${id} active cursor`);
          assertFlat(before, `${theme} ${id}`);
          if (restGround[id]) assert.equal(before.backgroundColor, restGround[id], `${theme} ${id} rest ground`);
          assert.equal(before.borderRadius, '4px');
          assert.equal(before.padding, padding);
          assert.equal(before.display, 'inline-flex', `${theme} ${id} is inline-flex`);
          assert.equal(before.gap, '8px', `${theme} ${id} gap`);
          assert.equal(before.fontSize, '14px', `${theme} ${id} label size`);
          assert.equal(before.minHeight, '36px', `${theme} ${id} minimum target`);
          assert.ok(Number.parseFloat(before.height) >= 36, `${theme} ${id} height ≥ 36px, got ${before.height}`);
          assert.match(before.border, /^1px solid /);
          await page.locator(`#${id}`).hover();
          const hovered = await style(id);
          assert.notEqual(hovered.backgroundColor, before.backgroundColor, `${theme} ${id} keeps its hover treatment`);
          hoverColors.push(hovered.backgroundColor);
        }
        assert.equal(new Set(hoverColors).size, 5, `${theme} preserves distinct action hover colors`);

        // Raw daisyUI .btn call sites converge on the same values.
        for (const [daisy, action, keys] of [
          ['daisy-primary', 'solid', ['backgroundColor', 'color', 'borderColor', 'borderRadius', 'fontSize']],
          ['daisy-secondary', 'secondary', ['backgroundColor', 'color', 'borderColor', 'borderRadius', 'fontSize']],
          ['daisy-ghost', 'ghost', ['backgroundColor', 'color', 'borderRadius', 'fontSize']],
          ['daisy-error', 'danger', ['backgroundColor', 'color', 'borderColor', 'borderRadius', 'fontSize']],
        ]) {
          await page.mouse.move(0, 0);
          const [a, b] = [await style(daisy), await style(action)];
          for (const key of keys) assert.equal(a[key], b[key], `${theme} .${daisy} ${key} matches .action-btn-${action}`);
          assert.equal(a.height, '36px', `${theme} ${daisy} md height`);
          assert.equal(a.boxShadow, 'none', `${theme} ${daisy} is flat`);
        }
        const xs = await style('daisy-xs');
        assert.equal(xs.height, '24px', `${theme} btn-xs keeps a 24px target`);
        assert.equal(xs.fontSize, '12px', `${theme} btn-xs uses the 12px label size`);

        for (const id of ['primary', 'secondary', 'ghost', 'danger', 'solid', 'daisy-primary']) {
          await page.keyboard.press('Tab');
          assert.equal(await page.evaluate(() => document.activeElement.id), id);
          const focused = await style(id);
          assert.equal(focused.outlineStyle, 'solid', `${id} keyboard focus remains visible`);
          assert.equal(focused.outlineColor, accent[theme], `${id} focus uses the conductor-blue ring`);
        }
        for (const id of ['primary', 'secondary', 'ghost', 'danger', 'solid', 'daisy']) {
          const disabled = await style(`disabled-${id}`);
          assert.equal(await page.locator(`#disabled-${id}`).isDisabled(), true);
          assert.equal(disabled.opacity, '0.5');
          if (id !== 'daisy') assert.equal(disabled.cursor, 'default');
        }

        // Text controls (DESIGN.md Inputs): every shared, raw and glass-panel
        // field shows a 1px control border at the 4px field radius that
        // clears 3:1 against the grounds it sits on (WCAG 1.4.11, L11).
        const grounds = [surface[theme], ground[theme], sunken[theme]];
        for (const id of ['field', 'glass-field', 'select-field', 'textarea-field']) {
          const field = await style(id);
          assert.equal(field.borderTopWidth, '1px', `${theme} ${id} border width`);
          assert.equal(field.borderTopStyle, 'solid', `${theme} ${id} border style`);
          assert.equal(field.borderRadius, '4px', `${theme} ${id} field radius`);
          assert.equal(field.borderColor, controlBorder[theme], `${theme} ${id} uses the control border`);
          assert.equal(field.backgroundColor, raised[theme], `${theme} ${id} raised ground`);
          for (const bg of grounds) {
            const ratio = contrast(field.borderColor, bg);
            assert.ok(ratio >= 3, `${theme} ${id} border ${field.borderColor} on ${bg} = ${ratio.toFixed(2)}:1 (needs 3:1)`);
          }
        }
        await page.mouse.move(0, 0);
        await page.locator('#field').hover();
        assert.notEqual((await style('field')).borderColor, controlBorder[theme], `${theme} field hover changes the edge`);
        await page.locator('#field').focus();
        const focusedField = await style('field');
        assert.equal(focusedField.borderColor, primary[theme], `${theme} field focus turns the border blue`);
        assert.equal(focusedField.outlineStyle, 'solid', `${theme} field keyboard focus outline`);
        assert.equal(focusedField.outlineColor, accent[theme]);
        const invalid = await style('invalid-field');
        assert.equal(invalid.borderColor, critical[theme], `${theme} aria-invalid uses the critical border`);
        assert.ok(contrast(invalid.borderColor, surface[theme]) >= 3, `${theme} invalid border clears 3:1`);
        assert.equal((await style('disabled-field')).opacity, '0.5', `${theme} disabled field`);
        assert.equal((await style('readonly-field')).backgroundColor, sunken[theme], `${theme} read-only field sits on the sunken ground`);
        const bare = await style('bare-field');
        assert.equal(bare.borderTopWidth, '0px', `${theme} .input-bare opts out of the box`);
        assert.equal((await style('check-field')).borderRadius, '2px', `${theme} checkbox keeps the selector radius`);

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
