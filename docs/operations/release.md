# Release contract

This document describes what a nrcc release publishes, when it is allowed
to publish and how it is verified. It is the operator-facing description
of the contract implemented by [`.github/workflows/release.yml`](../../.github/workflows/release.yml)
and by the packaging tool [`tools/releasepack`](../../tools/releasepack).

Publishing itself is a maintainer action: the pipeline aligns and verifies
the artifacts, the tag is what triggers the publication.

## Supported artifacts

The release publishes two native binaries and nothing else:

| Target | Binary |
| --- | --- |
| `linux/amd64` | `nrcc-linux-amd64` |
| `linux/arm64` | `nrcc-linux-arm64` |

Those are the only supported native targets because nrcc is Linux-only at
compile time: [ADR 0004](../adr/0004-linux-only-build.md) removed the
Windows and macOS fallbacks and kept the POSIX-only syscall usage behind a
`//go:build linux` constraint. Publishing a binary for another platform
would ship a file that cannot even be compiled.

The two Linux architectures are the ones the project supports and tests
([CONTRIBUTING.md](../../CONTRIBUTING.md), "Supported platforms") and are
exactly the ones the multi-architecture container image ships, so the
release artifacts and the image cover the same platforms.

`linux/arm/v7` (32-bit ARM) is not published: the backend does not
compile for 32-bit platforms, because `syscall.Sysinfo_t` exposes
`Totalram` and `Freeram` as `uint32` there while the metrics code
assigns them to `uint64` fields. Supporting 32-bit ARM means
implementing and testing that platform, and a release must not claim an
artifact it cannot build.

The published set is fixed in `tools/releasepack` and the verification step
rejects any artifact of an unsupported target that sneaks into the
distribution directory.

## Registries

- `ghcr.io/<owner>/nrcc` is always published. It is the reference image of
  the project.
- `docker.io/<owner>/nrcc` is published only when both `DOCKERHUB_USERNAME`
  and `DOCKERHUB_TOKEN` are configured as repository secrets.

The Docker Hub login and the Docker Hub image target share one condition:
the pipeline neither logs in nor publishes the `docker.io` image when the
credentials are missing. A registry is never pushed to without a login, and
a successful login never implies a publication that was not intended.

When the credentials are absent the workflow logs the reason and publishes
to GHCR only.

## Channels

A stable version `X.Y.Z` publishes three Docker tags:

- `X.Y.Z` — the immutable version tag.
- `X.Y` — the floating minor tag.
- `latest` — the floating stable tag.

A prerelease `X.Y.Z-rc.N` publishes two Docker tags:

- `X.Y.Z-rc.N` — the immutable version tag.
- `candidate` — the floating prerelease tag.

A prerelease never takes over `latest` and never moves the floating `X.Y`
tag: the stable series keeps pointing at the last stable release. The
matching GitHub release is marked as a prerelease and is not published as
the latest release.

The channel, the tags and the image labels are computed by
`releasepack plan`, so the workflow and the local dry run always agree.

## Publication gate

Nothing is published before the required checks of the exact tagged commit
have succeeded. The gate evaluates the check runs and the commit statuses
of the tagged commit — not of the branch head — and a missing, unfinished or
failed required check blocks the publication.

The required checks are:

- `Backend (Go 1.26.9)`
- `Frontend (Node 22)`
- `OpenAPI Spec`
- `Lint (golangci-lint)`
- `Docker stack acceptance (#434)`
- `govulncheck`
- `pnpm audit`
- `Trivy container scan`

The list mirrors the job names of the CI workflows. Renaming a job in
CI renames it here: both changes belong in the same commit, otherwise the
gate blocks every publication.

Two rules of the evaluation are worth knowing:

- A check that was skipped or that reported no conclusion counts as
  satisfied, exactly as GitHub counts it for a required status check. A
  job that the fork guard skips therefore never blocks a publication.
- Only the latest attempt of a check counts: re-running a failed job
  replaces the failure it retries. Every leg of the latest attempt of a
  matrix job still has to pass.

## Dry run

The packaging path runs locally without publishing anything. It is the same
code path the workflow uses, so a green dry run is a meaningful prediction
of the release.

Locally:

```bash
scripts/release/dry-run.sh                 # default version 0.0.0-dev.1
scripts/release/dry-run.sh 1.2.3           # package a specific version
scripts/release/dry-run.sh --skip-frontend # reuse the existing frontend/dist
scripts/release/dry-run.sh --skip-image    # do not build the container image
```

Or through the Taskfile:

```bash
task release:dry-run
```

The script never pushes an image, never creates a release and never writes
outside the repository: the temporary files live in a `mktemp` directory
that is removed on exit, and the container it starts is removed by a cleanup
trap.

In GitHub Actions:

1. Open `Actions → Release → Run workflow`.
2. Keep the `dry_run` input enabled and set `version` to the version to
   package (the dispatched ref name is used when it is empty).
3. Watch the run: it packages and verifies the artifacts, builds the image
   for the host architecture and loads it locally, and publishes nothing.

A dispatched run with `dry_run` disabled publishes; that is the only way
to publish without a tag, and it is a maintainer action.

## What is verified before a publication

- **Checksums.** `SHA256SUMS` holds the digest of every published binary,
  and each binary has a `*.sha256` sidecar next to it. The verification
  recomputes both and compares them.
- **Version metadata of the binaries.** Every binary carries the release
  version, stamped at link time and read back from the build information
  of the produced file.
- **Version metadata of the image.** The `org.opencontainers.image.version`
  label of the produced image equals the release version, and
  `nrcc --version` inside the image prints the very same version.
- **Installation of the produced artifacts.** The produced binary is
  installed on a clean prefix and must report the release version, and the
  produced image is started and must become healthy — its `/healthz`
  endpoint has to answer with `200`.
The local dry run performs the same four checks.

## Cutting a release

1. Make sure `main` is green and that the version is free.
2. Tag the release commit on `main`:

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
```

3. Push the tag:

```bash
git push origin vX.Y.Z
```

4. Watch the `Release` workflow. It resolves the plan, evaluates the
   required checks of the tagged commit, packages and verifies the
   artifacts, publishes the binaries to the GitHub release and pushes the
   image to the enabled registries.
5. Verify the published release:
   - the GitHub release for the tag lists `nrcc-linux-amd64`,
     `nrcc-linux-arm64`, `SHA256SUMS` and the two `*.sha256` sidecars,
     and nothing else;
   - `sha256sum -c SHA256SUMS` succeeds on the downloaded artifacts;
   - the image tags of the published version are the ones of its channel
     (`X.Y.Z`, `X.Y` and `latest` for a stable release, `X.Y.Z-rc.N` and
     `candidate` for a prerelease);
   - `docker run --rm --entrypoint /usr/local/bin/nrcc <image> --version`
     prints the released version.

## Related

- [`.github/workflows/release.yml`](../../.github/workflows/release.yml) —
  the pipeline that implements this contract.
- [`tools/releasepack`](../../tools/releasepack/main.go) — the plan, build, verify
  and gate implementation shared by the pipeline and the dry run.
- [`scripts/release/dry-run.sh`](../../scripts/release/dry-run.sh) — the
  local dry run.
- [ADR 0004 — Linux-only build](../adr/0004-linux-only-build.md) — why
  Linux is the only supported platform.
- [ADR 0003 — Docker-first deployment model](../adr/0003-docker-first-one-stack-per-node-red.md)
  — why the container image is the canonical deployment.
- [Production installation and launch guide](../production-install-launch-guide.md)
  — the operator checklist of a release or a test tag.
