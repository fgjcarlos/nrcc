# Issue #876 — Runtime-safe apply chain

> **Track the reviewed #876 units; hosted review, CI, and integration remain separate gates.**

Refs #876.

## Status At A Glance

| Concern | State |
|---|---|
| Tracker PR | Draft, no merge, no auto-merge |
| Implementation branches | Four reviewed work units (T1–T4), locally approved |
| Integration into `main` | Pending maintainer decision |
| Required CI before integration | Pending — observed check list below |
| Issue closure | Not authorized |

The tracker branch `fix/876-apply-runtime-rollback` itself carries
only this document. Review implementation through the four dependent
child PRs below.

## Local And Hosted Review States

| Layer | State |
|---|---|
| Local source review | Approved for each T1–T4 commit (authored bytes unchanged) |
| Local focused / independent checks | Passed for each unit |
| Hosted PR review | Required before integration; consult the tracker and child PRs |
| Hosted CI | Consult actual results: PR Validation runs on any base; Docker acceptance requires base `main` |
| Integration into `main` | Maintainer decision, not this document |

This document records **what is locally proved** and explicitly
states what is still **pending on GitHub**. It does not waive CI
and does not authorize merge or issue closure.

## Chain At A Glance

| Unit | Branch | Agreed review budget (lines) | Depends on | Purpose |
|---|---|---:|---|---|
| T1 | `fix/876-runtime-transaction` | 515 | `fix/876-apply-runtime-rollback` | Guarded runtime / recovery core |
| T2 | `fix/876-structured-transaction` | 489 | `fix/876-runtime-transaction` | Structured transaction and restoration |
| T3 | `fix/876-runtime-wiring` | 1125 | `fix/876-structured-transaction` | Production authority and audit wiring |
| T4 | `fix/876-runtime-acceptance` | 919 | `fix/876-runtime-wiring` | Real acceptance and operator docs |

These budgets were **explicitly agreed for this chain** under the
root `AGENTS.md` review-budget contract. The standard limit documented
in [`style.md`](style.md) remains unchanged. Each unit retains its
cohesive behavior, tests, or operator guidance; no tests or explanatory
content were removed to reduce counts. These agreed budgets grant no
CI waiver or protected-label change.

Authored line totals: 515 / 489 / 1125 / 919. The chain sums to
3 048 authored lines across four commits.

Commit identities for each unit:

- T1 — `12b1ff4427d60d1f8e6cde0861d69c683ef5fbce`
- T2 — `5027be122bf50ef613154a638010a1a186b2e60a`
- T3 — `b75811cfaec9301567412cf86f8d336dece14382`
- T4 — `7ddc627ce23cba320a26ca23211a095f14104e17`

## Dependency And Review Order

```
main ──▶ fix/876-apply-runtime-rollback   (tracker; this document)
              │
              ▼
          fix/876-runtime-transaction          (T1, 515)
              │
              ▼
          fix/876-structured-transaction       (T2, 489)
              │
              ▼
          fix/876-runtime-wiring               (T3, 1 125)
              │
              ▼
          fix/876-runtime-acceptance           (T4, 919)
```

Suggested review order: foundation (T1) → structured transaction
(T2) → production wiring (T3) → real acceptance and docs (T4).
Each child PR's base is its **immediate parent branch**, so its
three-dot diff contains only that unit's changes.

## Integration Is A Maintainer Decision

- The tracker stays **draft** until maintainers review the chain
  and the required hosted CI on each child PR is recorded.
- No automatic assumption is made that merging one child first will,
  on its own, rebase or update the parent of the next child.
  Whatever forward history each child carries is the history that
  was locally approved; rebases are a maintainer choice performed
  only when the chain is ready to integrate.
- Hosted CI on `.github/workflows/pr.yml` runs for the children
  because its `pull_request` trigger is unfiltered. The Docker
  acceptance gate in `.github/workflows/acceptance.yml` is filtered
  to `main`, so it will **not** trigger on the four feature-base
  children. Record this as not triggered, never as a passed check.
  Required integration checks remain unchanged.

## Start, End, And Out Of Scope

**Start:** the four reviewed commits T1–T4 (unchanged in this work)
plus this tracker document on `fix/876-apply-runtime-rollback`.

**End:** a maintainer — not this document — decides when to
integrate the chain into `main`, in what order, and after which
hosted CI run on each child PR is recorded. The chain is not
integration-ready until then.

**Out of scope for this PR:**

- Any merge, force-push, or rebase of the four reviewed T1–T4
  commits, or any change to their authored bytes.
- Changes to unrelated source, configuration, or local untracked artifacts.
- Changes to repository workflow configuration, protected-branch
  rules, or protected size labels.
- Any CI dispatch, waiver, or override.
- Closing issue #876. The PR body will reference it as `Refs #876`.

## Related

- [`style.md`](style.md) — work-unit and Conventional Commit
  conventions this chain follows.
- [`testing-fixtures.md`](testing-fixtures.md) — fixture conventions
  used by T4's real-acceptance work.
- [`../configuration/apply-pipeline.md`](../configuration/apply-pipeline.md) —
  the apply pipeline the runtime guards are part of.
- [`../../control-plane.md`](../../control-plane.md) — the umbrella
  roadmap and traceability document.
