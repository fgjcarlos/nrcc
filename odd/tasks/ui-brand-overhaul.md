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

- [x] T1 — S1,S3 — Phase 0: `/impeccable init` → `PRODUCT.md` (interview) — inline — commit: 59b9254
- [x] T2 — S1 — Phase 1: rendered diagnosis: `critique` per surface + `audit` + `detect <dev-server URL>` → prioritized P0–P3 backlog — inline + explore subagents — commit: this doc (snapshot in `frontend/.impeccable/critique/`, self-ignored)
- [ ] T3 — S2,S4 — Phase 2: new visual world (`new-work`), `DESIGN.md` (`document`), tokens rewritten in `index.css`, self-hosted fonts, logo/wordmark; retire stale `COLOR_REFERENCE.md` — worker — commits: 9bbc221 (surface brief + direction contract), 6f186eb (tokens, fonts, effects). Open: logo/wordmark, retire `COLOR_REFERENCE.md` + `SIGNAL_PRIME_TOKENS.md`, DESIGN.md after T9 builds the proof surface
- [ ] T4 — S2,S4 — Phase 3a: `extract` shared primitives (Button, Input/Select/Textarea, Card, Dialog, Table, Tabs, Badge/StatusChip, feedback states) — route TBD — commit: _pending_
- [ ] T5 — S5 — Phase 3b: dev-only `/styleguide` route (foundations + components, both themes, en/es) — route TBD — commit: _pending_
- [ ] T6 — S1 — Phase 4: app chrome (sidebar, header, command palette) + i18n leaks — commit: _pending_
- [ ] T7 — S1 — Phase 4: Login / Setup (`onboard`, `clarify`, `harden`) — commit: _pending_
- [ ] T8 — S1 — Phase 4: Overview (`layout`, `colorize`, `typeset`) — commit: _pending_
- [ ] T9 — S1,S4 — Phase 4: Configuration + change circuit, the proof surface of the new world (pulled forward right after T3 foundation; `harden`, `clarify`, `layout`) — commit: aeb02e3 (finish review pending)
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
- **L5** — Init interview: primary user = "Operador de su propia instancia"; brand commitment = "Solo el nombre NRCC" (mark may be redesigned); constraints = "WCAG 2.2 AA + offline". No `buildPath` recorded (app screens are code-first). Live mode not configured yet.
- **L6** — T2 baseline critique (whole app, isolated stack, dark+light desktop, dark 390px): **22/40 Acceptable**, P0=1, P1=5. Rendered detector totals: low-contrast 87 (73 in light), undersized-ui-text 60, nested-cards 32, gpt-thin-border-wide-shadow 24, overused-font 22, kicker-above-heading 18, radial-spotlight-glow 11, straggler 6, skipped-heading 2, bounce-easing 1. Backlog: P0 Environment header actions off-screen at 390px; P1 raw i18n keys + Spanish in EN UI; P1 AA contrast/1.4.11 failures (light input borders, 10.88px text); P1 restore dialog (UUID retype, ES copy); P1 gated change sequence invisible; P1 generic identity + light theme copied from dark; P2 header status contradicts page/hidden on mobile; P2 Security overload, Overview alarm without fix; P3 nav naming. Separate audit pass folded into the rendered detector run.
- **L7** — Functional bug found (outside rebrand scope): `frontend/src/features/backups/components/BackupsView.tsx:345` calls `retentionMutation.mutate(manual)`, so auto and pre-restore retention edits are never persisted.
- **L8** — User decisions after critique: order = "Rebrand primero (Recomendada)" (urgent fixes land inside each surface slice); brand axis = "Sí, el \"riel\" es la marca" (shared transaction rail for Config/Security/Env/Restore defines the visual language); retention bug = "Arreglarlo ya, aparte" (own branch from main, outside this feature).
- **L9** — Retention bug fixed apart on `fix/backup-retention-save` (commit e7fad89), PR #974. Root cause wider than reported: `PATCH /storage/retention` writes one value into all three retention kinds (`internal/handler/backup.go:400-402`); fix saves via `POST /backups/config`. Backend endpoint left as a follow-up.
- **L10** — Direction round (decision page unavailable: no browser in env; structured tool used): user chose the assigned direction "Esquema eléctrico" (IEC wiring-diagram world), seed:a927e97550a2032a, code-led (Operate). Surface questions: first surface = "Configuration + riel (Recomendada)"; scene = "Mezcla sin dominante" (light and dark first-class); wrong if = "Parece marketing / juguete", "Pierde densidad"; untouchable = "se peude cambiar todo".
- **L11** — Foundation slice 6f186eb verified: contrast strict 206 pairs pass both themes, unit 611 passed / 2 skipped, test:css, test:appearance, build pass; independent verify of Risk item 5 (fonts @5.3.0, OFL-1.1, no install scripts, no runtime deps, no external font hosts in dist, frozen-lockfile installs) all pass. Rendered check: glows gone, blue action, red only for critical. Known gap: light inputs use `glass-panel` and barely show a border (T4).
- **L12** — Slice boundary 1 (delivery: chained PRs): commits 59b9254..6f186eb (product context, baseline critique, direction contract, foundation) form PR slice 1 (~1.5k authored lines, mostly index.css rewrite: +540/−822). Next work starts slice 2.
- **L13** — T9 proof surface committed aeb02e3 (change circuit; truthful state mapping in `frontend/src/features/configuration/lib/changeCircuit.ts`; backup designator BK-… not shown because the API returns no backup id: backend follow-up). Evidence: unit 620 passed / 2 skipped, typecheck, i18n, contrast strict, appearance, build pass; e2e stack 13 passed / 1 skipped (before fix batch); rendered 1440 both themes + 390 dark: no overflow, visible control borders. Slice boundary 2: aeb02e3 (+e2e selector update) = PR slice 2 (~1k lines).
