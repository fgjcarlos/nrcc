#!/usr/bin/env node
/**
 * check-contrast-tokens.test.mjs — issue #766 slice G W4
 *
 * Behavioural tests for scripts/check-contrast-tokens.mjs.
 *
 * Run with `node scripts/check-contrast-tokens.test.mjs`. Exits 0
 * when every assertion holds, 1 otherwise. The test exercises:
 *   1. happy path (all pairs AA) → exits 0
 *   2. drift in body pair → exits 0 in report-only, 1 in --strict
 *   3. missing token → exits 2 regardless of --strict
 *
 * The fixtures live in a temp directory so the test never mutates
 * the real index.css / tailwind.config.js. Re-running the test is
 * idempotent.
 */
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SCRIPT = resolve(__dirname, 'check-contrast-tokens.mjs');

/** Run the script in a temporary workdir with fixture CSS + tailwind. */
function runScript(workdir, extraArgs = []) {
  const result = spawnSync('node', [SCRIPT, '--repo-root', workdir, ...extraArgs], {
    cwd: workdir,
    encoding: 'utf8',
  });
  return { status: result.status ?? -1, stdout: result.stdout, stderr: result.stderr };
}

function writeFixture(workdir, { darkMuted, lightMuted }) {
  const css = `
[data-theme="corporateDark"] {
  --color-base-100: #07090d;
  --color-base-200: #0f1419;
  --color-base-300: #141b23;
  --color-base-content: #f5f7fa;
  --color-primary: #d4472b;
  --color-primary-content: #ffffff;
  --color-accent: #0089b4;
  --color-accent-content: #f5f7fa;
  --color-success: #16a36a;
  --color-success-content: #ffffff;
  --color-warning: #e8a811;
  --color-warning-content: #07090d;
  --color-error: #e54233;
  --color-error-content: #ffffff;
  --ds-text-muted: ${darkMuted};
}

[data-theme="corporateLight"] {
  --color-base-100: #f6f7f9;
  --color-base-200: #edf1f5;
  --color-base-300: #eef3f7;
  --color-base-content: #10151c;
  --color-primary: #bd3f27;
  --color-primary-content: #ffffff;
  --color-accent: #007aa0;
  --color-accent-content: #ffffff;
  --color-success: #0f8f5f;
  --color-success-content: #ffffff;
  --color-warning: #c98905;
  --color-warning-content: #10151c;
  --color-error: #c93429;
  --color-error-content: #ffffff;
  --ds-text-muted: ${lightMuted};
}
`;
  mkdirSync(join(workdir, 'src'), { recursive: true });
  writeFileSync(join(workdir, 'src/index.css'), css, 'utf8');

  const tw = `
export default {
  daisyui: {
    themes: [
      {
        corporateDark: {
          "base-100": "#07090d",
          "base-content": "#f5f7fa",
          "primary": "#d4472b",
          "primary-content": "#ffffff",
          "accent": "#0089b4",
          "accent-content": "#f5f7fa",
          "success": "#16a36a",
          "success-content": "#ffffff",
          "warning": "#e8a811",
          "warning-content": "#07090d",
          "error": "#e54233",
          "error-content": "#ffffff"
        },
      },
      {
        corporateLight: {
          "base-100": "#f6f7f9",
          "base-content": "#10151c",
          "primary": "#bd3f27",
          "primary-content": "#ffffff",
          "accent": "#007aa0",
          "accent-content": "#ffffff",
          "success": "#0f8f5f",
          "success-content": "#ffffff",
          "warning": "#c98905",
          "warning-content": "#10151c",
          "error": "#c93429",
          "error-content": "#ffffff"
        }
      }
    ]
  }
};
`;
  writeFileSync(join(workdir, 'tailwind.config.js'), tw, 'utf8');
}

let failures = 0;
function assert(cond, label) {
  if (cond) {
    console.log(`  ✓ ${label}`);
  } else {
    console.error(`  ✗ ${label}`);
    failures += 1;
  }
}

