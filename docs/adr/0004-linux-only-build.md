# ADR 0004 — Linux-only build via `//go:build linux`

## Status

Accepted. Resolves #667.

## Context

`internal/service/flowlock.go` and the `isDevNull` helper in
`internal/service/host.go` use POSIX-only syscall symbols (`syscall.Flock`,
`syscall.LOCK_EX`, `syscall.LOCK_NB`, `syscall.O_NOFOLLOW`, `syscall.Fstat`,
`syscall.Stat_t`) without build constraints, so `go build ./...` fails on
Windows. The rest of the codebase is already half-platform-aware: it carries
`_linux.go` / `_unix.go` / `_windows.go` / `_other.go` pairs whose Linux
variants are the only ones used in CI, while the `_other.go` / `_windows.go`
fallbacks exist solely so the package would compile if anyone ever asked it
to. CONTRIBUTING.md does not state a supported-platform list, so the failure
reads as a broken repo rather than an intentional constraint.

The issue proposes two paths: (A) finish cross-platform support, including a
real `LockFileEx` implementation and an `os.O_EXCL` fallback; or (B) declare
the project Linux-only and delete the orphan fallbacks.

## Decision

Adopt Option B. The project is Linux-only at compile time.

- `internal/service/flowlock.go` → `flowlock_linux.go` with `//go:build linux`.
- `isDevNull` moves from `host.go` into a new `host_linux.go` with
  `//go:build linux`; `host.go` no longer imports `syscall`.
- Orphan fallbacks deleted (each one had a Linux counterpart that becomes
  the sole definition):
  - `internal/handler/disk_windows.go`
  - `internal/handler/system_other.go`
  - `internal/service/metrics_sampler_other.go`
- The `_unix.go` variant of `getDiskInfo` is renamed to `_linux.go` and its
  build tag tightened from `linux || darwin || freebsd || openbsd || netbsd`
  to `linux` — matches the convention used by `system_linux.go` and
  `metrics_sampler_linux.go`.
- The `isDevNull` tests move from `host_test.go` to a new
  `host_linux_test.go` (build tag `linux`) so the test file no longer
  references a Linux-only symbol.

## Consequences

- Windows and macOS no longer build natively. The README already calls
  Docker the canonical deployment model (ADR 0003), and Docker on Windows /
  WSL / macOS runs Linux containers — that path is unchanged.
- CI runs on Linux runners; the existing checks (`go build ./...`,
  `golangci-lint`, frontend gates) are unaffected.
- `GOOS=windows go build ./internal/service/...` now fails with
  `undefined: sampleHost` (no fallback was kept) rather than
  `undefined: syscall.Flock` — the constraint took effect, and the
  remaining missing symbol marks the package as Linux-only.
- CONTRIBUTING.md gains a "Supported platforms" section so the next
  Windows contributor finds the answer in the first file they read.

## Alternatives considered

- **Option A — finish cross-platform support.** Implement `LockFileEx`,
  build an `os.O_EXCL` lockfile fallback, and keep the `_other.go`
  stubs for `getSystemStats`, `getCPUUsage`, `sampleHost`, and
  `getDiskInfo`. Rejected: every NRCC deployment ships through Docker
  (ADR 0003), the underlying Node-RED process is Linux anyway, and the
  cost of maintaining a Windows path we never exercise is not justified.

## Related

- `CONTRIBUTING.md` — "Supported platforms" section lists Linux as the
  only build target.
- ADR 0003 — Docker-first deployment model (the basis for picking Option B).