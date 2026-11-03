# Release packaging and publication

The `Release` workflow packages the supported Linux artifacts, builds and
verifies the container image and — only for a version tag — publishes
both. It is the single place where "what may be published" is decided,
and every decision is unit tested in `scripts/release/test_workflow.py`.

## Quick path

**Dry run (nothing is published):**

1. Open **Actions → Release → Run workflow**.
2. Enter the version, e.g. `v1.2.3` or `v1.2.3-rc.1`.
3. Read the log: the artifacts are packaged and verified, the image is
   built and smoke tested. Download the verified binaries and checksums
   from the run's `nrcc-<version>` artifact. No registry login, image push
   or GitHub release happens.

**Publish:**

1. `git tag v1.2.3 && git push origin v1.2.3`.
2. The workflow validates the tag, re-runs the required checks at the
   tagged commit, packages the artifacts, pushes the image and creates
   the GitHub release.

There is no publish switch: a manual run is always a dry run.

## Entry points

| Trigger | Version source | Publishes? |
|---------|----------------|------------|
| `push` of a `v*` tag | the tag name | yes |
| `workflow_dispatch` | the `version` input | no — dry run only |

The `version` input accepts `v1.2.3` and `1.2.3`; both are normalised to
the same release version (`v1.2.3`).

## Version policy

Versions must be strict semver. A malformed tag fails the run before
anything is built.

| Accepted | Rejected |
|----------|----------|
| `v1.2.3`, `v0.0.1`, `v10.20.30` | `v1.2`, `v1.2.3.4` |
| `v1.2.3-rc.1`, `v1.2.3-0.3.7` | `v1.2.3-`, `v1.2.3-rc..1`, `v1.2.3-01` |
| `v1.2.3+build.5`, `v1.2.3-rc.1+build.5` | `v01.2.3`, `V1.2.3`, `release-1.2.3` |

Build metadata never reaches an image tag (`+` is not valid there); it
only qualifies the release version itself.

## Channels

| Release | Image tags | GitHub release |
|---------|------------|----------------|
| stable (`v1.2.3`) | `1.2.3`, `1.2`, `latest` | `prerelease: false`, `make_latest: true` |
| prerelease (`v1.2.3-rc.1`) | `1.2.3-rc.1` only | `prerelease: true`, `make_latest: false` |

A prerelease never takes over `latest` and never moves the floating
`major.minor` channel. The rules live in `scripts/release/metadata.py`,
which the release workflow calls instead of hardcoding tag patterns.

## Required checks

Publication and packaging wait for two reusable workflows that are
re-run **at the exact triggering SHA** (`github.sha`):

| Reusable workflow | Jobs |
|-------------------|------|
| `.github/workflows/pr.yml` | Backend, Frontend, OpenAPI, Lint, Release policy |
| `.github/workflows/security.yml` | govulncheck, pnpm audit, Trivy scan |

This replaces a lookup of commit statuses: no check name is guessed and
no status API is called. Because the whole Security workflow is reused,
the caller (`release.yml`) grants `security-events: write` — the Trivy
job uploads its SARIF report with that permission.

## Secrets and permissions

| Secret | Needed for | Without it |
|--------|-------------|------------|
| `GITHUB_TOKEN` (automatic) | GHCR login and image push | — |
| `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` | optional DockerHub target and login | the DockerHub target is neither published nor logged into |

The optional DockerHub credentials are assigned to the job environment
and a single availability flag drives both the image target list and
the registry login, so the workflow can never push to a registry it did
not authenticate against.

## Artifact and image contract

| Producer | Interface |
|----------|-----------|
| `scripts/release/package.sh` | `package.sh VERSION OUTPUT_DIR` — builds `nrcc-linux-{amd64,arm64,armv7}` plus checksum manifest and sidecars |
| `scripts/release/verify.sh` | `verify.sh VERSION OUTPUT_DIR` — checksums, cross-architecture Go build metadata, isolated install and start smoke |
| `Dockerfile` | `ARG VERSION` (default `dev`) — the OCI version label and the embedded binary version agree |

The image is built for `linux/amd64`, `linux/arm64` and `linux/arm/v7`.
Before anything is pushed, a single-platform build is loaded into the
local daemon and checked twice: the `org.opencontainers.image.version`
label must equal the release version, and the started container must
remain running, answer `/healthz` and report the same version. The smoke
container and its anonymous volume are removed afterward. Binary and OCI
version metadata retain any `+build` suffix; container tags omit it.

## Troubleshooting

| Symptom | Cause |
|---------|-------|
| `error: 'v1.2' is not a strict semantic version` | the tag or the manual version is not strict semver |
| `Run release policy tests` fails | `scripts/release/metadata.py` and its test suite disagree — run `python3 scripts/release/test_workflow.py` locally |
| The image job is skipped | the check workflows failed; read the `pr-checks` / `security-checks` jobs |
| No DockerHub image was published | `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` are not configured — this is expected, not an error |

## Next step

For the installation path of a published release, see
`docs/production-install-launch-guide.md`.