// ── Fixture 1: all pairs AA ──────────────────────────────────────
{
  const workdir = mkdtempSync(join(tmpdir(), 'contrast-test-'));
  try {
    // #6f7d90 on dark #07090d = 4.76:1 (passes body).
    // #5a6577 on light #f6f7f9 = 5.5:1 (passes body).
    writeFixture(workdir, { darkMuted: '#6f7d90', lightMuted: '#5a6577' });
    const { status } = runScript(workdir);
    assert(status === 0, 'all AA → exits 0 in report-only');
    const { status: strict } = runScript(workdir, ['--strict']);
    assert(strict === 0, 'all AA → exits 0 in --strict');
  } finally {
    rmSync(workdir, { recursive: true, force: true });
  }
}

// ── Fixture 2: light muted fails body AA ─────────────────────────
{
  const workdir = mkdtempSync(join(tmpdir(), 'contrast-test-'));
  try {
    // #6f7d90 on #f6f7f9 ≈ 3.91:1 (below 4.5).
    writeFixture(workdir, { darkMuted: '#6f7d90', lightMuted: '#6f7d90' });
    const { status, stdout } = runScript(workdir);
    assert(status === 0, 'drift present → exits 0 in report-only');
    assert(stdout.includes('muted text on base'), 'drift reported by name');
    const { status: strict } = runScript(workdir, ['--strict']);
    assert(strict === 1, 'drift present → exits 1 in --strict');
  } finally {
    rmSync(workdir, { recursive: true, force: true });
  }
}

// ── Fixture 3: missing token exits 2 ─────────────────────────────
{
  const workdir = mkdtempSync(join(tmpdir(), 'contrast-test-'));
  try {
    writeFixture(workdir, { darkMuted: '#5a6577', lightMuted: '#5a6577' });
    // Replace the daisyUI 'primary' key with a typo to simulate drift.
    const twPath = join(workdir, 'tailwind.config.js');
    const tw = readFileSync(twPath, 'utf8');
    writeFileSync(twPath, tw.replace('"primary-content": "#ffffff"', '"primary-contentx": "#ffffff"'), 'utf8');
    const cssPath = join(workdir, 'src/index.css');
    const css = readFileSync(cssPath, 'utf8');
    writeFileSync(cssPath, css.replaceAll('--color-primary-content:', '--color-primary-contentx:'), 'utf8');
    const { status } = runScript(workdir, ['--strict']);
    assert(status === 2, 'missing token → exits 2 regardless of --strict');
  } finally {
    rmSync(workdir, { recursive: true, force: true });
  }
}

// ── Fixture 4: calculator identity check on known values ─────────
{
  const workdir = mkdtempSync(join(tmpdir(), 'contrast-test-'));
  try {
    writeFixture(workdir, { darkMuted: '#5a6577', lightMuted: '#5a6577' });
    const { stdout } = runScript(workdir);
    // base-content (#f5f7fa) on base-100 (#07090d) in corporateDark
    assert(stdout.includes('18.57:1'), 'calculator reports 18.57:1 for body on base (dark)');
  } finally {
    rmSync(workdir, { recursive: true, force: true });
  }
}

// ── Fixture 5: DaisyUI v5 CSS variable names are authoritative ────
{
  const workdir = mkdtempSync(join(tmpdir(), 'contrast-test-'));
  try {
    writeFixture(workdir, { darkMuted: '#6f7d90', lightMuted: '#5a6577' });
    writeFileSync(join(workdir, 'tailwind.config.js'), 'export default {};', 'utf8');
    const { status, stdout, stderr } = runScript(workdir);
    assert(status === 0, `CSS palette tokens resolve without legacy DaisyUI config (${stderr})`);
    assert(stdout.includes('All 14 documented contrast pairs'), 'CSS palette resolves all documented pairs');
  } finally {
    rmSync(workdir, { recursive: true, force: true });
  }
}

if (failures > 0) {
  console.error(`\n✗ ${failures} test assertion(s) failed.`);
  process.exit(1);
}
console.log('\n✓ All contrast script assertions passed.');
process.exit(0);