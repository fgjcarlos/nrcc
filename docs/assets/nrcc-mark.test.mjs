// Tests for the NRCC mark SVG asset.
// Run with: node docs/assets/nrcc-mark.test.mjs
//
// Asserts:
//   1. The SVG file exists and is non-empty.
//   2. It declares an accessible role/aria-label.
//   3. It draws with the app's light-theme tokens (ink #15171c on drawing
//      white #f6f5f0), checked against
//      frontend/src/index.css, so the public site and the application
//      share one visual identity. No legacy Signal Prime colours remain.
//   4. The mark, the docs favicon and the app favicon share one glyph:
//      the closed IEC contact (two open terminal rings, conductor into
//      each, a flat blade between them heavier than the conductor) on a
//      16-unit grid, with strokes >= 1.5 so it reads at 16px.

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '..', '..');

const markPath = path.join(here, 'nrcc-mark.svg');
const faviconPath = path.join(here, 'favicon.svg');
const indexPath = path.join(repoRoot, 'docs', 'index.html');
const appFaviconPath = path.join(repoRoot, 'frontend', 'public', 'favicon.svg');
const tokensPath = path.join(repoRoot, 'frontend', 'src', 'index.css');

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
// Colour checks ignore comments so prose cannot satisfy them.
const stripComments = (svg) => svg.replace(/<!--[\s\S]*?-->/g, '');
const markBody = stripComments(mark);
const faviconBody = stripComments(favicon);
const TOKENS = [
  ['ink', '--ds-text-primary', '#15171c'],
  ['drawing white', '--ds-bg-void', '#f6f5f0'],
];
const tokensCss = fs.readFileSync(tokensPath, 'utf8');
for (const [name, token, hex] of TOKENS) {
  assert(new RegExp(`${token}:\\s*${hex};`, 'i').test(tokensCss), `frontend/src/index.css defines ${name} as ${token}: ${hex}`);
  assert(markBody.toLowerCase().includes(`"${hex}"`), `nrcc-mark.svg uses the ${name} token (${hex})`);
  assert(faviconBody.toLowerCase().includes(`"${hex}"`), `favicon.svg uses the ${name} token (${hex})`);
}
for (const legacy of ['#07090d', '#0089b4']) {
  assert(!mark.toLowerCase().includes(legacy) && !favicon.toLowerCase().includes(legacy), `no legacy Signal Prime colour ${legacy} remains`);
}
assert(!/signal prime/i.test(mark) && !/signal prime/i.test(favicon), 'no Signal Prime reference remains in the SVG assets');

// The glyphs must share the same geometry so the mark and both
// favicons look identical. We compare the <g> body after stripping
// comments so a future prose tweak doesn't drift the visuals.
function extractGlyphBody(svg) {
  const noComments = svg.replace(/<!--[\s\S]*?-->/g, '');
  return noComments.match(/<g [^>]*>[\s\S]*?<\/g>/)?.[0] ?? '';
}
const markGlyph = extractGlyphBody(mark);
const faviconGlyph = extractGlyphBody(favicon);
assert(markGlyph.length > 0 && markGlyph === faviconGlyph, 'nrcc-mark.svg and favicon.svg share the same glyph body');
const appFaviconGlyph = extractGlyphBody(fs.readFileSync(appFaviconPath, 'utf8'));
assert(markGlyph === appFaviconGlyph, 'frontend/public/favicon.svg shares the same glyph body');

// Closed contact geometry on a 16-unit square grid.
for (const [name, body] of [['nrcc-mark.svg', markBody], ['favicon.svg', faviconBody]]) {
  assert(/viewBox="0 0 16 16"/.test(body), `${name} draws on a 16-unit square grid`);
}
const attrs = (tag) => Object.fromEntries([...tag.matchAll(/([\w-]+)="([^"]*)"/g)].map(([, k, v]) => [k, v]));
const groupAttrs = attrs(markGlyph.match(/<g [^>]*>/)?.[0] ?? '');
const shapes = (name) => [...markGlyph.matchAll(new RegExp(`<${name} [^>]*>`, 'g'))].map(([tag]) => {
  const a = attrs(tag);
  return Object.fromEntries(Object.entries({ ...groupAttrs, ...a }).map(([k, v]) => [k, Number.isNaN(Number(v)) ? v : Number(v)]));
});
const rings = shapes('circle').sort((a, b) => a.cx - b.cx);
const lines = shapes('line');
assert(groupAttrs.fill === 'none', 'the glyph is unfilled, so the terminal rings stay open on any ground');
assert(!/<rect /.test(markGlyph), 'the glyph carries no square indicator');
assert(rings.length === 2 && rings.every((r) => r.cy === 8 && r.r === rings[0].r), 'two equal terminal rings sit on the centre line (y=8)');
assert(lines.length === 3 && lines.every((l) => l.y1 === 8 && l.y2 === 8), 'three horizontal pieces on the centre line: two conductor runs and the blade');
assert([...rings, ...lines].every((s) => s['stroke-width'] >= 1.5), 'every stroke is >= 1.5 units, legible at 16px');
const [left, right] = rings;
const leftWire = lines.find((l) => l.x1 === 0);
const rightWire = lines.find((l) => l.x2 === 16);
const blade = lines.find((l) => left && l.x1 === left.cx + left.r);
assert(leftWire?.x2 === left?.cx - left?.r && rightWire?.x1 === right?.cx + right?.r, 'the conductor runs in from each edge and ends under a terminal ring');
assert(blade?.x2 === right?.cx - right?.r, 'the blade lies flat on the line, bridging the two terminals');
assert(blade !== undefined && leftWire !== undefined && blade['stroke-width'] > leftWire['stroke-width'], 'the blade is drawn heavier than the conductor');

// docs/index.html references both assets at the expected paths.
const index = fs.readFileSync(indexPath, 'utf8');
assert(/assets\/nrcc-mark\.svg/.test(index), 'docs/index.html references assets/nrcc-mark.svg');
assert(/assets\/favicon\.svg/.test(index), 'docs/index.html references assets/favicon.svg');
