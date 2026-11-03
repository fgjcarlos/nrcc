#!/usr/bin/env bash
#
# package.sh — build the supported Linux release artifacts for nrcc.
#
# Usage: scripts/release/package.sh VERSION OUTPUT_DIR
#
# Outputs (in OUTPUT_DIR):
#   nrcc-linux-amd64      static linux/amd64 binary
#   nrcc-linux-arm64      static linux/arm64 binary
#   nrcc-linux-armv7      static linux/arm (GOARM=7) binary
#   SHA256SUMS            digest manifest over the three binaries
#   nrcc-linux-<arch>.sha256
#                        per-binary sidecar using the normalized "nrcc"
#                        filename expected by the installer conventions
#
# nrcc is Docker-first (ADR 0003) and Linux-only at compile time
# (ADR 0004), so no darwin/windows targets are produced here.
#
# Exit codes:
#   0 — artifacts built and checksummed
#   1 — build failure
#   2 — usage error
#
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"

readonly ARTIFACT_AMD64="nrcc-linux-amd64"
readonly ARTIFACT_ARM64="nrcc-linux-arm64"
readonly ARTIFACT_ARMV7="nrcc-linux-armv7"

# Accepted version characters: semver-ish tags ("v1.2.3", "1.2.3-rc.1")
# plus the "dev" default. The value is embedded through -ldflags, so it
# must never contain whitespace, quotes or shell metacharacters.
readonly VERSION_PATTERN='^[A-Za-z0-9][A-Za-z0-9._+-]*$'

log() {
    printf '[package] %s\n' "$*"
}

die() {
    printf '[package] ERROR: %s\n' "$*" >&2
    exit 1
}

usage() {
    cat >&2 <<'EOF'
Usage: scripts/release/package.sh VERSION OUTPUT_DIR

Builds the supported Linux release artifacts into OUTPUT_DIR.

Arguments:
  VERSION     Version embedded in the binaries (e.g. v1.2.3, v1.2.3-rc.1, dev).
  OUTPUT_DIR  Directory that receives the binaries, SHA256SUMS and *.sha256.

Environment:
  NRCC_SKIP_FRONTEND_BUILD=1   Fail instead of building a missing frontend/dist.
EOF
}

# validate_version VERSION — reject anything that is unsafe to embed.
validate_version() {
    local version="$1"
    if [[ -z "${version}" ]]; then
        printf '[package] ERROR: VERSION must not be empty.\n' >&2
        return 1
    fi
    if [[ ! "${version}" =~ ${VERSION_PATTERN} ]]; then
        printf '[package] ERROR: invalid VERSION %q — expected something like v1.2.3 (letters, digits, ".", "_", "+", "-").\n' "${version}" >&2
        return 1
    fi
    return 0
}

# ensure_frontend_dist — the Go binary embeds frontend/dist (embed.go),
# so the compiled frontend must exist before the Go build.
ensure_frontend_dist() {
    if [[ -f "${REPO_ROOT}/frontend/dist/index.html" ]]; then
        log "frontend/dist present — reusing it"
        return 0
    fi
    if [[ "${NRCC_SKIP_FRONTEND_BUILD:-0}" == "1" ]]; then
        die "frontend/dist/index.html is missing and NRCC_SKIP_FRONTEND_BUILD=1"
    fi
    if ! command -v pnpm >/dev/null 2>&1; then
        die "frontend/dist/index.html is missing and pnpm is not available — build the frontend first"
    fi
    log "frontend/dist missing — building the frontend with pnpm"
    (
        cd -- "${REPO_ROOT}/frontend"
        pnpm install --frozen-lockfile
        pnpm build
    ) || die "frontend build failed"
    [[ -f "${REPO_ROOT}/frontend/dist/index.html" ]] || die "frontend build did not produce frontend/dist/index.html"
}

# build_binary GOARCH GOARM OUTPUT_PATH — static Linux build with the
# version embedded in main.Version.
build_binary() {
    local goarch="$1"
    local goarm="${2:-}"
    local output="$3"
    local -a env_args=("CGO_ENABLED=0" "GOOS=linux" "GOARCH=${goarch}")
    if [[ -n "${goarm}" ]]; then
        env_args+=("GOARM=${goarm}")
    fi
    log "building ${output} (linux/${goarch}${goarm:+ v${goarm}})"
    (
        cd -- "${REPO_ROOT}"
        env "${env_args[@]}" go build \
            -tags netgo \
            -ldflags="-s -w -X main.Version=${VERSION} -extldflags=-static" \
            -o "${output}" \
            .
    ) || die "go build failed for linux/${goarch}${goarm:+ v${goarm}}"
}

main() {
    if [[ $# -ne 2 ]]; then
        usage
        exit 2
    fi

    local version="$1"
    local output_dir="$2"

    validate_version "${version}" || { usage; exit 2; }

    if [[ -z "${output_dir}" ]]; then
        printf '[package] ERROR: OUTPUT_DIR must not be empty.\n' >&2
        usage
        exit 2
    fi

    VERSION="${version}"
    export VERSION

    command -v go >/dev/null 2>&1 || die "the go toolchain is required"
    command -v sha256sum >/dev/null 2>&1 || die "sha256sum is required"

    ensure_frontend_dist

    mkdir -p -- "${output_dir}" || die "cannot create output directory: ${output_dir}"
    output_dir="$(cd -- "${output_dir}" && pwd -P)"

    build_binary "amd64" "" "${output_dir}/${ARTIFACT_AMD64}"
    build_binary "arm64" "" "${output_dir}/${ARTIFACT_ARM64}"
    # GOARM=7 keeps the armv7 ABI aligned with the linux/arm/v7 image.
    build_binary "arm" "7" "${output_dir}/${ARTIFACT_ARMV7}"

    local -a artifacts=("${ARTIFACT_AMD64}" "${ARTIFACT_ARM64}" "${ARTIFACT_ARMV7}")

    (
        cd -- "${output_dir}"
        sha256sum "${artifacts[@]}" > SHA256SUMS
    ) || die "cannot write SHA256SUMS"

    local artifact
    for artifact in "${artifacts[@]}"; do
        (
            cd -- "${output_dir}"
            # Normalized sidecar: "<digest>  nrcc" — the filename the
            # installer conventions expect.
            sha256sum "${artifact}" | sed 's/nrcc-.*/nrcc/' > "${artifact}.sha256"
        ) || die "cannot write ${artifact}.sha256"
    done

    log "artifacts written to ${output_dir}"
    for artifact in "${artifacts[@]}"; do
        log "  ${artifact}"
    done
    log "  SHA256SUMS"
    log "  *.sha256"
}

main "$@"
