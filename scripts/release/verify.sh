#!/usr/bin/env bash
#
# verify.sh — verify a packaged nrcc release directory.
#
# Usage: scripts/release/verify.sh VERSION OUTPUT_DIR
#
# The verifier checks, in order:
#   1. every expected artifact is present (three Linux binaries, the
#      SHA256SUMS manifest and the normalized *.sha256 sidecars);
#   2. the digests recorded in SHA256SUMS and in the sidecars match the
#      bytes actually on disk (re-computed, not trusted);
#   3. the Go build metadata embedded in every binary reports the
#      requested version and the expected GOOS/GOARCH/GOARM target;
#   4. the native binary installs into a fresh temporary directory,
#      starts an isolated server, answers /healthz, prints the version
#      banner, and leaves no process or state behind.
#
# Exit codes:
#   0 — every check passed
#   1 — a check failed
#   2 — usage error
#
set -Eeuo pipefail

readonly ARTIFACT_AMD64="nrcc-linux-amd64"
readonly ARTIFACT_ARM64="nrcc-linux-arm64"
readonly ARTIFACT_ARMV7="nrcc-linux-armv7"

readonly VERSION_PATTERN='^[A-Za-z0-9][A-Za-z0-9._+-]*$'

SMOKE_DIR=""
SMOKE_PID=""

log() {
    printf '[verify] %s\n' "$*"
}

ok() {
    printf '[verify] OK: %s\n' "$*"
}

fail() {
    printf '[verify] FAIL: %s\n' "$*" >&2
    exit 1
}

usage() {
    cat >&2 <<'EOF'
Usage: scripts/release/verify.sh VERSION OUTPUT_DIR

Verifies a directory produced by scripts/release/package.sh.

Arguments:
  VERSION     Version that must be embedded in the binaries (e.g. v1.2.3).
  OUTPUT_DIR  Directory holding the binaries, SHA256SUMS and *.sha256.
EOF
}

cleanup() {
    local status=$?
    if [[ -n "${SMOKE_PID}" ]] && kill -0 "${SMOKE_PID}" 2>/dev/null; then
        kill -TERM "${SMOKE_PID}" 2>/dev/null || true
        local waited=0
        while kill -0 "${SMOKE_PID}" 2>/dev/null && [[ "${waited}" -lt 50 ]]; do
            sleep 0.2
            waited=$((waited + 1))
        done
        kill -KILL "${SMOKE_PID}" 2>/dev/null || true
        wait "${SMOKE_PID}" 2>/dev/null || true
    fi
    if [[ -n "${SMOKE_DIR}" && -d "${SMOKE_DIR}" ]]; then
        case "${SMOKE_DIR}" in
            */nrcc-release-smoke.*) rm -rf -- "${SMOKE_DIR}" || true ;;
        esac
    fi
    exit "${status}"
}
trap cleanup EXIT

# build_setting METADATA KEY — read one build setting out of `go version -m`.
build_setting() {
    printf '%s\n' "$1" | sed -n "s/^[[:space:]]*build[[:space:]]\{1,\}$2=\([^[:space:]]*\).*/\1/p" | head -n 1
}

# build_ldflags METADATA — read the -ldflags build setting.
build_ldflags() {
    printf '%s\n' "$1" | sed -n 's/^[[:space:]]*build[[:space:]]\{1,\}-ldflags="\(.*\)"$/\1/p' | head -n 1
}

sha256_of() {
    sha256sum "$1" | awk '{print $1}'
}

http_ok() {
    local url="$1"
    if command -v curl >/dev/null 2>&1; then
        curl -fsS -o /dev/null --max-time 5 "${url}" >/dev/null 2>&1
    elif command -v python3 >/dev/null 2>&1; then
        python3 - "${url}" <<'PY' >/dev/null 2>&1
import sys
import urllib.request

try:
    with urllib.request.urlopen(sys.argv[1], timeout=5) as response:
        sys.exit(0 if response.status == 200 else 1)
except Exception:  # noqa: BLE001 — any failure means "not healthy yet"
    sys.exit(1)
PY
    else
        return 1
    fi
}

pick_free_port() {
    if command -v python3 >/dev/null 2>&1; then
        python3 - <<'PY'
import socket

sock = socket.socket()
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("127.0.0.1", 0))
print(sock.getsockname()[1])
sock.close()
PY
        return 0
    fi
    local port
    local _
    for _ in $(seq 1 50); do
        port=$(( 20000 + (RANDOM % 20000) ))
        if ! (exec 3<>"/dev/tcp/127.0.0.1/${port}") 2>/dev/null; then
            printf '%s\n' "${port}"
            return 0
        fi
    done
    return 1
}

