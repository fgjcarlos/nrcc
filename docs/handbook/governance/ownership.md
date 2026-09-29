# Ownership

> **Three owners, one audit log.** NRCC's ownership model is
> intentionally thin: a control-plane owner, an operator-experience
> owner, and a release owner. Anything that crosses owners is
> decided in the open and recorded in the audit log.

## The three roles

### Control-plane owner

Owns the Go binary, the audit log, the apply pipeline, the
catalog, and the parser/renderer pair.

**Files owned:**

- `cmd/nrcc/`
- `internal/`
- `go.mod`, `go.sum`
- `scripts/check-pages-links.mjs` (the link checker that ships
  with the docs site)
- `.github/workflows/` (workflow YAML only; the actions under
  `.github/actions/` are owned by the operator-experience owner)

**Decisions:**

- Adding a new catalog entry: open an issue with the use case.
  The owner reviews the shape, validation rule, and the per-entry
  narrative before merging.
- Changing the apply pipeline: requires a typed `*ApplyError`
  for the new failure mode, a fixture in
  `internal/service/testdata/`, and an audit-event change.
- Changing the audit hook: requires a migration plan; audit logs
  are append-only and never rewritten.

**Rotation policy.** The control-plane owner rotates every six
months. The handover includes a full walkthrough of the audit
log and a review of the open issues tagged
`area: devops` or `area: backend`.

### Operator-experience owner

Owns the structured UI, the i18n catalogs, the docs site, and
the contributor style guide.

**Files owned:**

- `frontend/src/`
- `frontend/tailwind.config.js`
- `frontend/vitest.config.ts`
- `frontend/e2e/`
- `docs/`
- `docs/sections/`
- `docs/handbook/`
- `docs/assets/`
- `.github/actions/`

**Decisions:**

- Adding a new feature area in the UI: open an issue with the
  acceptance criteria and the loading / empty / error state
  plan.
- Changing the design system: requires an ADR (`docs/adr/`).
  Existing `ds-*` tokens cannot be renamed without a deprecation
  cycle.
- Translating a new language: requires a translator review pass
  on every catalog entry; partial translations are forbidden.

**Rotation policy.** Same as the control-plane owner.

### Release owner

Owns the release cadence, the version tags, the changelog, and
the migration scripts.

**Files owned:**

- `CHANGELOG.md`
- `docs/control-plane.md` (the roadmap umbrella)
- `docs/openapi.yaml`
- The container image build (Dockerfile, `.dockerignore`)

**Decisions:**

- Bumping the catalog version: requires a sync update from the
  control-plane owner and a release note.
- Tagging a release: requires a green CI on `main`, an updated
  changelog, and a signed tag.
- Hotfixing a production incident: the release owner coordinates
  with the control-plane and operator-experience owners; the
  hotfix branch is merged to `main` and backported to the
  release branch in a single PR.

**Rotation policy.** The release owner does not rotate; the role
is held by the project maintainer.

## Decision flow

A change that crosses owners follows this flow:

1. **Open an issue** in `fgjcarlos/nrcc/issues` with the use case.
   Tag the issue with the affected owner's label.
2. **Discussion** — the affected owner proposes a design in the
   issue. Cross-owner changes require both owners to agree in
   the issue thread.
3. **ADR** — if the change is a significant architecture shift,
   write an ADR under `docs/adr/`. The ADR is a numbered Markdown
   file (`00NN-title.md`) that links the issue and lists the
   alternatives considered.
4. **Implementation** — the work-unit branches off `main`. Each
   work-unit commit is ≤400 authored lines.
5. **PR** — the PR description cites the issue, lists the
   work-units, and shows the test command(s) the author ran.
6. **Merge** — the affected owner approves. The release owner
   merges to `main` after CI is green.

## Audit log

The audit log is the canonical record of every operator-visible
state transition. The audit log:

- Is append-only (`/data/audit.jsonl`).
- Is owned by the control-plane owner.
- Is exported as JSON lines.
- Is the source of truth for incident response (every postmortem
  cites audit event IDs).
- Is rotated by the periodic snapshot loop. The rotation policy
  is documented in
  [`../operator/troubleshooting.md § scenario 5`](../operator/troubleshooting.md#scenario-5--audit-log-fills-the-disk).

## Handbook change policy

This handbook is owned by the operator-experience owner. Changes
to the handbook:

- Land in a work-unit branch (one commit per page).
- Pass the freshness gate (no forbidden tokens, every
  freshness stamp is in the allowlist).
- Pass the link integrity check.
- Are reviewed by a second set of eyes before merge.

A page is deprecated by:

1. Adding `> **Deprecated.** This page is kept for historical
   reference. Use `<new-page>` instead.` at the top.
2. Adding a redirect note in `docs/handbook/index.md` pointing
   to the replacement.
3. Removing the page from any cross-references in the rest of
   the handbook.

Deprecated pages are kept in the repo indefinitely so old
links do not break.

## Related

- [`../contributing/style.md`](../contributing/style.md) —
  contributor conventions.
- [`../operator/support-matrix.md`](../operator/support-matrix.md)
  — what the project supports.
- [`docs/control-plane.md`](../../control-plane.md) — roadmap
  umbrella.
- [`docs/adr/`](../../adr/) — architectural decision records.
