#!/usr/bin/env bash
#
# nrcc release dry run — package and verify the release artifacts locally
# without publishing anything (issue #884).
#
# The script mirrors the Release workflow (.github/workflows/release.yml):
# it resolves the release plan, builds the supported Linux binaries, writes
# the checksums, verifies the artifacts, installs the produced binary and —
# unless it is told to skip it — builds and starts the container image.
#
# It NEVER pushes an image, NEVER creates a release and NEVER writes outside
# the repository: the temporary files live in a mktemp directory that is
# removed on exit.
#
# Usage:
#   scripts/release/dry-run.sh [version] [--skip-frontend] [--skip-image]
#
# Options:
#   --skip-frontend   Reuse the existing frontend/dist instead of building it.
#   --skip-image      Do not build the container image.
#   -h, --help        Show this help.

set -Eeuo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"

# The default version is a development version: it is never published.
readonly DEFAULT_VERSION="0.0.0-dev.1"

# The image reference used by the plan. Docker Hub is deliberately absent:
# the local dry run never talks to a registry.
readonly PLAN_IMAGE="ghcr.io/fgjcarlos/nrcc"

# The local image tag and container name used by the image verification.
readonly IMAGE_TAG="nrcc:release-verification"
readonly CONTAINER_NAME="nrcc-release-verification"

# The maximum time the produced container may take to become healthy.
readonly HEALTH_TIMEOUT_SECONDS=120

version="${DEFAULT_VERSION}"
version_set="false"
skip_frontend="false"
skip_image="false"
dist_dir="dist"
temp_dir=""
container_started="false"

# ── Output helpers ──────────────────────────────────────────────────────────

log() {
  printf '%s\n' "$*"
}

step() {
  printf '\n==> %s\n' "$*"
}

fail() {
  printf 'release dry run: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage: scripts/release/dry-run.sh [version] [options]

Packages and verifies the nrcc release artifacts locally. Nothing is
published: no image is pushed and no release is created.

Arguments:
  version            Version to package (default: 0.0.0-dev.1).

Options:
  --skip-frontend    Reuse the existing frontend/dist instead of building it.
  --skip-image       Do not build the container image.
  -h, --help         Show this help.
EOF
}

# ── Traps ───────────────────────────────────────────────────────────────────

on_error() {
  local exit_code=$?
  printf 'release dry run: failed at line %s (exit code %s).\n' "${BASH_LINENO[0]:-unknown}" "${exit_code}" >&2
  exit "${exit_code}"
}

cleanup() {
  if [ "${container_started}" = "true" ]; then
    docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    container_started="false"
  fi
  if [ -n "${temp_dir}" ] && [ -d "${temp_dir}" ]; then
    rm -rf -- "${temp_dir}"
    temp_dir=""
  fi
}

trap on_error ERR
trap cleanup EXIT

# ── Arguments ───────────────────────────────────────────────────────────────

while [ "$#" -gt 0 ]; do
  case "$1" in
    --skip-frontend)
      skip_frontend="true"
      ;;
    --skip-image)
      skip_image="true"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      break
      ;;
    -*)
      printf 'release dry run: unknown option %s.\n' "$1" >&2
      usage >&2
      exit 2
      ;;
    *)
      if [ "${version_set}" = "true" ]; then
        fail "unexpected argument '$1' — only one version may be given."
      fi
      version="$1"
      version_set="true"
      ;;
  esac
  shift
done

if [ "$#" -gt 0 ]; then
  fail "unexpected argument '$1'."
fi

command -v go >/dev/null 2>&1 || fail "go is not installed."
command -v git >/dev/null 2>&1 || fail "git is not installed."

temp_dir="$(mktemp -d)"
log "Temporary directory: ${temp_dir}"
log "Repository: ${REPO_ROOT}"
log "Version: ${version}"

cd "${REPO_ROOT}"

# ── Step 1 — build the frontend ─────────────────────────────────────────────

if [ "${skip_frontend}" = "true" ]; then
  step "Reusing the existing frontend/dist (--skip-frontend)"
  if [ ! -d "frontend/dist" ]; then
    fail "frontend/dist does not exist — run without --skip-frontend or build the frontend first."
  fi
else
  step "Building the frontend"
  command -v pnpm >/dev/null 2>&1 || fail "pnpm is not installed."
  (
    cd frontend
    pnpm install --frozen-lockfile
    pnpm build
  )
fi

# ── Step 2 — resolve the release plan ───────────────────────────────────────

step "Resolving the release plan"
if ! plan_json="$(go run ./tools/releasepack plan \
  --version "${version}" \
  --image "${PLAN_IMAGE}" \
  --format json)"; then
  fail "the release plan could not be resolved for version '${version}'."
fi
printf '%s\n' "${plan_json}"

