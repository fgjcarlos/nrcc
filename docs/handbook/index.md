# NRCC Handbook

> **Status.** The handbook accompanies the Node-RED 5 control-plane
> roadmap (#765). It is the documentation backbone that #771 asked
> for: a single maintained structure explaining how visual controls
> map to Node-RED 5 settings, how configuration sources resolve,
> which changes require restart, and how the four authentication
> boundaries differ. The handbook does not replace existing docs —
> it indexes them.

## Who this is for

- **Operators** running NRCC in production. Start at
  [operator/playbook.md](operator/playbook.md), then jump into the
  topic page that matches your task (upgrade, certificate rotation,
  backup, recovery, dashboard access, …).
- **Contributors** extending the configuration catalog, the parser,
  the renderer, or the API. Start at
  [contributing/style.md](contributing/style.md) and follow the
  cross-links to parser/renderer contracts and testing fixtures.
- **Reviewers** approving changes that touch the apply pipeline or
  any of the four authentication surfaces. The
  [governance/ownership.md](governance/ownership.md) page tells you
  which file owns which slice.

## How the handbook is organised

```
handbook/
├── index.md                 ← you are here
├── glossary.md              ← terms used across the handbook
├── configuration/
│   ├── setting-catalog.md   ← every Node-RED 5 setting NRCC manages
│   └── apply-pipeline.md    ← import → edit → preview → apply → rollback
├── security/
│   └── auth-surfaces.md     ← NRCC access, adminAuth, httpNodeAuth,
│                              httpStaticAuth, Dashboard
├── architecture/
│   └── system.md            ← process model, parser, renderer, API
├── contributing/
│   ├── style.md             ← English source policy + commit + PR hygiene
│   ├── parser-renderer-contract.md
│   └── testing-fixtures.md
├── operator/
│   ├── playbook.md          ← task-oriented guides
│   ├── troubleshooting.md   ← symptom → diagnosis → recovery
│   └── support-matrix.md    ← Node-RED ↔ NRCC ↔ Dashboard versions
└── governance/
    └── ownership.md         ← who owns what, link check, release review
```

## Companion docs (not part of the handbook, but linked from it)

- [`docs/control-plane.md`](../control-plane.md) — umbrella
  traceability matrix (every cluster of #765 → closing PR(s) →
  per-cluster evidence).
- [`docs/configuration/env-contract.md`](../configuration/env-contract.md)
  — bootstrap vs. runtime variables, placeholder rejection,
  precedence rules, per-stack guarantees.
- [`docs/architecture/multi-instance-node-red.md`](../architecture/multi-instance-node-red.md)
  — one Compose stack per Node-RED; per-instance boundaries.
- [`docs/operations/docker-stack.md`](../operations/docker-stack.md)
  — bring up + verify the contract.
- [`docs/operations/setup-recovery.md`](../operations/setup-recovery.md)
  — first-login + admin setup + recovery runbook.
- [`docs/operations/production-install-launch-guide.md`](../operations/production-install-launch-guide.md)
  — production deployment walkthrough.
- [`docs/adr/`](../adr/) — accepted architectural decisions (ADRs
  0001–0004 today; ADRs are the citation source for "why is it
  this way?" questions).
- [`docs/i18n.md`](../i18n.md) — language policy, glossary, and
  catalog coverage. The handbook follows the same English-source
  rule.

## Reading order

1. **Operator first.** Read
   [`operator/support-matrix.md`](operator/support-matrix.md) to know
   which Node-RED major your stack supports, then jump to the
   playbook for your task.
2. **Contributor first.** Read
   [`contributing/style.md`](contributing/style.md), then the parser
   / renderer contract, then the testing-fixtures page, then the
   setting catalog to understand what you can extend.
3. **Reviewer first.** Read
   [`governance/ownership.md`](governance/ownership.md) to know which
   file or test owns the slice you are reviewing.

## Conventions

- **English-only source files**, per #768. Handbook Markdown is
  English. User-facing UI copy is localisable (#767) and lives in
  `frontend/src/locales/<lang>/`.
- **No new design tokens**. The handbook reuses Markdown conventions
  (tables, bullet lists, fenced code blocks, Mermaid for diagrams).
- **Glossary entries** are owned by [`glossary.md`](glossary.md).
  Every acronym introduced elsewhere is added to the glossary in the
  same commit.
- **ADR links**. When the handbook explains a "why", it cites an ADR
  by number, not by reproducing the rationale.
- **Forbidden tokens**. The same freshness gate from #770 forbids
  legacy tokens (`sp-*`, `Beta ·`, `hardening phase`, `no
  production guarantees`, `API keys may change`, `Vitest suites on
  every PR`). They must not appear in any handbook page either.

## Maintenance

See [`governance/ownership.md`](governance/ownership.md). The
freshness gate `.github/workflows/pages-freshness.yml` runs on every
PR that touches `docs/**` and weekly (Monday 06:00 UTC); a broken
link or a stale freshness stamp fails the workflow.
