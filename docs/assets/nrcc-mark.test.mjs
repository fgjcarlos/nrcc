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
//      the drawing title block (a solid ink tile split into a 2x2 grid
//      that spells N R / C C in drawing white) on a 32-unit grid, with
//      letters drawn as stroke paths (no font) and strokes >= 1.5.
//   5. Both favicons invert under prefers-color-scheme: dark.

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
// Repeat until stable so a nested '<!-<!---->-' cannot leave a comment behind.
const stripComments = (svg) => {
  let previous;
  do {
    previous = svg;
    svg = svg.replace(/<!--[\s\S]*?-->/g, '');
  } while (svg !== previous);
  return svg;
};
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
// favicons look identical. We compare the tile plus the knockout group
// after stripping comments so a future prose tweak doesn't drift the
// visuals.
function extractGlyphBody(svg) {
  const body = stripComments(svg);
  const from = body.indexOf('<rect class="tile"');
  const to = body.lastIndexOf('</g>');
  return from >= 0 && to > from ? body.slice(from, to + 4) : '';
}
const markGlyph = extractGlyphBody(mark);
const faviconGlyph = extractGlyphBody(favicon);
const appFavicon = fs.readFileSync(appFaviconPath, 'utf8');
assert(markGlyph.length > 0 && markGlyph === faviconGlyph, 'nrcc-mark.svg and favicon.svg share the same glyph body');
assert(markGlyph === extractGlyphBody(appFavicon), 'frontend/public/favicon.svg shares the same glyph body');

// Title-block geometry on a 32-unit square grid.
for (const [name, body] of [['nrcc-mark.svg', markBody], ['favicon.svg', faviconBody]]) {
  assert(/viewBox="0 0 32 32"/.test(body), `${name} draws on a 32-unit square grid`);
}
const attrs = (tag) => Object.fromEntries([...tag.matchAll(/([\w-]+)="([^"]*)"/g)].map(([, k, v]) => [k, Number.isNaN(Number(v)) ? v : Number(v)]));
const tile = attrs(markGlyph.match(/<rect [^>]*>/)?.[0] ?? '');
const cut = attrs(markGlyph.match(/<g [^>]*>/)?.[0] ?? '');
const rules = [...markGlyph.matchAll(/<line [^>]*>/g)].map(([tag]) => attrs(tag));
const letters = [...markGlyph.matchAll(/<path [^>]*>/g)].map(([tag]) => attrs(tag));
const letterStroke = attrs(markGlyph.match(/<g [^>]*>[\s\S]*?(<g [^>]*>)/)?.[1] ?? '');
assert(tile.width === 32 && tile.height === 32 && tile.fill === '#15171c', 'a solid ink tile fills the whole grid');
assert(cut.fill === 'none' && cut.stroke === '#f6f5f0', 'letters and rules are drawn-white strokes cut into the tile');
const horizontal = rules.find((l) => l.y1 === l.y2);
const vertical = rules.find((l) => l.x1 === l.x2);
assert(rules.length === 2 && horizontal?.y1 === 16 && vertical?.x1 === 16, 'two rules cross at the centre, splitting the tile into a 2x2 grid');
assert(letters.length === 4, 'four letter paths, one per cell: N R / C C');
assert(!/<text[\s>]/.test(markGlyph), 'letters are stroke paths, never text, so the mark needs no font');
assert(rules.every((l) => l['stroke-width'] >= 1.5) && letterStroke['stroke-width'] >= 1.5, 'every stroke is >= 1.5 units');
assert(!/<circle /.test(markGlyph), 'no terminal rings from the retired contact mark remain');
for (const [name, svg] of [['favicon.svg', favicon], ['frontend/public/favicon.svg', appFavicon]]) {
  assert(/@media \(prefers-color-scheme: dark\)/.test(stripComments(svg)), `${name} inverts on dark browser tabs`);
}

// docs/index.html references both assets at the expected paths.
const index = fs.readFileSync(indexPath, 'utf8');
assert(/assets\/nrcc-mark\.svg/.test(index), 'docs/index.html references assets/nrcc-mark.svg');
assert(/assets\/favicon\.svg/.test(index), 'docs/index.html references assets/favicon.svg');
