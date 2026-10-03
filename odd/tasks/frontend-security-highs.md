# Frontend high-severity dependency remediation

## Scope and authorization

- Branch: `fix/nrcc-security-highs`, child of `fix/nrcc-stabilization-updates`.
- Base: `9fe1eba63053234d09cf1a78aa00ac653deb57ab`.
- User authorized remediation, then chose four published fixes first.
- One verified local work-unit commit under the standing authorization;
  no push, PR, merge, publication, waiver, or audit-gate change.
- Preserve existing `odd/tasks/nrcc-stabilization.md` post-commit updates and
  untracked `.codegraph/`; do not stage either in this new unit.
- No production access, dependency installs/build outputs in the worktree,
  broad dependency upgrades, forced majors, or unrelated medium/low remediation.

## H1 — Remove the four high advisories with published fixes

- [ ] Complete targeted fixes, checks, and local work-unit commit.
- Status: **in progress — all targeted checks verified; local commit pending**.
- Allowed source surfaces:
  - `frontend/pnpm-workspace.yaml` (two override floors only)
  - `frontend/pnpm-lock.yaml` (regenerated, reviewed resolution delta)
  - `CHANGELOG.md` (one scoped security note)
- Targets: undici `GHSA-rfgv-xxqx-mfg5`, `GHSA-w293-vg96-wgc3`;
  brace-expansion `GHSA-qhr7-859c-m2p7`, `GHSA-6j4f-fj2g-mc7p`.
- Baseline resolutions: undici 7.29.0 and brace-expansion 5.0.9.
  Floors raised to `^7.29.1` and `^5.0.11`, retaining current majors 7 and 5.
  Official npm manifests confirm both floor versions are published.
- Preserve all other overrides, generator 7.13.0, Redocly 1.34.15 patch registration
  and bytes, package.json, schema, Dockerfile, toolchain pins, and direct deps.
- Explicitly regenerate the lock in an isolated snapshot, then frozen-install
  the result. Frozen install must not be claimed to regenerate an outdated lock.
- RED observed before edits: baseline audit exit 1, five high/zero critical,
  all four target IDs present. Frozen candidate install passed; target IDs removed,
  zero new high/critical. Audit still exits 1 for braces: one high/five moderate,
  zero low/critical (baseline five high/11 moderate/three low).
  This is targeted remediation, not a clean overall audit.
- Checks: actual resolved versions/graph delta, frozen install, targeted package
  compatibility, normal schema generation/hash stability, frontend tests/typecheck,
  locale/lint/build, repository diff check, and independent final Docker/runtime.
  Attribute any failed broad check against matching baseline; never weaken tests.
- Tests use guarded absolute temporary frontend cwd, Corepack pnpm 11.12.0;
  Go 1.26.0/external caches if applicable. Docker uses literal verified snapshot
  context/default final target, own labeled resources only, retained after stopping.
- After writer, follow native ASSESS; RDD is off, no enable/stage bypass.
  Independent verification required if high/unassessable; source hashes must match.
- Review budget: 400 total diff lines including lock/tracker, not additions only.
  Stop for a split decision if unrelated resolution churn exceeds this unit.
- Writer `musuyo0m-14-tunm` completed only the three allowed source paths:
  undici resolved 7.30.0, brace-expansion 5.0.12; lock delta only those existing
  routes (12 additions/12 deletions). Public-consumer smoke/schema generation
  and diff checks passed; patch/schema hashes unchanged. Snapshot:
  `/tmp/nrcc-audit-DRYCSyHs/h1-writer-UM4a4Z`.
- Independent verifier `musvj4y2-16-rrov`: fresh archive plus three source copies,
  frozen install and four-ID audit comparison passed. Both overall audits exit 1
  for braces. `corepack pnpm@11.12.0 exec vitest run`: 84 files, 581 passed,
  two skipped; typecheck/i18n coverage 858/858/lint/build/API generation twice,
  Go 1.26 `go test -count=1 ./...`, canonical default-final Docker all passed.
  Lint: five warnings; build: CSS warnings. Eight source-input hashes unchanged;
  no worktree modules/dist/root lockfile. Evidence:
  `/tmp/nrcc-audit-DRYCSyHs/h1-verify-exact-R1-evidence/`.
- Runtime setup/login/bootstrap/updates and protected reload passed; 5.0.7,
  writable/editable, image-local, no apply action, whole-image guidance.
  First probe missed bootstrap capture (exit 1); corrected capture passed.
  Aggregate refresh array included setup/login/navigation/manual refresh, not
  a per-reload result. `musw3r8o-17-9s9y` observed three isolated reloads:
  exactly one POST refresh 200 each, protected path and auth/me 200 retained,
  no page errors; NRCC and Node-RED HTTP 200. Helper exit 0, source hashes
  unchanged, owned container verified/stopped. `runtime-refresh-scope.json`
  in the evidence directory records the corrected measurement.
- Current source diff: 17 additions/14 deletions; new tracker also counts toward
  the review budget. Existing stabilization tracker and .codegraph stay unstaged.
- Rollback boundary: this unit's two override floors, corresponding lock routes,
  scoped changelog and new tracker only; preserve all S1–S3 behavior and patches.
- Commit: pending; do not check off until verified and committed.

## Deferred blocker — braces

`GHSA-vfj7-8cjw-p6xm` remains HIGH through Tailwind/micromatch/chokidar.
Official npm `braces/3.0.4` returns version-not-found; `braces/latest` is 3.0.3.
Do not introduce an unavailable floor, claim remediation, or ignore this advisory.
A vendor patch alone would not necessarily clear the version-based audit record.
Investigate a verified alternative separately; any exception needs explicit human
maintainer authorization under SECURITY.md. Merge remains blocked in this unit.

## Recovery

Full Engram mirror: `odd/frontend-security-highs/tasks`.
Baseline audit: `/tmp/nrcc-audit-DRYCSyHs/s3-final-N6qW3j-audit-after.log`.
Prior stabilization is functionally closed; preserved snapshots/resources/secret
files stay untouched. No source writer active before H1 starts; one writer only.