check_artifacts_present() {
    local output_dir="$1"
    local artifact
    for artifact in "${ARTIFACT_AMD64}" "${ARTIFACT_ARM64}" "${ARTIFACT_ARMV7}"; do
        [[ -f "${output_dir}/${artifact}" ]] || fail "missing artifact: ${artifact}"
        [[ -f "${output_dir}/${artifact}.sha256" ]] || fail "missing checksum sidecar: ${artifact}.sha256"
    done
    [[ -f "${output_dir}/SHA256SUMS" ]] || fail "missing checksum manifest: SHA256SUMS"
    ok "all expected artifacts are present"
}

check_checksums() {
    local output_dir="$1"
    local artifact
    local expected
    local actual
    local sidecar_digest
    local sidecar_name

    (
        cd -- "${output_dir}"
        sha256sum --quiet -c SHA256SUMS >/dev/null
    ) || fail "SHA256SUMS does not match the artifacts on disk (checksum mismatch)"

    for artifact in "${ARTIFACT_AMD64}" "${ARTIFACT_ARM64}" "${ARTIFACT_ARMV7}"; do
        expected="$(awk -v name="${artifact}" '$2 == name {print $1; exit}' "${output_dir}/SHA256SUMS")"
        [[ -n "${expected}" ]] || fail "SHA256SUMS has no entry for ${artifact}"
        actual="$(sha256_of "${output_dir}/${artifact}")"
        [[ "${actual}" == "${expected}" ]] || fail "checksum mismatch for ${artifact}: manifest ${expected}, actual ${actual}"

        read -r sidecar_digest sidecar_name < "${output_dir}/${artifact}.sha256" || \
            fail "cannot read sidecar ${artifact}.sha256"
        [[ "${sidecar_digest}" == "${actual}" ]] || \
            fail "checksum mismatch in sidecar ${artifact}.sha256: ${sidecar_digest} != ${actual}"
        [[ "${sidecar_name}" == "nrcc" ]] || \
            fail "sidecar ${artifact}.sha256 must use the normalized filename 'nrcc', got '${sidecar_name}'"
    done
    ok "SHA256SUMS and the *.sha256 sidecars match the actual artifact bytes"
}

check_metadata() {
    local output_dir="$1"
    local version="$2"
    local artifact
    local goarch
    local goarm
    local metadata
    local settings_goos
    local settings_goarch
    local settings_goarm
    local settings_cgo
    local ldflags

    command -v go >/dev/null 2>&1 || fail "the go toolchain is required to read the embedded build metadata"

    for artifact in "${ARTIFACT_AMD64}:amd64:" "${ARTIFACT_ARM64}:arm64:" "${ARTIFACT_ARMV7}:arm:7"; do
        IFS=':' read -r artifact goarch goarm <<< "${artifact}"

        metadata="$(go version -m "${output_dir}/${artifact}")" || \
            fail "cannot read the build metadata of ${artifact}"
        settings_goos="$(build_setting "${metadata}" "GOOS")"
        settings_goarch="$(build_setting "${metadata}" "GOARCH")"
        settings_goarm="$(build_setting "${metadata}" "GOARM")"
        settings_cgo="$(build_setting "${metadata}" "CGO_ENABLED")"
        ldflags="$(build_ldflags "${metadata}")"

        [[ "${settings_goos}" == "linux" ]] || fail "${artifact}: expected GOOS=linux, got '${settings_goos:-<unset>}'"
        [[ "${settings_goarch}" == "${goarch}" ]] || fail "${artifact}: expected GOARCH=${goarch}, got '${settings_goarch:-<unset>}'"
        if [[ -n "${goarm}" ]]; then
            [[ "${settings_goarm}" == "${goarm}" ]] || fail "${artifact}: expected GOARM=${goarm}, got '${settings_goarm:-<unset>}'"
        else
            [[ -z "${settings_goarm}" ]] || fail "${artifact}: expected no GOARM setting, got '${settings_goarm}'"
        fi
        [[ "${settings_cgo}" == "0" ]] || fail "${artifact}: expected a static build (CGO_ENABLED=0), got '${settings_cgo:-<unset>}'"
        [[ " ${ldflags} " == *" main.Version=${version} "* ]] || \
            fail "${artifact}: embedded version mismatch — expected main.Version=${version}, got '${ldflags:-<unset>}'"
    done
    ok "embedded build metadata reports version ${version} and the expected Linux targets"
}

banner_reports_version() {
    # Remove terminal colors, then compare the complete value token.
    sed -E 's/\x1B\[[0-9;]*[[:alpha:]]//g' "$1" |
        awk -v expected="$2" '{
            for (i = 1; i < NF; i++) {
                if ($i == "Version:" && $(i + 1) == expected) found = 1
            }
        } END { exit !found }'
}

