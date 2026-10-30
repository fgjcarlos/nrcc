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
- T2 [S5] inline, done: version surface and image metadata — `nrcc --version` output, `Dockerfile` `VERSION`/`VCS_REF` build args, OCI labels, `main.Version` stamping. Commit: 6145c10.
- T3 [S1, S2, S3, S4, S5, S6, S7, S8, S11] inline, in progress: `.github/workflows/release.yml` rework — Linux-only artifacts, conditional Docker Hub login + image targets, stable/prerelease channels, required-check gate on the tagged commit, dry-run packaging path, artifact verification steps. Commit: this focused work-unit.
- T4 [S3, S5, S8, S11] delegated, done: release contract documentation and local dry-run entry point (`scripts/release/dry-run.sh`, `docs/operations/release.md`, launch-guide checklist, CHANGELOG, `release:dry-run` Taskfile task). Commit: ca92aa4.
- T5 [S1, S2, S3, S4, S5] delegated, done: independent verifier (`gentle-ai-verify`, task `mvuvnsl1-2-kkfk`) — PASS on the points A-H and on S1-S11, one MAJOR finding (a flaky tamper test) and five MINOR findings, all fixed in commit 1b35b09. Commit: 1b35b09.

## Log

- L1 User: "Resuelve la issue #884"
- L2 Issue #884 (approved, P1, area: devops): "The binaries workflow cross-compiles macOS/Windows after ADR 0004 made the project Linux-only. Registry selection and candidate tagging also need to match the supported release contract." Acceptance criteria quoted verbatim in S1-S5.
- L3 Baseline read: `release.yml` builds `nrcc-darwin-amd64`, `nrcc-darwin-arm64`, `nrcc-windows-amd64.exe` (ADR 0004 declares Linux-only), always emits `type=raw,value=latest`, always lists `docker.io/fgjcarlos/nrcc`, and gates the Docker Hub login on `env.DOCKERHUB_USERNAME`/`env.DOCKERHUB_TOKEN` that the workflow never assigns from `secrets`. The `push: tags: v*` trigger publishes with no required-check gate and no dry-run path.
- L4 Contract decision: Linux amd64/arm64/arm-v7 are the only native artifacts (the same set the multi-arch image ships). GHCR is always published; Docker Hub is published only when both `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured. Stable versions publish `X.Y.Z`, `X.Y` and `latest`; prereleases publish `X.Y.Z-rc.N` and `candidate`, never `latest` and never the floating `X.Y` tag, and the GitHub release is marked prerelease and not latest.
- L5 Delivery strategy for the oversized slice: the first work unit (release plan model + CLI, 760 authored lines) exceeds the ~400-line review budget because the version model, the target manifest and the command that exposes them must compile together. The remaining units are smaller (build 266, verify 508, gate 370, version surface + image metadata ~120, workflow ~320, documentation ~300). The feature lands as one branch with one conventional commit per unit; the maintainer decides whether to publish it as a single pull request or as a chained series (the repository already carries the `gentle-ai-chained-pr` convention for slices above the budget).
- L6 Toolchain finding: `go build -trimpath` drops the `-ldflags` build setting from the recorded build information, so the release builder must not use `-trimpath` — the version metadata check reads the linker flags from the build information of every published binary.
- L7 Image metadata verified locally: `docker build --build-arg VERSION=1.2.3 --build-arg VCS_REF=abc1234` produced an image whose `org.opencontainers.image.version` label is `1.2.3`, whose `org.opencontainers.image.revision` label is `abc1234` and whose `nrcc --version` prints `1.2.3`.
- L8 Workflow verified locally: `actionlint v1.7.12` reports no findings for the rewritten `.github/workflows/release.yml`; the release plan output was inspected for a stable version (`1.2.3`, `1.2`, `latest`) and for a prerelease (`1.2.3-rc.1`, `candidate`, no `latest`).
- L9 Workflow corrections after review: the published image is pulled and re-tagged before the verification steps (a multi-architecture push is not loaded into the runner daemon), the binary installation check uses a temporary prefix instead of `/usr/local/bin`, the image revision comes from the same commit the plan resolved, the metadata action publishes only the planned tags (`flavor: latest=false`), and the resolved plan is written to the run summary so the registries and the channels of a run are observed, not asserted.
- L10 Delegated documentation unit (worker `mvusghro-1-hcqv`): `scripts/release/dry-run.sh`, `docs/operations/release.md`, the launch-guide checklist, the CHANGELOG entry and the `release:dry-run` Taskfile task. The worker also reported that `linux/arm/v7` does not compile.
- L11 Verified independently: `GOOS=linux GOARCH=arm GOARM=7 go build .` fails on the untouched tree with `cannot use info.Totalram (variable of type uint32) as uint64 value` (`internal/service/resource_metrics_linux.go:63-65`, same defect in `internal/handler/system_linux.go:17`). The defect is pre-existing (reproducible on `origin/main` a1daeb6) and outside the packaging scope; 32-bit ARM was never a supported platform (CONTRIBUTING.md, "Supported platforms", lists amd64 and arm64 only).
- L12 Contract decision (S1): `linux/arm/v7` is removed from the release contract. S1 allows removing unsupported native targets, and the release must not claim an artifact it cannot build. The published set is now `linux/amd64` and `linux/arm64`, which are the architectures the project supports, tests and ships in the multi-architecture image. The 32-bit compile defect stays a separate follow-up (fixing it means implementing and testing 32-bit ARM, not repackaging).
- L13 Independent verification (task `mvuvnsl1-2-kkfk`): PASS on the supported targets, the registry condition, the channels, the publication gate, the verification, the dry-run path, the documentation consistency and the shared-code regression surface. Findings, all fixed in 1b35b09: MAJOR — the tampered-checksum test tampered the digest only when it did not start with a zero, so the suite was intermittently red (2 failures in 40 runs); MINOR — the Docker Hub login also ran in dry-run mode, the `--source` flag made the build and the verification disagree about the dist path, the gate counted every attempt of a check instead of the latest one, the dry run removed a fixed container name and hid the artifacts it leaves behind, and the documentation promised a `/healthz` status code that only the dry run checked.
- L14 Parent checks after the fixes: `go test -count=2 ./tools/releasepack/...` green, `bash -n scripts/release/dry-run.sh` green, `actionlint v1.7.12` green on the workflow, `go run ./tools/langscan .` reports no findings, and the local dry run ends with `healthz HTTP status: 200`, `The artifacts stay in dist/ and the image stays tagged nrcc:release-verification.` and no leftover container.
