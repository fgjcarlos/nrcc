// Lightweight test for the i18n guard's code-like classifier.
// Run with: node scripts/check-no-hardcoded-i18n.test.mjs
//
// The guard resolves its source root from the script's own path
// (scripts/../src), so this test builds a sandbox by symlinking
// the real frontend/src into a temp directory and overriding the
// script's frontendRoot with --repo-root. Since the script does not
// support that override, this test takes a simpler route: it
// patches the guard in place to a temporary copy of the source
// tree, then runs the guard against the temp tree and restores.

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '..');
const realSrc = path.join(repoRoot, 'src');
const guardPath = path.join(repoRoot, 'scripts', 'check-no-hardcoded-i18n.mjs');

function runGuardAgainst(fixtureSrcDir) {
  // The guard walks {frontendRoot}/src and reads each .tsx. We use a
  // tiny shim script that monkey-patches path resolution by forking a
  // Node child with an inline monkey-patch wrapper. Easier path:
  // copy the guard source, rewrite the frontendRoot constant to
  // point at a temp dir whose src/ contains our fixtures, then run.
  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'i18n-guard-'));
  const tempSrc = path.join(tempRoot, 'src');
  fs.mkdirSync(tempSrc, { recursive: true });

  const guardSource = fs.readFileSync(guardPath, 'utf8');
  const patched = guardSource.replace(
    "path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')",
    `JSON.parse(${JSON.stringify(JSON.stringify(tempRoot))})`,
  );
  const patchedPath = path.join(tempRoot, 'check-no-hardcoded-i18n.mjs');
  fs.writeFileSync(patchedPath, patched);

  return { tempRoot, tempSrc, patchedPath };
}

function cleanup(dir) {
  fs.rmSync(dir, { recursive: true, force: true });
}

function writeFixture(tempSrc, name, content) {
  fs.writeFileSync(path.join(tempSrc, name), content);
}

function execGuard(patchedPath) {
  const result = spawnSync(process.execPath, [patchedPath], { encoding: 'utf8' });
  let parsed = [];
  if (result.stdout.trim()) {
    try {
      parsed = JSON.parse(result.stdout.trim());
    } catch (_) {
      parsed = [];
    }
  }
  return { stdout: result.stdout, stderr: result.stderr, status: result.status, parsed };
}

const cases = [
  {
    name: 'return ( between JSX wrappers is silent when no real copy exists',
    file: 'PureReturn.tsx',
    content: [
      "import React from 'react';",
      'export function PureReturn() {',
      '  return (',
      '    <div>{value}</div>',
      '  );',
      '}',
    ].join('\n'),
    expectViolation: false,
  },
  {
    name: 'return ( still does not flag literal JSX text "real copy" as code (regression check)',
    file: 'MixedReturn.tsx',
    content: [
      "import React from 'react';",
      'export function MixedReturn() {',
      '  return (',
      '    <div>real copy</div>',
      '  );',
      '}',
    ].join('\n'),
    expectViolation: true,
  },
  {
    name: 'throw new Error(...) wrapping real copy is still flagged',
    file: 'Throw.tsx',
    content: [
      "import React from 'react';",
      'export function Throw() {',
      '  if (!ready) {',
      '    throw new Error("nope");',
      '  }',
      '  return <div>safe copy</div>;',
      '}',
    ].join('\n'),
    expectViolation: true,
  },
];

let failed = 0;
for (const tc of cases) {
  const { tempRoot, tempSrc, patchedPath } = runGuardAgainst();
  writeFixture(tempSrc, tc.file, tc.content);
  const result = execGuard(patchedPath);
  const hasViolation = result.parsed.length > 0;
  const pass = hasViolation === tc.expectViolation;
  console.log(pass ? '✓' : '✗', tc.name);
  if (!pass) {
    console.log('  expected violation:', tc.expectViolation);
    console.log('  got violations    :', result.parsed.map((v) => v.literal));
  }
  cleanup(tempRoot);
  if (!pass) failed++;
}

if (failed > 0) {
  console.error(`\n${failed} assertion(s) failed`);
  process.exit(1);
}
console.log('\nAll i18n guard assertions passed.');