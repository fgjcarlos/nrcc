# UI/UX and brand overhaul (Impeccable)

Rebrand the NRCC frontend with the Impeccable skill: product context, new visual
world, shared primitives, an in-app dev-only style guide, then surface-by-surface
application. Branch: `feat/ui-brand-overhaul`.

## Specs

- **S1** — "me gustaria mejorar la estetica, cambiar la ui/ux de la app, le falta marca, estilo y falla en muchas cosas en el uso para el usuario."
- **S2** — "Falta estilos, paleta de colores, tipografia etc, en fin un storybook de marca."
- **S3** — "Crea un plan para aplicar esta skill a lo largo de las necesidades del proyecto" (skill: Impeccable, installed at `.pi/skills/impeccable/`).
- **S4** — Decision 1, "opcion 2": full rebrand. New visual world (palette, typography, logo); the current "Signal Prime" look is evidence and anti-reference only. Product truth, content, functions and constraints are preserved.
- **S5** — Decision 2, "opcion 2": the brand "storybook" is a dev-only in-app `/styleguide` route (no Storybook dependency), covering foundations (color, type, spacing, motion, voice, logo) and components.

## Tasks

- [ ] T1 — S1,S3 — Phase 0: `/impeccable init` → `PRODUCT.md` (interview) — inline — commit: _pending_
- [ ] T2 — S1 — Phase 1: rendered diagnosis: `critique` per surface + `audit` + `detect <dev-server URL>` → prioritized P0–P3 backlog — inline — commit: _pending_
- [ ] T3 — S2,S4 — Phase 2: new visual world (`new-work`), `DESIGN.md` (`document`), tokens rewritten in `index.css`, self-hosted fonts, logo/wordmark; retire stale `COLOR_REFERENCE.md` — inline — commit: _pending_
- [ ] T4 — S2,S4 — Phase 3a: `extract` shared primitives (Button, Input/Select/Textarea, Card, Dialog, Table, Tabs, Badge/StatusChip, feedback states) — route TBD — commit: _pending_
- [ ] T5 — S5 — Phase 3b: dev-only `/styleguide` route (foundations + components, both themes, en/es) — route TBD — commit: _pending_
- [ ] T6 — S1 — Phase 4: app chrome (sidebar, header, command palette) + i18n leaks — commit: _pending_
- [ ] T7 — S1 — Phase 4: Login / Setup (`onboard`, `clarify`, `harden`) — commit: _pending_
- [ ] T8 — S1 — Phase 4: Overview (`layout`, `colorize`, `typeset`) — commit: _pending_
- [ ] T9 — S1 — Phase 4: Configuration (`harden`, `clarify`, `layout`) — commit: _pending_
- [ ] T10 — S1 — Phase 4: Backups, Env vars, Security (`polish`, `harden`) — commit: _pending_
- [ ] T11 — S1 — Phase 4: Maintenance (updates, libraries) and Users (`polish`) — commit: _pending_
- [ ] T12 — S1 — Phase 4: `adapt` (responsive) and `animate` pass — commit: _pending_
- [ ] T13 — S1 — Phase 5: guardrails: `impeccable detect --github` in CI, final `critique`/`audit` vs T2 baseline — commit: _pending_
- [ ] T14 — S1 — Phase 6 (needs user OK): remove unrouted `features/flows`, `features/files`, `LandingView` — commit: _pending_

## Log

- **L1** — User (original request): "me gustaria mejorar la estetica, cambiar la ui/ux de la app, le falta marca, estilo y falla en muchas cosas en el uso para el usuario. Falta estilos, paleta de colores, tipografia etc, en fin un storybook de marca. Crea un plan para aplicar esta skill a lo largo de las necesidades del proyecto"
- **L2** — User: "1. opcion 2, 2. opcion 2" → decision 1 = rebrand (S4), decision 2 = in-app `/styleguide` (S5).
- **L3** — Prior art to reconcile: `odd/tasks/issue-766-slice-*` (earlier "Signal Prime" design-system foundation). This feature supersedes its visual layer; its a11y/contrast tooling (`check-contrast-tokens`, `tailwind-ds-vars` tests, axe e2e) stays.
- **L4** — Baseline: `impeccable detect --json src` (source-only rules) = 3 findings (overused-font ×2, bounce-easing ×1). Rendered rules not yet run.
