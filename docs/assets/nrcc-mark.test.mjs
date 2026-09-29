// Tests for the NRCC mark SVG asset.
// Run with: node docs/assets/nrcc-mark.test.mjs
//
// Asserts:
//   1. The SVG file exists and is non-empty.
//   2. It declares an accessible role/aria-label.
//   3. It uses the canonical cyan + graphite tokens (#0089b4 + #07090d)
//      so the public site and the application share one visual identity.
//   4. The favicon uses the same glyph (favicon.svg exists).

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '..', '..');

const markPath = path.join(here, 'nrcc-mark.svg');
const faviconPath = path.join(here, 'favicon.svg');
const indexPath = path.join(repoRoot, 'docs', 'index.html');

function assert(condition, message) {
  if (!condition) {
    console.error('✗', message);
    process.exitCode = 1;
    return false;
  }
  console.log('✓', message);
  return true;
}

const mark = fs.readFileSync(markPath, 'utf8');
const favicon = fs.readFileSync(faviconPath, 'utf8');

assert(mark.length > 0, 'nrcc-mark.svg exists and is non-empty');
assert(favicon.length > 0, 'favicon.svg exists and is non-empty');
assert(/role="img"/.test(mark) && /aria-label="NRCC mark"/.test(mark), 'nrcc-mark.svg declares role=img + aria-label');
assert(/role="img"/.test(favicon) && /aria-label="NRCC"/.test(favicon), 'favicon.svg declares role=img + aria-label');
assert(/#07090d/i.test(mark), 'nrcc-mark.svg uses the graphite void token (#07090d)');
assert(/#0089b4/i.test(mark), 'nrcc-mark.svg uses the cyan accent token (#0089b4)');
assert(/#07090d/i.test(favicon), 'favicon.svg uses the graphite void token (#07090d)');
assert(/#0089b4/i.test(favicon), 'favicon.svg uses the cyan accent token (#0089b4)');

// The two glyphs must share the same geometry so the mark and the
// favicon look identical. We compare the <g> body after stripping
// comments so a future prose tweak doesn't drift the visuals.
function extractGlyphBody(svg) {
  const noComments = svg.replace(/<!--[\s\S]*?-->/g, '');
  return noComments.match(/<g [^>]*>[\s\S]*?<\/g>/)?.[0] ?? '';
}
const markGlyph = extractGlyphBody(mark);
const faviconGlyph = extractGlyphBody(favicon);
assert(markGlyph.length > 0 && markGlyph === faviconGlyph, 'nrcc-mark.svg and favicon.svg share the same glyph body');

// docs/index.html references both assets at the expected paths.
const index = fs.readFileSync(indexPath, 'utf8');
assert(/assets\/nrcc-mark\.svg/.test(index), 'docs/index.html references assets/nrcc-mark.svg');
assert(/assets\/favicon\.svg/.test(index), 'docs/index.html references assets/favicon.svg');
