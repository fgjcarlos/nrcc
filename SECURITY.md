# Security Policy

## Reporting Vulnerabilities

Email security issues to software@fishfarmfeeder.com. Do **not** open public issues for unpatched vulnerabilities.

## Automated Scanning

| Tool | What it checks | Blocks merge |
|------|---------------|-------------|
| **govulncheck** | Known Go module CVEs | Yes |
| **pnpm audit** | npm advisory DB (high + critical) | Yes |
| **Trivy** | Container image CVEs (high + critical) | **No** — `exit-code: "0"` in `.github/workflows/security.yml`. Trivy runs, uploads SARIF to the Security tab, and reports findings, but it does not fail the job. **Critical/High findings must be reviewed manually and addressed in a follow-up PR before the next release.** |
| **Dependabot** | Outdated deps across Go, npm, Docker, Actions | No (opens PRs) |

Scans run on every PR, every push to `main`, and weekly on Monday at 06:00 UTC.

## Operator Responsibilities

- **`NRCC_ENCRYPTION_KEY`** — must NOT be a known placeholder (`change-me-in-production`, `replace-me-XXX`, etc.). The server rejects placeholder values at boot via `service.ValidateSecret`. See `auditoria/REPORT.md` and issue #584 for the fix history.
- **`JWT_SECRET`** — must NOT be a known placeholder; rejected at startup by `main.resolveJWTSecret`. Same helper family.
- **`NRCC_TRUSTED_PROXIES`** — CIDR list of reverse proxies permitted to send `X-Forwarded-For`. Audit log and rate-limit both honor this; untrusted XFF is silently ignored.
- **`NRCC_CORS_ORIGINS`** — explicit origin allowlist for cross-origin browser requests. Empty default = deny-all. Wildcard (`*`) requires `NRCC_CORS_UNSAFE_WILDCARD=true`.

## Triaging Failures

1. **Critical / High** — fix or pin before merging. If the vulnerability is in a transitive dependency with no available fix, document the exception in the PR description and request maintainer approval.
2. **Medium / Low** — tracked but not merge-blocking. Dependabot PRs handle these over time.
3. **False positives** — add a `.trivyignore` entry (container) or inline ignore comment with justification.
