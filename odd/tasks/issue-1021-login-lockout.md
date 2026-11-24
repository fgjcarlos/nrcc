# Issue #1021: stop unauthenticated callers from locking out logins

Branch `fix/1021-login-lockout` from `origin/main` (0dfe2d2). Security fix (high-risk item 2).

## Specs

- S1: "Key the lockout on username plus IP, keep a slower global limit per username" — "Six failures from one address do not block a login for the same user from another address."
- S2: "take the rightmost untrusted `X-Forwarded-For` entry" — "A spoofed leftmost `X-Forwarded-For` value does not change the bucket."
- S3: "Warn at startup when forwarded headers arrive and no trusted proxy is configured."
- S4: "An unauthenticated caller cannot lock a legitimate user out, and a client cannot choose its own rate-limit bucket."

## Tasks

- T1 [S1-S4] inline: limiter keys (`auth-user-ip:` 6/15 min, global `auth-user:` 30/15 min), rightmost-untrusted `ExtractIP`, one-time XFF warning; Login, MFA verify and password change use the new keys; Go tests, docs, changelog. Commit: 6aa4e04.
- T2 [S1-S4] verify: independent `gentle-ai-verify` (Risk item 2): S1, S3, S4 pass; S2 passed in code, blocked by the stale audit test (B1), fixed in T1. Commit: n/a.

## Log

- L1: "Resuelve la issue #1021"
- L2: Decision: the per-address user bucket keeps the old 6-failure/15-minute lockout; the global per-user bucket allows 30 failures in the same window (five addresses' worth). A successful login resets the IP and user+IP buckets but not the global one, so a distributed attacker gains nothing from the victim logging in. "Warn at startup" is implemented as a one-time log line on the first request carrying `X-Forwarded-For` while `NRCC_TRUSTED_PROXIES` is empty, because headers can only be seen at request time.
- L3: Evidence: RED observed for the new login, MFA and middleware tests on the old behavior; GREEN `go test ./internal/middleware/ ./internal/audit/` ok, focused handler tests ok, `go vet` clean. Full `go test ./...`: only `TestSaveConfig_WritesExplicitRuntimeSettingsAndRestarts` fails, pre-existing on origin/main (see #1020 L4). `TestAudit_Log_IPExtraction_TrustedProxy` pinned the leftmost-XFF bug; it now expects the rightmost untrusted entry.
- L4: Accepted residuals (verifier advisories): an attacker with many addresses (for example an IPv6 /64) can still trip the global 30-failure lock, by design; a locked legacy `auth-user:` entry in ratelimit.json stays locked until its window ends (max 15 min); malformed or `ip:port` XFF entries fall back to the last trusted address, which shares one bucket.