# The plan carries the normalised version (for example "1.2.3" for the
# input "v1.2.3"). Every verification below compares against it, exactly
# like the workflow does with the plan outputs.
expected_version="$(printf '%s\n' "${plan_json}" | sed -n 's/^  "version": "\(.*\)",\{0,1\}$/\1/p')"
if [ -z "${expected_version}" ]; then
  fail "the release plan does not carry a version — the releasepack tool changed?"
fi
log "Expected version: ${expected_version}"

# ── Step 3 — build the artifacts ────────────────────────────────────────────

step "Building the release artifacts"
if ! go run ./tools/releasepack build --version "${version}" --dist "${dist_dir}"; then
  fail "the release artifacts could not be built for version '${version}'."
fi

# ── Step 4 — verify the artifacts ───────────────────────────────────────────

step "Verifying the release artifacts"
if ! go run ./tools/releasepack verify --version "${version}" --dist "${dist_dir}"; then
  fail "the release artifacts did not pass the verification for version '${version}'."
fi

# ── Step 5 — install the produced binary ────────────────────────────────────

step "Installing the produced binary on a clean prefix"
prefix="${temp_dir}/prefix"
mkdir -p "${prefix}/bin"
install -m 0755 "${dist_dir}/nrcc-linux-amd64" "${prefix}/bin/nrcc"
reported="$("${prefix}/bin/nrcc" --version)"
if [ "${reported}" != "${expected_version}" ]; then
  fail "the installed binary reported version '${reported}', expected '${expected_version}'."
fi
log "The installed binary reported ${reported}."

# ── Step 6 — build and verify the container image ───────────────────────────

if [ "${skip_image}" = "true" ]; then
  step "Skipping the container image (--skip-image)"
  log "Dry run finished: the artifacts were packaged and verified, nothing was published."
  exit 0
fi

command -v docker >/dev/null 2>&1 || fail "docker is not installed."

step "Building the container image"
revision="$(git rev-parse HEAD 2>/dev/null || true)"
if [ -z "${revision}" ]; then
  fail "cannot resolve the current git revision — run the dry run inside a git checkout."
fi
docker build \
  --build-arg "VERSION=${expected_version}" \
  --build-arg "VCS_REF=${revision}" \
  -t "${IMAGE_TAG}" \
  -f Dockerfile \
  .

step "Verifying the image version metadata"
image_version="$(docker image inspect --format '{{ index .Config.Labels "org.opencontainers.image.version" }}' "${IMAGE_TAG}")"
if [ "${image_version}" != "${expected_version}" ]; then
  fail "the image carries version '${image_version}', expected '${expected_version}'."
fi
log "The image carries version ${image_version}."

step "Verifying the image reports the release version"
image_reported="$(docker run --rm --entrypoint /usr/local/bin/nrcc "${IMAGE_TAG}" --version)"
if [ "${image_reported}" != "${expected_version}" ]; then
  fail "the image reported version '${image_reported}', expected '${expected_version}'."
fi
log "The image reported ${image_reported}."

step "Starting the produced image"
# A leftover container of a previous run would make the start fail, so it is
# removed first. The name belongs to this script.
docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
container_started="true"
docker run -d --name "${CONTAINER_NAME}" \
  -e NRCC_ENCRYPTION_KEY=release-verification-encryption-key-not-a-secret \
  -e JWT_SECRET=release-verification-jwt-secret-not-a-production-value \
  "${IMAGE_TAG}" >/dev/null

health_status="unknown"
deadline=$(( SECONDS + HEALTH_TIMEOUT_SECONDS ))
while [ "${SECONDS}" -lt "${deadline}" ]; do
  health_status="$(docker inspect --format '{{ .State.Health.Status }}' "${CONTAINER_NAME}" 2>/dev/null || echo unknown)"
  case "${health_status}" in
    healthy|unhealthy)
      break
      ;;
  esac
  sleep 3
done

log "healthz result: ${health_status}"
if [ "${health_status}" != "healthy" ]; then
  docker logs "${CONTAINER_NAME}" || true
  fail "the produced image did not become healthy (status: ${health_status})."
fi

# The health check of the image queries /healthz, which answers with an
# empty 200 body. The HTTP status is the meaningful result.
healthz_code="$(docker exec "${CONTAINER_NAME}" wget -q -S -O /dev/null http://localhost:3001/healthz 2>&1 |
  awk '/  HTTP\//{code=$2} END{print code}')"
if [ "${healthz_code}" != "200" ]; then
  fail "the produced image did not answer /healthz with 200 (status: ${healthz_code:-unknown})."
fi
log "healthz HTTP status: ${healthz_code}"

# The container is removed by the cleanup trap.
log "Dry run finished: the artifacts were packaged and verified, nothing was published."
