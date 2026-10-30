# Issue #884 — Align packaging with supported platforms and publication channels

Baseline: `origin/main` a1daeb6. Branch: `fix/issue-884-release-packaging`.
Scope: release packaging (workflow, image metadata, packaging tooling) and the
release contract documentation. No publication is performed by this work: the
release pipeline is aligned and made verifiable, the actual publish stays a
maintainer action.

## Specs

- S1: "Remove unsupported native targets, or implement/test them before claiming support; preserve supported Linux architectures."
- S2: "Make optional registry login and image targets conditional together."
- S3: "Keep prereleases out of latest and mark candidate releases appropriately."
- S4: "Gate publication on successful required checks for the exact commit; provide a dry-run packaging path."
- S5: "Verify checksums, image/binary version metadata and installation of produced artifacts."
- S6: "Docker Hub login checks env values never assigned from secrets while Docker Hub tags are unconditional."
- S7: "Any v* tag starts publication without test dependencies and explicitly emits latest."
- S8: "Registry/channel observations are static, not a performed release."
- S9: "macOS build fails with undefined ProcessManager. The workflow still attempts Windows."
- S10: "The binaries workflow cross-compiles macOS/Windows after ADR 0004 made the project Linux-only. Registry selection and candidate tagging also need to match the supported release contract."
- S11: "A validated candidate can package supported targets and publish only intended registries/channels."

## Tasks

- T1 [S1, S6, S9, S10] inline, in progress: `tools/releasepack` — supported-target manifest, release plan (channels/tags/images/labels), artifact builder, checksum + version-metadata verifier, required-check gate evaluator. Unit tests for every rule. Commit: 287018d (plan model + CLI); build, verify and gate slices follow.
- T2 [S5] inline, pending: version surface and image metadata — `nrcc --version` output, `Dockerfile` `VERSION`/`VCS_REF` build args, OCI labels, `main.Version` stamping.
- T3 [S1, S2, S3, S4, S5, S6, S7, S8, S11] inline, pending: `.github/workflows/release.yml` rework — Linux-only artifacts, conditional Docker Hub login + image targets, stable/prerelease channels, required-check gate on the tagged commit, dry-run packaging path, artifact verification steps.
- T4 [S3, S5, S8, S11] inline, pending: release contract documentation and local dry-run entry point (`scripts/release/dry-run.sh`, `docs/operations/release.md`, launch-guide checklist, CHANGELOG).
- T5 [S1, S2, S3, S4, S5] delegated, pending: independent verifier (`gentle-ai-verify`) — risk items 1, 3 and 5 (irreversible publication, consumed contracts, delivery/CI).

## Log

- L1 User: "Resuelve la issue #884"
- L2 Issue #884 (approved, P1, area: devops): "The binaries workflow cross-compiles macOS/Windows after ADR 0004 made the project Linux-only. Registry selection and candidate tagging also need to match the supported release contract." Acceptance criteria quoted verbatim in S1-S5.
- L3 Baseline read: `release.yml` builds `nrcc-darwin-amd64`, `nrcc-darwin-arm64`, `nrcc-windows-amd64.exe` (ADR 0004 declares Linux-only), always emits `type=raw,value=latest`, always lists `docker.io/fgjcarlos/nrcc`, and gates the Docker Hub login on `env.DOCKERHUB_USERNAME`/`env.DOCKERHUB_TOKEN` that the workflow never assigns from `secrets`. The `push: tags: v*` trigger publishes with no required-check gate and no dry-run path.
- L4 Contract decision: Linux amd64/arm64/arm-v7 are the only native artifacts (the same set the multi-arch image ships). GHCR is always published; Docker Hub is published only when both `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured. Stable versions publish `X.Y.Z`, `X.Y` and `latest`; prereleases publish `X.Y.Z-rc.N` and `candidate`, never `latest` and never the floating `X.Y` tag, and the GitHub release is marked prerelease and not latest.
- L5 Delivery strategy for the oversized slice: the first work unit (release plan model + CLI, 760 authored lines) exceeds the ~400-line review budget because the version model, the target manifest and the command that exposes them must compile together. The remaining units are smaller (build 266, verify 508, gate 370, version surface + image metadata ~120, workflow ~320, documentation ~300). The feature lands as one branch with one conventional commit per unit; the maintainer decides whether to publish it as a single pull request or as a chained series (the repository already carries the `gentle-ai-chained-pr` convention for slices above the budget).
- L6 Toolchain finding: `go build -trimpath` drops the `-ldflags` build setting from the recorded build information, so the release builder must not use `-trimpath` — the version metadata check reads the linker flags from the build information of every published binary.
