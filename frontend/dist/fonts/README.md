# Berkeley Mono Fonts

This directory contains Berkeley Mono typeface files in `.woff2` format for the Node-RED Control Center frontend redesign.

## Files Required

The following files must be placed in this directory:

- `BerkeleyMono-Regular.woff2` (weight 400)
- `BerkeleyMono-Medium.woff2` (weight 500)
- `BerkeleyMono-Bold.woff2` (weight 700)

## Acquisition

Berkeley Mono is a commercial typeface available for purchase at:
https://berkeleygraphics.com/typefaces/berkeley-mono/

## Fallback Chain

If Berkeley Mono files are not available, the CSS fallback chain will use:
1. IBM Plex Mono
2. ui-monospace
3. Menlo, Monaco, Consolas
4. System monospace

This ensures the frontend remains functional even without the primary font.

## Font Loading

Fonts are loaded via:
1. HTML preload links in `index.html` (for early fetching)
2. CSS `@font-face` declarations in `src/styles.css`

See `src/styles.css` for `@font-face` rules.

## Status (Phase 1)

- [ ] BerkeleyMono-Regular.woff2 placed
- [ ] BerkeleyMono-Medium.woff2 placed
- [ ] BerkeleyMono-Bold.woff2 placed
- [ ] Fonts verified in browser DevTools

As of 2026-04-15, these files have NOT been acquired. Phase 1 implementation assumes they will be provided externally.
