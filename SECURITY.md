# Security Policy

## Reporting Vulnerabilities

Email security issues to software@fishfarmfeeder.com. Do **not** open public issues for unpatched vulnerabilities.

## Automated Scanning

| Tool | What it checks | Blocks merge | Blocks release |
|------|---------------|-------------|----------------|
| **govulncheck** | Known Go module CVEs | Yes | Yes |
| **pnpm audit** | npm advisory DB (high + critical) | Yes | Yes |
| **Trivy** | Container image CVEs (high + critical) | **No** — advisory: uploads SARIF to the Security tab and never fails the job | **Yes** — any critical or high finding not listed in `.trivyignore` |
| **Dependabot** | Outdated deps across Go, npm, Docker, Actions | No (opens PRs) | No |

Scans run on every PR, every push to `main`, and weekly on Monday at 06:00 UTC.

### Release gate

The release workflow reuses `.github/workflows/security.yml` with
`fail-on-findings: true`. That turns on an extra Trivy step that fails
on any critical or high image finding not listed in `.trivyignore`, so
the package and image jobs never run and nothing is published. A
`workflow_dispatch` dry run applies the same gate, so it previews the
outcome of a tag push. Pull requests, pushes to `main`, and the weekly
scan stay advisory.

To unblock a release, fix the finding (usually a base-image or
dependency bump), or have a maintainer accept it in `.trivyignore`:

```
# <why it is accepted: not reachable, no upstream fix, ...> (#<issue>)
CVE-YYYY-NNNNN exp:YYYY-MM-DD
```

Every entry needs the reason comment and should carry an `exp:` date so
the exception is reviewed again. Ignored findings also disappear from
the Security tab report.

## Operator Responsibilities

- **`NRCC_ENCRYPTION_KEY`** — must NOT be a known placeholder (`change-me-in-production`, `replace-me-XXX`, etc.). The server rejects placeholder values at boot via `service.ValidateSecret`. See `auditoria/REPORT.md` and issue #584 for the fix history.
- **`JWT_SECRET`** — must NOT be a known placeholder; rejected at startup by `main.resolveJWTSecret`. Same helper family.
- **`NRCC_TRUSTED_PROXIES`** — CIDR list of reverse proxies permitted to send `X-Forwarded-For`. Audit log and rate-limit both honor this and use the rightmost `X-Forwarded-For` entry that is not a trusted proxy, so client-written entries on the left are ignored. XFF from an untrusted peer is ignored; when it arrives while this variable is empty, the server logs one warning, because behind a proxy every client then shares the proxy's rate-limit bucket.
- **`NRCC_CORS_ORIGINS`** — explicit origin allowlist for cross-origin browser requests. Empty default = deny-all. Wildcard (`*`) requires `NRCC_CORS_UNSAFE_WILDCARD=true`.
- **`NRCC_METRICS_PUBLIC`** — when `true`, the Prometheus `/metrics` endpoint is served unauthenticated (legacy behaviour). Default is `false` (#671): `/metrics` requires the same JWT as every other authenticated route. Use the opt-out only when the metrics port is bound to a private network reachable by your scraper. The endpoint exposes login-failure counters (a brute-force oracle) and Go runtime fingerprinting data, so do not leave it open on a publicly reachable port.

## Triaging Failures

1. **Critical / High** — fix or pin before merging. If the vulnerability is in a transitive dependency with no available fix, document the exception in the PR description and request maintainer approval. For container findings the approved exception goes to `.trivyignore` (see [Release gate](#release-gate)); without it the next release fails.
2. **Medium / Low** — tracked but not merge-blocking. Dependabot PRs handle these over time.
3. **False positives** — add a `.trivyignore` entry (container) or inline ignore comment with justification.
