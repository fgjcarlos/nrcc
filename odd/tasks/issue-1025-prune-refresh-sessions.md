# Issue #1025 — Refresh session pruning

Source: https://github.com/fgjcarlos/nrcc/issues/1025

## Specs

- S1: Call `PruneSessions` at startup and once a day from the existing scheduler.
- S2: After a prune, the file holds no session that expired or was revoked more than 24 hours earlier.

## Tasks

- [x] T1 (S1–S2) — Wire and verify scheduled pruning; inline with independent verification; `fb50e34`.

## Log

- L1: Resuelve la issue #1025 <!-- l10n: verbatim user request -->
- L2: Abre PR <!-- l10n: verbatim user request -->
- L3: Baseline tests reproduced missing startup pruning and failure to prune by the first revocation time. The corresponding regression tests passed after the fix.
- L4: Independent verification covered schedule lifecycle, cutoff and legacy records, failed writes, and concurrent session updates. It found a race in the scheduler test counter, fixed with an atomic counter.
- L5: `go test -race ./...` passed with host Docker isolated; the initial unisolated run picked up the host's Node-RED instead of the handler test stub. CI-pinned `golangci-lint` v2.14.0 on Go 1.27.4 reported zero issues. `git diff --check` passed.
