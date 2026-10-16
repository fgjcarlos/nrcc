# NRCC Control Plane — Roadmap Traceability

Single source of truth for the 9 sub-clusters of umbrella issue
[#765](https://github.com/fgjcarlos/nrcc/issues/765) ("make NRCC a trustworthy
Node-RED 5 control plane"). Per-cluster evidence lives in
[`odd/audits/issue-765/`](../odd/audits/issue-765/); this document is the
pointer the audit gap report asked for.

## Cluster status

| # | Theme | Status | Closing PR(s) | Per-cluster evidence |
|---|-------|--------|---------------|----------------------|
| [#756](https://github.com/fgjcarlos/nrcc/issues/756) | NR 5.x compatibility contract | 🟢 GREEN | [#772](https://github.com/fgjcarlos/nrcc/pull/772), [#775](https://github.com/fgjcarlos/nrcc/pull/775) | [`roadmap-traceability.md` §2](../odd/audits/issue-765/roadmap-traceability.md) |
| [#757](https://github.com/fgjcarlos/nrcc/issues/757) | Settings.js source preservation | 🟢 GREEN | [#777](https://github.com/fgjcarlos/nrcc/pull/777)–[#781](https://github.com/fgjcarlos/nrcc/pull/781) | same |
| [#758](https://github.com/fgjcarlos/nrcc/issues/758) | Transactional settings apply | 🟢 GREEN | [#781](https://github.com/fgjcarlos/nrcc/pull/781), [#792](https://github.com/fgjcarlos/nrcc/pull/792), [#793](https://github.com/fgjcarlos/nrcc/pull/793) | same |
| [#759](https://github.com/fgjcarlos/nrcc/issues/759) | Reliable access administration | 🟡 AMBER | [#794](https://github.com/fgjcarlos/nrcc/pull/794) | [`gap-report.md` G3](../odd/audits/issue-765/gap-report.md) |
| [#760](https://github.com/fgjcarlos/nrcc/issues/760) | Authentication surfaces | 🟡 AMBER | [#795](https://github.com/fgjcarlos/nrcc/pull/795), [#797](https://github.com/fgjcarlos/nrcc/pull/797) | [`gap-report.md` G2](../odd/audits/issue-765/gap-report.md) |
| [#761](https://github.com/fgjcarlos/nrcc/issues/761) | Dashboard access surfaces | 🟢 GREEN | [#800](https://github.com/fgjcarlos/nrcc/pull/800) | [`roadmap-traceability.md` §2](../odd/audits/issue-765/roadmap-traceability.md) |
| [#762](https://github.com/fgjcarlos/nrcc/issues/762) | TLS, credentialSecret, requireHttps | 🟢 GREEN | [#776](https://github.com/fgjcarlos/nrcc/pull/776), [#777](https://github.com/fgjcarlos/nrcc/pull/777) | same |
| [#763](https://github.com/fgjcarlos/nrcc/issues/763) | Navigation focused on configuration | 🟢 GREEN | [#827](https://github.com/fgjcarlos/nrcc/pull/827), [#828](https://github.com/fgjcarlos/nrcc/pull/828), [#829](https://github.com/fgjcarlos/nrcc/pull/829) | same |
| [#764](https://github.com/fgjcarlos/nrcc/issues/764) | Advanced settings escape hatches | 🟡 AMBER | [#828](https://github.com/fgjcarlos/nrcc/pull/828) | [`gap-report.md` G1](../odd/audits/issue-765/gap-report.md) |

**Overall:** 6 🟢 GREEN, 3 🟡 AMBER, 0 🔴 RED.

The three AMBER items are policy/evidence gaps, not implementation gaps —
see [`gap-report.md`](../odd/audits/issue-765/gap-report.md) for remediation
per cluster.

## Raw settings validation

`POST /api/config/apply/raw` rejects malformed JavaScript with HTTP 400 and
`error.code: VALIDATION_ERROR` before creating a backup, replacing `settings.js`,
changing its revision, or restarting Node-RED. A rejected apply does not modify
`config.json`. Attempt/failure audit events are still emitted.

The entire source is compiled before isolated authentication inspection:

| Inspection outcome | Apply behavior |
|---|---|
| Syntax error | Reject with a validation error, including errors after a host API reference. |
| Successful inspection without authentication blocks | Accept; authentication is absent. |
| Host-dependent settings (`require`, `process`, or other blocked host globals) | Preserve source-managed settings; host APIs stay blocked and authentication cannot be inspected. |
| Other execution/inspection failure | Reject rather than silently treating it as missing authentication. |

This is not a Node-RED startup guarantee. In particular, compiling host-dependent
source does not verify required modules, environment values, certificate files,
or authentication computed by that source. Host-dependent scripts may fail
later in Node-RED; the inspector never exposes the filesystem, process APIs,
network access, or timers to resolve them. Callback-based settings remain
source-managed. Execution inside the inspector remains time-bounded.

Regression coverage: `internal/handler/config_apply_validation_test.go`,
`internal/service/settings_validation_test.go`, and the real-stack test
`invalid raw settings are rejected without restarting healthy Node-RED` in
`frontend/e2e/stack.spec.ts` (issue [#875](https://github.com/fgjcarlos/nrcc/issues/875)).

## Cross-cutting review documents

- [`roadmap-traceability.md`](../odd/audits/issue-765/roadmap-traceability.md)
  — per-cluster evidence + managed-settings catalog inventory.
- [`navigation-mission.md`](../odd/audits/issue-765/navigation-mission.md)
  — per-route mission classification for every retained live route.
- [`compatibility-mode-pre-flight.md`](../odd/audits/issue-765/compatibility-mode-pre-flight.md)
  — what's in place for `CompatibilityModeE2E` and what remains.
- [`gap-report.md`](../odd/audits/issue-765/gap-report.md)
  — explicit gaps with severity and remediation.
