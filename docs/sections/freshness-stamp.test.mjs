// Tests for the freshness stamp regex used by the GitHub Pages
// freshness checker. Run with: node docs/sections/freshness-stamp.test.mjs
//
// Asserts:
//   1. The stamp shape is `data-freshness-stamp="<token>"` and tokens
//      are lowercase alphanumeric + dash (matches the slice-h plan).
//   2. The current docs/index.html uses only allowlisted tokens.
//   3. No stale TBD/WIP/empty stamp appears anywhere in docs/.

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '..', '..');
const indexHtml = path.join(repoRoot, 'docs', 'index.html');
const docsRoot = path.join(repoRoot, 'docs');

const allowedStamps = new Set(['slice-h']);

const stampRegex = /data-freshness-stamp="([^"]*)"/g;

function assert(condition, message) {
  if (!condition) {
    console.error('✗', message);
    process.exitCode = 1;
    return false;
  }
  console.log('✓', message);
  return true;
}

// 1. Token shape contract: lowercase, alnum + dash, non-empty.
const tokenShape = /^[a-z0-9][a-z0-9-]*$/;
const samples = ['slice-h', 'slice-i', 'release-v1', 'release-v2-3'];
for (const sample of samples) {
  assert(tokenShape.test(sample), `token shape accepts "${sample}"`);
}
for (const bad of ['', 'Slice-H', 'WIP', 'TBD', 'slice h', 'slice_h']) {
  assert(!tokenShape.test(bad), `token shape rejects "${bad}"`);
}

// 2. docs/index.html only uses allowlisted stamps.
const indexText = fs.readFileSync(indexHtml, 'utf8');
const usedStamps = [...indexText.matchAll(stampRegex)].map((m) => m[1]);
assert(usedStamps.length > 0, `docs/index.html declares at least one freshness stamp (got ${usedStamps.length})`);
for (const stamp of usedStamps) {
  assert(
    allowedStamps.has(stamp),
    `docs/index.html stamp "${stamp}" is allowlisted (allowed: ${[...allowedStamps].join(', ')})`,
  );
}

// 3. No stale TBD / WIP / empty stamps anywhere in docs/.
function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(full));
    } else if (entry.isFile() && (entry.name.endsWith('.html') || entry.name.endsWith('.md'))) {
      files.push(full);
    }
  }
  return files;
}

let foundStale = false;
for (const file of walk(docsRoot)) {
  const text = fs.readFileSync(file, 'utf8');
  for (const match of text.matchAll(stampRegex)) {
    const stamp = match[1];
    if (stamp === '' || /^(TBD|WIP|tbd|wip)$/.test(stamp)) {
      console.error('✗', `stale freshness stamp "${stamp}" in ${path.relative(repoRoot, file)}`);
      foundStale = true;
    }
  }
}
if (!foundStale) {
  console.log('✓', 'no stale TBD/WIP/empty freshness stamps in docs/');
} else {
  process.exitCode = 1;
}

// 4. The stamp policy is documented in docs/sections/screenshots.md so
// the next contributor knows how to retire placeholders.
const screenshotsPath = path.join(here, 'screenshots.md');
if (fs.existsSync(screenshotsPath)) {
  const screenshots = fs.readFileSync(screenshotsPath, 'utf8');
  assert(
    /data-freshness-stamp/.test(screenshots),
    'docs/sections/screenshots.md documents the data-freshness-stamp policy',
  );
}