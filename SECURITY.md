# Security Policy

## Reporting Vulnerabilities

Email security issues to software@fishfarmfeeder.com. Do **not** open public issues for unpatched vulnerabilities.

## Automated Scanning

| Tool | What it checks | Blocks merge |
|------|---------------|-------------|
| **govulncheck** | Known Go module CVEs | Yes |
| **pnpm audit** | npm advisory DB (high + critical) | Yes |
| **Trivy** | Container image CVEs (high + critical) | Yes |
| **Dependabot** | Outdated deps across Go, npm, Docker, Actions | No (opens PRs) |

Scans run on every PR, every push to `main`, and weekly on Monday at 06:00 UTC.

## Triaging Failures

1. **Critical / High** — fix or pin before merging. If the vulnerability is in a transitive dependency with no available fix, document the exception in the PR description and request maintainer approval.
2. **Medium / Low** — tracked but not merge-blocking. Dependabot PRs handle these over time.
3. **False positives** — add a `.trivyignore` entry (container) or inline ignore comment with justification.
