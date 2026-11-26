# Brand mark A2 (solid title-block grid)

Source: UI/UX critique 2026-11-26 (`odd/audits/ui-ux-2026-11-26/`), logo
exploration `logos-a.html`, option A2 chosen by the user.

The mark becomes the drawing title block: a 2×2 grid with N R / C C,
knocked out of a solid ink tile. Letters are drawn as stroke paths (no
font dependency; the app must work offline).

## Tasks

- [x] `NrccMark.tsx`: replace the closed-contact glyph with the A2 tile;
      knockout via an SVG mask with a per-instance id (`useId`) so the
      letters show the ground on any surface; drop the container border
      and background (the glyph is the tile).
- [x] `NrccMark.test.tsx`: replace closed-contact geometry tests with A2
      geometry tests (32-unit grid, 2×2 cells, mask knockout).
- [x] `frontend/public/favicon.svg` + `docs/assets/favicon.svg`: A2 with
      explicit colours; invert under `prefers-color-scheme: dark`.
- [x] `docs/assets/nrcc-mark.svg`: A2, ink tile, drawing-white letters.
- [x] Styleguide `Foundations.tsx` copy and `DESIGN.md` "The mark" section.
- [x] Verify: `npm test -- --run` (NrccMark + styleguide), typecheck, lint.

## Verification (2026-11-26)

- `npx vitest --run`: 92 files, 744 passed, 1 skipped.
- `tsc --noEmit`: ok. `eslint` on touched files: ok.
- `i18n:check`, `test:appearance`, `test:css`, `check:contrast`: pass.
- Not done: visual check in a browser; e2e (`test:e2e`) not run.
