#!/usr/bin/env node
/**
 * check-contrast-tokens.test.mjs — issues #766 slice G W4, #889 and #965
 *
 * Behavioural tests for scripts/check-contrast-tokens.mjs.
 *
 * Run with `pnpm test:contrast` (or `node scripts/check-contrast-tokens.test.mjs`).
 * Exits 0 when every assertion holds, 1 otherwise. The test exercises:
 *   1. happy path (every pair AA) → exits 0 in the default and --report modes
 *   2. text drift → exits 1 by default (strict), 0 with --report
 *   3. small button labels are text (4.5:1), not UI components (3:1)
 *   4. text is audited on every documented surface, not only base-100,
 *      and chip/badge labels on their 18% tint over every surface
 *   4b. semantic text (`text-primary`, `text-accent`, ...) is audited
 *      on every surface through its readable `ds-on-tint-*` token (#965)
 *   5. missing token → exits 2 in every mode
 *   6. calculator identity check on known values
 *
 * The fixtures live in a temp directory so the test never mutates the
 * real index.css / tailwind.config.js. Re-running the test is idempotent.
 */
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SCRIPT = resolve(__dirname, 'check-contrast-tokens.mjs');

/** Every documented pair resolves to an AA-passing value in this palette. */
const PASSING = {
  corporateDark: {
    'color-base-100': '#07090d',
    'color-base-200': '#0f1419',
    'color-base-300': '#141b23',
    'color-base-content': '#f5f7fa',
    'color-primary': '#1d4ed8',
    'color-primary-content': '#ffffff',
    'color-accent': '#1d4ed8',
    'color-accent-content': '#ffffff',
    'color-info': '#1d4ed8',
    'color-info-content': '#ffffff',
    'color-success': '#1d4ed8',
    'color-success-content': '#ffffff',
    'color-warning': '#e8a811',
    'color-warning-content': '#07090d',
    'color-error': '#1d4ed8',
    'color-error-content': '#ffffff',
    'ds-bg-surface': '#141b23',
    'ds-bg-elevated': '#1a2332',
    'ds-bg-overlay': '#223044',
    'ds-text-secondary': '#d8dee8',
    'ds-text-muted': '#9aa7b8',
    'ds-brand-hover': '#1e40af',
    'ds-on-tint-primary': '#f5f7fa',
    'ds-on-tint-success': '#f5f7fa',
    'ds-on-tint-info': '#f5f7fa',
    'ds-on-tint-warning': '#f5f7fa',
    'ds-on-tint-error': '#f5f7fa',
    'ds-on-tint-accent': '#f5f7fa',
  },
  corporateLight: {
    'color-base-100': '#f6f7f9',
    'color-base-200': '#edf1f5',
    'color-base-300': '#eef3f7',
    'color-base-content': '#10151c',
    'color-primary': '#1d4ed8',
    'color-primary-content': '#ffffff',
    'color-accent': '#1d4ed8',
    'color-accent-content': '#ffffff',
    'color-info': '#1d4ed8',
    'color-info-content': '#ffffff',
    'color-success': '#1d4ed8',
    'color-success-content': '#ffffff',
    'color-warning': '#c98905',
    'color-warning-content': '#10151c',
    'color-error': '#1d4ed8',
    'color-error-content': '#ffffff',
    'ds-bg-surface': '#ffffff',
    'ds-bg-elevated': '#f9fafb',
    'ds-bg-overlay': '#f0f4fb',
    'ds-text-secondary': '#263241',
    'ds-text-muted': '#5a6577',
    'ds-brand-hover': '#1e40af',
    'ds-on-tint-primary': '#10151c',
    'ds-on-tint-success': '#10151c',
    'ds-on-tint-info': '#10151c',
    'ds-on-tint-warning': '#10151c',
    'ds-on-tint-error': '#10151c',
    'ds-on-tint-accent': '#10151c',
  },
};

