// Tests for the docs/sections/*.md source files.
// Run with: node docs/sections/index.test.mjs
//
// Asserts:
//   1. Every section file is non-empty.
//   2. The roadmap section mirrors docs/control-plane.md exactly:
//        - 9 cluster rows
//        - 6 GREEN, 3 AMBER (0 RED)
//        - All 9 issue numbers present (#756..#764)
//   3. The compatibility section lists 5.x (full), 4.x (read-only),
//      and >=6.0 (read-only) versions.
//   4. The limitations section lists at least the documented non-goals:
//      flow editor, cluster orchestrator, npm library browser,
//      Node-RED 4 read-only, no Docker socket, local backups only.

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '..', '..');

function assert(condition, message) {
  if (!condition) {
    console.error('✗', message);
    process.exitCode = 1;
    return false;
  }
  console.log('✓', message);
  return true;
}

const expected = [
  'overview.md',
  'configuration.md',
  'security.md',
  'recovery.md',
  'quick-start.md',
  'roadmap.md',
  'limitations.md',
  'compatibility.md',
  'screenshots.md',
];

for (const name of expected) {
  const file = path.join(here, name);
  const text = fs.readFileSync(file, 'utf8');
  assert(text.length > 0, `${name} exists and is non-empty (${text.length} chars)`);
}

// Roadmap mirrors control-plane.md: 9 rows, 6 GREEN, 3 AMBER.
// We count table rows (markdown table rows start with `|`) whose cells
// contain a cluster issue link. This avoids counting the word
// "Overall" in the summary line.
function countRoadmapRows(text) {
  const rows = text.split('\n').filter((line) => /^\s*\|.*\/(75[6-9]|76[0-4])\b/.test(line));
  const green = rows.filter((line) => /\bGREEN\b/.test(line)).length;
  const amber = rows.filter((line) => /\bAMBER\b/.test(line)).length;
  const red = rows.filter((line) => /\bRED\b/.test(line)).length;
  const issues = rows.map((line) => line.match(/\/(75[6-9]|76[0-4])\b/)[1]);
  return { rows: rows.length, green, amber, red, issues };
}

const roadmap = fs.readFileSync(path.join(here, 'roadmap.md'), 'utf8');
const counts = countRoadmapRows(roadmap);
assert(counts.rows === 9, `roadmap.md has 9 cluster rows (got ${counts.rows})`);
assert(new Set(counts.issues).size === 9, `roadmap.md issue numbers are unique (got ${new Set(counts.issues).size})`);
assert(counts.green === 6, `roadmap.md shows 6 GREEN clusters (got ${counts.green})`);
assert(counts.amber === 3, `roadmap.md shows 3 AMBER clusters (got ${counts.amber})`);
assert(counts.red === 0, `roadmap.md shows 0 RED clusters (got ${counts.red})`);

// Cross-check the roadmap against the canonical control-plane.md so a
// future drift in either file fails CI.
const controlPlane = fs.readFileSync(path.join(repoRoot, 'docs', 'control-plane.md'), 'utf8');
const cpCounts = countRoadmapRows(controlPlane);
assert(cpCounts.rows === 9, `docs/control-plane.md has 9 cluster rows (got ${cpCounts.rows})`);
assert(cpCounts.green === counts.green, `roadmap GREEN count matches docs/control-plane.md (cp=${cpCounts.green}, rd=${counts.green})`);
assert(cpCounts.amber === counts.amber, `roadmap AMBER count matches docs/control-plane.md (cp=${cpCounts.amber}, rd=${counts.amber})`);
assert(cpCounts.red === counts.red, `roadmap RED count matches docs/control-plane.md (cp=${cpCounts.red}, rd=${counts.red})`);

// Compatibility covers the three version branches.
const compatibility = fs.readFileSync(path.join(here, 'compatibility.md'), 'utf8');
assert(/>=5\.0\s*<\s*6\.0/.test(compatibility), 'compatibility.md documents >=5.0 <6.0 as full editing');
assert(/\b4\.x\b/.test(compatibility), 'compatibility.md documents 4.x as read-only');
assert(/>=6\.0/.test(compatibility), 'compatibility.md documents >=6.0 as read-only');

// Limitations lists the documented non-goals.
const limitations = fs.readFileSync(path.join(here, 'limitations.md'), 'utf8');
const nonGoals = [
  'flow editor',
  'cluster orchestrator',
  'npm library browser',
  'Node-RED 4',
  'Docker socket',
  'local backups',
];
for (const needle of nonGoals) {
  assert(
    limitations.toLowerCase().includes(needle.toLowerCase()),
    `limitations.md mentions the non-goal: ${needle}`,
  );
}