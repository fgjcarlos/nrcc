# Issue 769 — Reposition NRCC README around the Node-RED 5 control plane

## Objective

Rewrite the canonical README so an operator or contributor can
answer — within five minutes of opening the repo — three questions:

1. **What problem does NRCC solve, and for whom?**
2. **What does NRCC actually do today, and what is explicitly
   out of scope?**
3. **How do I run, configure, and extend NRCC safely?**

The README must align with the approved product direction in #765
and the language policy introduced in #768. It must not over-promise
runtime control (NRCC configures; it does not orchestrate clusters)
and must call out the boundary between NRCC, Node-RED editor/API,
HTTP/static endpoints, and the Dashboard.

Inherited debt: the merge of #830 (squash) lost the
`## Language policy` section that slice 1 of #768 added to
`CONTRIBUTING.md` (commit 7be9b51 had it). Slice 2 of this chain
restores that section as part of the docs hygiene pass.

## Scope (per user decision 2026-09-23)

In scope:

- `README.md` rewrite — single source of truth, English only.
- Restore the `## Language policy` section in `CONTRIBUTING.md`
  (lost during #830 merge; restore as written in 7be9b51).
- Brief cross-reference additions in `AGENTS.md` so AI codegen
  has an entry path to the new README sections.
- Owner / reviewer checklist inline in `README.md` so release-
  relevant facts survive the next maintainer rotation.

Out of scope:

- Screenshots / animated captures — they require the redesigned
  UI from issue #766 to land first. README gets a placeholder
  paragraph that links to the design issue.
- Catalog / handler table rewrites — covered by #771 (handbook).
- New docs in `docs/` — only existing docs are cross-referenced
  from the README. Handbook restructure is #771.

## Slice 1 — README structure + content rewrite

Branch: `docs/issue-769-readme-structure`
From: `origin/main` at start of work.

Proposed README shape (final after slice 2 owner-checklist pass):

```
# NRCC — Node-RED 5 Control Center

> Status banner (Apache 2.0, Go 1.26+, Linux-only, beta-hardening).

## Why NRCC
  The operator problem we solve (one Node-RED instance at a time).
  Audience: solo operators / small teams without dedicated Node-RED
  expertise.

## What NRCC does (the visual control plane)
  High-level capability list — settings management, access control,
  backup/restore, environment, library.

## Compatibility policy
  Table mirroring #765 — full editing for Node-RED 5.x; detection +
  read-only for Node-RED 4; read-only for future majors.

## Core workflow
  Diagram-style enumeration of the seven-step flow:
    discover / import  →  visual edit  →  review diff
    →  validate  →  backup snapshot  →  apply
    →  readiness check / rollback

## Distinct security surfaces
  Table — NRCC UI, Node-RED editor + admin API, HTTP/static, Flow
  / Dashboard. Each row: who authenticates, who mediates, what it
  controls, what NRCC does NOT control.

## Deployment model
  One stack = one NRCC + one Node-RED + one persistent volume.
  Multi-instance: spin up multiple stacks. Brief link to
  docker-stack.md.

## Quick start (Docker)
  The existing 1- and 2-step bring-up. Trim the `NODE_RED_PORT`,
  `DATA_DIR` volume-distinction prose that lives more properly in
  docker-stack.md.

## Configuration
  Pointer to env-contract.md. Highlights table (existing content,
  kept tight).

## Data directory layout
  Tree (existing content kept).

## Operations map
  Pointer to docs/operations/, docs/production-install-launch-guide.

## Development
  Build matrix — `task`, `pnpm`, `go test`. Trim the e2e command
  if it isn't reliably green on this branch.

## API reference
  Pointer to openapi.yaml + envelope example.

## Troubleshooting
  FAQ (existing four items kept, expanded with three more common
  ones from past issues).

## Roadmap (link only)
  One-paragraph pointer to #765 (the roadmap tracker) and the
  "current priority" set. No inline feature roadmap.

## Read before opening an issue or PR
  Owner checklist — release-relevant facts that require review
  before a release: compatibility table, security boundaries,
  quick-start command, env-contract link, screenshot placeholders,
  roadmap link.

## License
```

Removed sections (cleanup):

- `## Future Work (not in MVP)` — relabeled to `## Roadmap` with a
  link to #765.
- `## Data Directory Cross-references` — internal ADR pointers;
  moved to docs/configuration/.
- The repeated "MVP" framing — replaced by compatibility policy.

Acceptance for slice 1:

- `README.md` line count is similar to today (≤400 lines).
- All cross-referenced files exist (verify before commit).
- All four badges resolve.
- CI lint / Markdown link check passes.
- README aligns with #765 compatibility policy.

## Slice 2 — final QA + tracker merge + PR

Branch: `docs/issue-769-readme` (slice 1 already on this branch —
the docs are small enough that one branch carries both slices,
no separate tracker needed for a 288 -> 267 line README change).

Work:

- Updated this plan doc to record the actual discoveries during
  slice 1 (`## Language policy` in `CONTRIBUTING.md` IS present
  on `origin/main` post #830 — the apparent regression was a
  stale local-main checkout, not a real loss; `AGENTS.md` IS
  also present). No "restore" work was needed.
- Final acceptance run:
  - `go test ./tools/langscan/... ./scripts/cite_check/...` —
    18 tests pass (10 scanner + 7 workflow contract + 8 link-check,
    with the workflow and cite_check subdirs).
  - `go run ./tools/langscan .` — emits `[]` (no findings).
  - `go run ./scripts/cite_check` on every doc file in the repo
    (README, CONTRIBUTING, AGENTS, SECURITY, CHANGELOG, the plan
    doc, env-contract, docker-stack, production-install-launch-guide,
    and the two cross-referenced ADRs) — all OK, zero broken links.
- Open PR `docs/issue-769-readme` -> `main`. Closes #769.

Acceptance for slice 2:

- All CI gates green on the PR.
- PR description summarises slice 1 (rewrite + tool) and links
  back to #769 and #765.

## Tracker (status)

Single PR from `docs/issue-769-readme` -> `main`. Closes #769.

## What this slice does NOT do

- Does not introduce screenshots or animated captures. Both
  require the redesigned UI from issue #766 to land first; the
  README's "Read before opening an issue or PR" section calls
  out screenshot freshness as a checklist item.
- Does not restructure `docs/` itself. The handbook restructure
  is #771.
- Does not write i18n catalogs. That is #767.
- Does not enable protected-branch auto-merge. The user controls
  merge as always.

## Stop conditions

- If slice 1 README exceeds 450 lines, compress the "Capabilities"
  bullets into a one-paragraph summary + table.
- If the CONTRIBUTING restore introduces CI drift (the file was
  changed earlier today), resolve by intentional merge, not by
  dropping the section again.