run_native_smoke() {
    local output_dir="$1"
    local version="$2"

    local host_arch
    host_arch="$(go env GOHOSTARCH 2>/dev/null || uname -m)"
    local artifact
    case "${host_arch}" in
        amd64) artifact="${ARTIFACT_AMD64}" ;;
        arm64) artifact="${ARTIFACT_ARM64}" ;;
        arm) artifact="${ARTIFACT_ARMV7}" ;;
        *)
            log "SKIP: host architecture '${host_arch}' has no native artifact (metadata checks still ran)"
            return 0
            ;;
    esac

    local install_dir
    local data_dir
    local home_dir
    local port
    local deadline
    local healthy=0
    local waited
    local banner_stream="stdout"

    SMOKE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/nrcc-release-smoke.XXXXXX")" || fail "cannot create a temporary directory"
    install_dir="${SMOKE_DIR}/bin"
    data_dir="${SMOKE_DIR}/data"
    home_dir="${SMOKE_DIR}/home"
    mkdir -p "${install_dir}" "${data_dir}" "${home_dir}"

    # "Install" the artifact into a fresh bin directory.
    cp -- "${output_dir}/${artifact}" "${install_dir}/nrcc" || fail "cannot install ${artifact}"
    chmod 0755 "${install_dir}/nrcc"

    port="$(pick_free_port)" || fail "cannot find a free TCP port"
    log "starting the installed ${artifact} on 127.0.0.1:${port}"

    env PATH="${install_dir}:${PATH}" \
        DATA_DIR="${data_dir}" \
        HOME="${home_dir}" \
        PORT="${port}" \
        NRCC_MANAGE_NODE_RED=false \
        "${install_dir}/nrcc" >"${SMOKE_DIR}/stdout.log" 2>"${SMOKE_DIR}/stderr.log" &
    SMOKE_PID=$!

    deadline=$((SECONDS + 30))
    while [[ "${SECONDS}" -lt "${deadline}" ]]; do
        if ! kill -0 "${SMOKE_PID}" 2>/dev/null; then
            fail "the installed binary exited before answering /healthz — $(tail -n 5 "${SMOKE_DIR}/stderr.log" 2>/dev/null || true)"
        fi
        if http_ok "http://127.0.0.1:${port}/healthz"; then
            healthy=1
            break
        fi
        sleep 0.3
    done
    [[ "${healthy}" -eq 1 ]] || fail "the installed binary did not answer /healthz on 127.0.0.1:${port}"
    ok "the installed ${artifact} answers /healthz"

    if ! banner_reports_version "${SMOKE_DIR}/stdout.log" "${version}"; then
        if banner_reports_version "${SMOKE_DIR}/stderr.log" "${version}"; then
            banner_stream="stderr"
        else
            fail "the startup banner does not report 'Version: ${version}' — $(tail -n 20 "${SMOKE_DIR}/stdout.log" 2>/dev/null || true)"
        fi
    fi
    ok "the startup banner reports Version: ${version} (${banner_stream})"

    [[ -f "${data_dir}/jwt_secret" ]] || fail "the isolated run did not create its state under ${data_dir}"

    # Shut the server down and make sure nothing is left running.
    kill -TERM "${SMOKE_PID}" 2>/dev/null || true
    waited=0
    while kill -0 "${SMOKE_PID}" 2>/dev/null && [[ "${waited}" -lt 50 ]]; do
        sleep 0.2
        waited=$((waited + 1))
    done
    if kill -0 "${SMOKE_PID}" 2>/dev/null; then
        fail "the installed binary did not stop after SIGTERM"
    fi
    wait "${SMOKE_PID}" 2>/dev/null || true
    SMOKE_PID=""
    ok "the isolated server stopped cleanly"

    if command -v pgrep >/dev/null 2>&1 && pgrep -f "${install_dir}/nrcc" >/dev/null 2>&1; then
        fail "a leftover nrcc process is still running from ${install_dir}"
    fi
    ok "no leftover processes"

    # Remove the isolated state and prove the cleanup happened.
    rm -rf -- "${SMOKE_DIR}" || fail "cannot remove the temporary directory ${SMOKE_DIR}"
    [[ ! -e "${SMOKE_DIR}" ]] || fail "the temporary directory ${SMOKE_DIR} still exists"
    SMOKE_DIR=""
    ok "the isolated state was removed"
}

main() {
    if [[ $# -ne 2 ]]; then
        usage
        exit 2
    fi

    local version="$1"
    local output_dir="$2"

    if [[ -z "${version}" || ! "${version}" =~ ${VERSION_PATTERN} ]]; then
        printf '[verify] ERROR: invalid VERSION %q\n' "${version}" >&2
        usage
        exit 2
    fi
    if [[ -z "${output_dir}" ]]; then
        printf '[verify] ERROR: OUTPUT_DIR must not be empty.\n' >&2
        usage
        exit 2
    fi
    [[ -d "${output_dir}" ]] || fail "output directory does not exist: ${output_dir}"
    output_dir="$(cd -- "${output_dir}" && pwd -P)"

    command -v sha256sum >/dev/null 2>&1 || fail "sha256sum is required"

    check_artifacts_present "${output_dir}"
    check_checksums "${output_dir}"
    check_metadata "${output_dir}" "${version}"
    run_native_smoke "${output_dir}" "${version}"

    log "all checks passed for version ${version}"
}

main "$@"