/** Write src/index.css (+ an empty tailwind.config.js) from PASSING plus overrides. */
function writeFixture(workdir, overrides = {}) {
  const blocks = Object.entries(PASSING).map(([theme, tokens]) => {
    const merged = { ...tokens, ...(overrides[theme] ?? {}) };
    const lines = Object.entries(merged)
      .filter(([, value]) => value !== undefined)
      .map(([name, value]) => `  --${name}: ${value};`);
    return `[data-theme="${theme}"] {\n${lines.join('\n')}\n}\n`;
  });
  mkdirSync(join(workdir, 'src'), { recursive: true });
  writeFileSync(join(workdir, 'src/index.css'), blocks.join('\n'), 'utf8');
  writeFileSync(join(workdir, 'tailwind.config.js'), 'export default {};\n', 'utf8');
}

/** Run the script against a fixture workdir. */
function runScript(workdir, extraArgs = []) {
  const result = spawnSync('node', [SCRIPT, '--repo-root', workdir, ...extraArgs], {
    cwd: workdir,
    encoding: 'utf8',
  });
  return { status: result.status ?? -1, stdout: result.stdout, stderr: result.stderr };
}

function withFixture(overrides, fn) {
  const workdir = mkdtempSync(join(tmpdir(), 'contrast-test-'));
  try {
    writeFixture(workdir, overrides);
    fn(workdir);
  } finally {
    rmSync(workdir, { recursive: true, force: true });
  }
}

let failures = 0;
function assert(cond, label, detail = '') {
  if (cond) {
    console.log(`  ✓ ${label}`);
  } else {
    console.error(`  ✗ ${label}${detail ? `\n${detail}` : ''}`);
    failures += 1;
  }
}

// ── 1. Every pair AA ─────────────────────────────────────────────
withFixture({}, (workdir) => {
  const run = runScript(workdir);
  assert(run.status === 0, 'all AA → exits 0 by default', run.stderr);
  assert(/All \d+ documented contrast pairs meet WCAG 2\.1 AA/.test(run.stdout), 'all AA → success summary');
  assert(run.stderr === '', 'all AA → nothing on stderr', run.stderr);
  const report = runScript(workdir, ['--report']);
  assert(report.status === 0, 'all AA → exits 0 with --report', report.stderr);
});

// ── 2. Strict by default; --report is the only non-failing mode ──
withFixture({ corporateLight: { 'ds-text-muted': '#6f7d90' } }, (workdir) => {
  // #6f7d90 on #f6f7f9 ≈ 3.91:1 (below 4.5).
  const run = runScript(workdir);
  assert(run.status === 1, 'text drift → exits 1 by default', `status=${run.status}`);
  assert(
    run.stderr.includes('corporateLight') && run.stderr.includes('muted text on base-100'),
    'text drift → names the theme and pair on stderr',
    run.stderr,
  );
  assert(run.stderr.includes('3.91:1') && run.stderr.includes('need ≥ 4.5:1 for text'), 'text drift → ratio and threshold reported', run.stderr);
  const strict = runScript(workdir, ['--strict']);
  assert(strict.status === 1, 'text drift → --strict is accepted and still exits 1', `status=${strict.status}`);
  const report = runScript(workdir, ['--report']);
  assert(report.status === 0, 'text drift → exits 0 with --report', `status=${report.status}`);
  assert(report.stderr.includes('muted text on base-100'), 'text drift → --report still lists the drift', report.stderr);
  assert(report.stderr.includes('Report-only mode'), 'text drift → --report announces report-only mode', report.stderr);
});

// ── 3. Small button labels use the text threshold ────────────────
withFixture({ corporateDark: { 'color-primary': '#d4472b' } }, (workdir) => {
  // White on #d4472b ≈ 4.43:1: passes 3:1 (UI) but fails 4.5:1 (text).
  const run = runScript(workdir);
  assert(run.status === 1, 'small label at 4.43:1 → exits 1', `status=${run.status}`);
  assert(
    run.stderr.includes('primary button label') && run.stderr.includes('need ≥ 4.5:1 for text'),
    'small label → judged against the 4.5:1 text threshold',
    run.stderr,
  );
});

