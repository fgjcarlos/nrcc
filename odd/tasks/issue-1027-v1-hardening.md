# Issue 1027: v1 hardening backlog

Source: https://github.com/fgjcarlos/nrcc/issues/1027 (approved). Verify current behavior before changing it; keep larger findings separate.

## Specs

S1. An abandoned MFA enrolment blocks re-enrolment: the pending row counts as enrolled, so the user gets 409 while status says disabled (`internal/service/mfa.go:133,445-448`).
S2. *Suspected:* MFA verify sets the refresh cookie at `Path=/api/auth` while refresh and clear use `Path=/`, which can sign MFA users out after the first refresh (`internal/handler/mfa.go:289`, `internal/handler/auth.go:822,836`).
S3. *Suspected:* structured saves inject `httpStatic: "uploads"`, which may expose uploaded files without authentication on the Node-RED port (`internal/service/config.go:1110,1189-1195`).
S4. The JWT middleware never re-checks the user, so a deleted or demoted account keeps access for up to 15 minutes (`internal/middleware/auth.go:28-37`).
S5. A TOTP code can be replayed inside its time window.
S6. Login for an unknown user skips bcrypt, which allows timing-based user enumeration (`internal/handler/auth.go:285-296`).
S7. The content security policy allows `'unsafe-inline'` scripts.
S8. The npm name validator accepts `name@<anything>`, including git and `file:` specs, and installs run lifecycle scripts (`internal/service/packagemanager.go:98,112-116,143`).
S9. The restic snapshot id is not checked for a leading dash (`internal/service/backup_restic.go:250`).
S10. The settings sandbox has a 2 s interrupt and no memory cap (`internal/service/settings_sandbox.go:86-93`).
S11. The env encryption key is derived with one SHA-256 pass and no minimum length (`internal/service/crypto.go:79-82`).
S12. A client disconnect cancels the request context and rolls back a valid apply (`internal/handler/config_apply.go:214`).
S13. Raw apply rejects settings without a literal numeric `uiPort`, including the file NRCC writes at bootstrap (`internal/server/apply_runtime.go:25-28,210-211`).
S14. Shutdown can take 5 s to stop Node-RED plus 5 s to drain HTTP, which equals Docker's default 10 s. Set `stop_grace_period` in the compose file.
S15. Node-RED is stopped before HTTP drains, so an in-flight apply rolls back during shutdown (`main.go:152`).
S16. `Server.Shutdown` and `UpdateService.Shutdown` are never called outside tests.
S17. The 100 MiB upload cap is unreachable: `ReadTimeout` is 15 s for the whole body and multipart spills to a 64 MB tmpfs (`main.go:122`, `internal/handler/files.go:36-39`).
S18. Settings snapshots (`*.js.bak`) and `.failed-*` files are never pruned.
S19. Listing backups re-hashes every archive on each call, so any authenticated user can generate CPU load (`internal/service/backup.go:194,1103`).
S20. Each boot stores a duplicate flow version (`internal/service/flow_version.go:100-105`).

## Tasks

- T1 (S1–S2): Restore MFA enrollment and refresh consistency. Route: inline + independent verification. Commit: `5bf1eb9` (verified).
- T2 (S20): Deduplicate unchanged flow versions across boots. Route: inline. Status: verified; commit recorded in the next unit.
- T3 (S9, S13–S16): Verify restic guard and address bounded runtime/lifecycle findings. Route: pending. Commit: pending.
- T4 (S3–S8, S10–S11): Review remaining security findings; separate larger changes. Route: pending. Commit: pending.
- T5 (S12, S17–S19): Review apply cancellation, uploads, retention and backup listing; separate larger changes. Route: pending. Commit: pending.

- T6 (S1–S20): Run the full Go suite in an isolated runtime environment. Route: inline. Commit: n/a.

## Log

- L1: “Resuelve la issue #1027”
- L2: “Toda la issue, por tandas revisables”
- L3: Based on origin/main at `699c475`; existing untracked audit and Pi files are out of scope.
- L4: T1: the pending-enrollment, MFA refresh rotation and legacy-session migration tests failed before their fixes and passed after. Independent verification passed S1 and S2 after correction. The broad auth/setup checks and OpenAPI regeneration passed. Risk: item 2 (MFA enrollment and session cookies).
- L5: `go test ./...` failed on `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts` with host Node-RED 5.0.10 instead of the test stub 5.0.6. Targeted baseline/candidate comparison reproduced the same failure and exit 1 on both. T6 tracks the full-suite environment separately.
- L6: T2 RED: two real polling startups under a fake clock captured two identical versions. GREEN: the same test preserves the latest version on unchanged restart and captures again after an edit; all FlowVersion tests and the service suite passed. Risk: none.
- L7: T6 setup: overriding Docker with an unused Unix socket let the failing handler test pass; the host Docker runtime was overriding its native Node-RED stub.