withFixture({ corporateDark: { 'ds-brand-hover': '#f05a3d' } }, (workdir) => {
  // White on #f05a3d ≈ 3.37:1 — the hovered primary button label.
  const run = runScript(workdir);
  assert(run.status === 1, 'hovered label drift → exits 1', `status=${run.status}`);
  assert(run.stderr.includes('primary button label (hover)'), 'hovered label → named on stderr', run.stderr);
});

// ── 4. Text is audited on every documented surface ───────────────
withFixture({ corporateDark: { 'ds-text-muted': '#6f7d90' } }, (workdir) => {
  // #6f7d90 passes on base-100 (4.76:1) but fails on surface (4.14:1)
  // and overlay (3.18:1).
  const run = runScript(workdir);
  assert(run.status === 1, 'muted text failing on cards only → exits 1', `status=${run.status}`);
  assert(run.stderr.includes('muted text on ds-bg-surface'), 'surface pair reported', run.stderr);
  assert(run.stderr.includes('muted text on ds-bg-overlay'), 'overlay pair reported', run.stderr);
  assert(!run.stderr.includes('muted text on base-100'), 'passing base-100 pair is not reported as drift', run.stderr);
});

withFixture({ corporateLight: { 'ds-on-tint-warning': '#c98905' } }, (workdir) => {
  // Amber text on its own 18% tint over white ≈ 2.4:1.
  const run = runScript(workdir);
  assert(run.status === 1, 'tinted chip label drift → exits 1', `status=${run.status}`);
  assert(
    run.stderr.includes('warning chip label on ds-bg-surface') && run.stderr.includes('warning 18% tint over ds-bg-surface'),
    'tinted chip label → names the composite background',
    run.stderr,
  );
});

// ── 4b. Semantic text is audited on every surface (#965) ─────────
withFixture({ corporateDark: { 'ds-on-tint-primary': '#cf452a' } }, (workdir) => {
  // #cf452a on #223044 ≈ 2.88:1 — the old dark text-primary colour.
  const run = runScript(workdir);
  assert(run.status === 1, 'semantic text drift → exits 1', `status=${run.status}`);
  assert(run.stderr.includes('primary text on ds-bg-overlay'), 'semantic text → plain-surface pair reported', run.stderr);
  assert(run.stderr.includes('primary text on base-100'), 'semantic text → base-100 pair reported', run.stderr);
});

withFixture({}, (workdir) => {
  const { stdout } = runScript(workdir);
  for (const colour of ['primary', 'error', 'success', 'warning', 'info', 'accent']) {
    assert(stdout.includes(`${colour} text on ds-bg-surface`), `semantic ${colour} text pair audited`);
  }
  assert(stdout.includes('accent chip label on ds-bg-surface'), 'accent chip label on its tint audited');
});

withFixture({ corporateLight: { 'ds-on-tint-accent': undefined } }, (workdir) => {
  const run = runScript(workdir);
  assert(run.status === 2, 'missing accent text token → exits 2', `status=${run.status}`);
  assert(run.stderr.includes('ds-on-tint-accent'), 'missing accent text token named', run.stderr);
});

// ── 5. Missing token exits 2 in every mode ───────────────────────
withFixture({ corporateLight: { 'ds-bg-overlay': undefined } }, (workdir) => {
  for (const args of [[], ['--report'], ['--strict']]) {
    const run = runScript(workdir, args);
    assert(run.status === 2, `missing token → exits 2 (${args.join(' ') || 'default'})`, `status=${run.status}`);
    assert(run.stderr.includes('corporateLight') && run.stderr.includes('ds-bg-overlay'), `missing token named (${args.join(' ') || 'default'})`, run.stderr);
  }
});

// ── 6. Calculator identity check ─────────────────────────────────
withFixture({}, (workdir) => {
  const { stdout } = runScript(workdir);
  // base-content (#f5f7fa) on base-100 (#07090d) in corporateDark.
  assert(stdout.includes('18.57:1'), 'calculator reports 18.57:1 for body text on base-100 (dark)');
});

if (failures > 0) {
  console.error(`\n✗ ${failures} test assertion(s) failed.`);
  process.exit(1);
}
console.log('\n✓ All contrast script assertions passed.');
process.exit(0);
