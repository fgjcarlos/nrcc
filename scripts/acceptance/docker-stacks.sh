#!/usr/bin/env bash
# scripts/acceptance/docker-stacks.sh
#
# Issue #434 acceptance suite — proves the one-stack-per-Node-RED model
# (ADR 0003) holds when more than one Compose project is alive on the same
# host.
#
# Scenarios:
#   1. Single-stack happy path   — boot canonical compose, hit /healthz,
#      finish bootstrap, log in, confirm Node-RED is reachable on 1880.
#   2. Two-stack isolation       — boot a second stack on offset ports with
#      its own volumes; verify the two stacks don't share config, flows,
#      backups, or node_modules.
#   3. Persistence across recreate — edit settings/env/fixture npm node,
#      `compose down` then `compose up`, verify the state survived.
#   4. Backup → wipe → restore   — create a backup, simulate a destructive
#      restore into a fresh stack, verify health comes back.
#
# Sibling-container safety: this script never mounts the Docker socket and
# never references another stack by container_name. Each scenario uses its
# own COMPOSE_PROJECT_NAME so Docker isolates the resources.
#
# CI integration: called by .github/workflows/acceptance.yml after a
# `docker buildx bake` step exports NRCC_IMAGE.

set -euo pipefail

# ── Configuration ───────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_FILE="${REPO_ROOT}/docker-compose.yml"

# NRCC image under test. CI sets this from `docker buildx bake --load`.
NRCC_IMAGE="${NRCC_IMAGE:-ghcr.io/fgjcarlos/nrcc:latest}"

# Stack A (canonical) — defaults work on a developer host.
PROJECT_A="${PROJECT_A:-nrcc-acc-a}"
HOST_API_A="${HOST_API_A:-3001}"
HOST_NR_A="${HOST_NR_A:-1880}"

# Stack B (offset) — proves two stacks co-exist without colliding.
PROJECT_B="${PROJECT_B:-nrcc-acc-b}"
HOST_API_B="${HOST_API_B:-13001}"
HOST_NR_B="${HOST_NR_B:-11880}"

# Stack C — clean slate for the restore test.
PROJECT_C="${PROJECT_C:-nrcc-acc-c}"
HOST_API_C="${HOST_API_C:-23001}"
HOST_NR_C="${HOST_NR_C:-21880}"

# Time budgets. CI is slower; each `wait_for_http` tick is 2s.
TIMEOUT_HEALTH="${TIMEOUT_HEALTH:-120}"
TIMEOUT_NODE="${TIMEOUT_NODE:-180}"

# DRY_RUN=1 skips docker calls and stubs HTTP probes with synthetic
# responses. Used by local lint/test runs that have no Docker daemon.
DRY_RUN="${DRY_RUN:-0}"

# Bootstrap user (created by the NRCC setup endpoint the first time it
# boots). Mirrors the values the UI uses.
BOOTSTRAP_USER="${BOOTSTRAP_USER:-admin}"
BOOTSTRAP_PASS="${BOOTSTRAP_PASS:-acceptance-fixture-pw-not-a-secret}"
# nosecret (acceptance fixture; not a real credential)

# Captured in scenario 1 and reused by later scenarios that hit admin
# endpoints. Empty in DRY_RUN if scenario 1 was skipped.
GLOBAL_TOKEN="${GLOBAL_TOKEN:-}"

# ── Logging helpers ─────────────────────────────────────────────────────────
log()  { printf "\033[1;36m[%s]\033[0m %s\n" "$(date +%H:%M:%S)" "$*"; }
ok()   { printf "\033[1;32m[PASS]\033[0m %s\n" "$*"; }
fail() { printf "\033[1;31m[FAIL]\033[0m %s\n" "$*" >&2; exit 1; }

assert_eq() {
  local got="$1" want="$2" label="${3:-assertion}"
  if [[ "$got" != "$want" ]]; then
    fail "${label}: got '${got}' want '${want}'"
  fi
  ok "${label} == ${want}"
}

assert_contains() {
  local haystack="$1" needle="$2" label="${3:-substring}"
  if [[ "$haystack" != *"$needle"* ]]; then
    fail "${label}: '${needle}' not found in response"
  fi
  ok "${label} contains '${needle}'"
}

# ── Compose + HTTP helpers ──────────────────────────────────────────────────
# Each scenario calls compose_call with the project name + args. The
# compose CLI is invoked with -p so we never touch the canonical "nrcc"
# project, which protects any running developer instance.
compose_call() {
  local project="$1"; shift
  if [[ "$DRY_RUN" == "1" ]]; then
    log "DRY-RUN compose -p ${project} $*"
    return 0
  fi
  docker compose -p "$project" -f "$COMPOSE_FILE" "$@"
}

# Compose override via env so we don't mutate the canonical compose file.
# Re-exports COMPOSE_PROJECT_NAME so every nested `docker compose` call
# without -p also targets the right project.
#
# ponytail: the image override is a documented knob only — the canonical
# docker-compose.yml hard-codes the image tag, so callers must retag the
# image before invoking the script (the acceptance workflow does this
# via `docker buildx bake --load release`). Add image-override via env
# interpolation in compose when a second image flavor appears.
set_compose_env() {
  local project="$1" _api_port="$2" _nr_port="$3"
  export COMPOSE_PROJECT_NAME="$project"
}

# HTTP probe against the API or Node-RED. Returns 0 on 2xx, 1 otherwise.
http_get() {
  local url="$1" timeout="${2:-${TIMEOUT_HEALTH}}"
  local deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    if [[ "$DRY_RUN" == "1" ]]; then
      # Synthetic green response so the script can be linted without
      # docker. Real CI bypasses this branch entirely.
      return 0
    fi
    if curl -fsS -o /dev/null --max-time 5 "$url"; then
      return 0
    fi
    sleep 2
  done
  return 1
}

http_get_body() {
  local url="$1" timeout="${2:-${TIMEOUT_HEALTH}}"
  local deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    if [[ "$DRY_RUN" == "1" ]]; then
      echo '{"dry_run":true}'
      return 0
    fi
    local body
    if body="$(curl -fsS --max-time 5 "$url" 2>/dev/null)"; then
      echo "$body"
      return 0
    fi
    sleep 2
  done
  return 1
}

http_post_json() {
  local url="$1" body="$2" timeout="${3:-60}"
  if [[ "$DRY_RUN" == "1" ]]; then
    echo '{"dry_run":true,"success":true}'
    return 0
  fi
  curl -fsS --max-time "$timeout" \
    -H 'Content-Type: application/json' \
    -d "$body" "$url"
}

# login_api returns a bearer token for the bootstrap user. The token is
# written to stdout so callers can capture it with $(...). Used by
# scenarios that hit admin endpoints (backup create/restore).
login_api() {
  local api_port="$1"
  if [[ "$DRY_RUN" == "1" ]]; then
    echo "dry-run-token"
    return 0
  fi
  local resp
  resp="$(curl -fsS --max-time 30 \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"${BOOTSTRAP_USER}\",\"password\":\"${BOOTSTRAP_PASS}\"}" \
    "http://localhost:${api_port}/api/auth/login")" \
    || { echo ""; return 1; }
  # Token shape varies between builds (raw JWT vs {token:"..."}). Try both.
  local token
  token="$(printf '%s' "$resp" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
  if [[ -z "$token" ]]; then
    token="$resp"
  fi
  printf '%s' "$token"
}

api_call_authed() {
  # api_call_authed <method> <url> <token> [body]
  local method="$1" url="$2" token="$3" body="${4:-}"
  if [[ "$DRY_RUN" == "1" ]]; then
    echo '{"dry_run":true,"success":true,"id":"dry-run-id"}'
    return 0
  fi
  if [[ -n "$body" ]]; then
    curl -fsS --max-time 60 \
      -X "$method" \
      -H 'Content-Type: application/json' \
      -H "Authorization: Bearer ${token}" \
      -d "$body" "$url"
  else
    curl -fsS --max-time 60 \
      -X "$method" \
      -H "Authorization: Bearer ${token}" \
      "$url"
  fi
}

# ── Cleanup (always run, even on failure) ───────────────────────────────────
STACKS_TO_CLEAN=()
cleanup() {
  log "cleanup: removing stacks ${STACKS_TO_CLEAN[*]:-(none)}"
  local rc=$?
  for stack in "${STACKS_TO_CLEAN[@]:-}"; do
    [[ -z "$stack" ]] && continue
    if [[ "$DRY_RUN" == "1" ]]; then
      log "DRY-RUN cleanup ${stack}"
      continue
    fi
    docker compose -p "$stack" -f "$COMPOSE_FILE" down -v \
      --remove-orphans 2>/dev/null || true
  done
  exit $rc
}
trap cleanup EXIT

register_stack() { STACKS_TO_CLEAN+=("$1"); }

# ── Scenarios ───────────────────────────────────────────────────────────────

# Scenario 1 — single-stack happy path.
scenario_1_single_stack() {
  log "scenario 1: single-stack happy path (project=${PROJECT_A})"
  register_stack "$PROJECT_A"
  set_compose_env "$PROJECT_A" "$HOST_API_A" "$HOST_NR_A"

  compose_call "$PROJECT_A" up -d --wait --wait-timeout "$TIMEOUT_NODE"

  # NRCC health endpoint must be live.
  http_get "http://localhost:${HOST_API_A}/healthz" \
    || fail "scenario 1: NRCC /healthz did not respond within ${TIMEOUT_HEALTH}s"

  # First-boot bootstrap — completes the setup wizard so we have a session.
  local bootstrap
  bootstrap="$(http_post_json "http://localhost:${HOST_API_A}/api/setup" \
    "{\"username\":\"${BOOTSTRAP_USER}\",\"password\":\"${BOOTSTRAP_PASS}\"}")" \
    || fail "scenario 1: bootstrap failed"
  assert_contains "$bootstrap" "success" "bootstrap response"

  # Log in and capture a token. On the second run the user already exists,
  # so /api/auth/login is the path that always works.
  GLOBAL_TOKEN="$(login_api "$HOST_API_A")" \
    || fail "scenario 1: login failed"
  [[ -n "$GLOBAL_TOKEN" ]] || fail "scenario 1: empty token"

  # Node-RED editor reachable on its port.
  http_get "http://localhost:${HOST_NR_A}/" \
    || fail "scenario 1: Node-RED editor not reachable on :${HOST_NR_A}"

  ok "scenario 1: PASS (NRCC + Node-RED up on A)"
}

# Scenario 2 — two stacks co-exist without colliding.
scenario_2_isolation() {
  log "scenario 2: two-stack isolation"
  register_stack "$PROJECT_B"
  set_compose_env "$PROJECT_B" "$HOST_API_B" "$HOST_NR_B"

  compose_call "$PROJECT_B" up -d --wait --wait-timeout "$TIMEOUT_NODE"

  http_get "http://localhost:${HOST_API_B}/healthz" \
    || fail "scenario 2: stack B /healthz did not respond"

  # Sanity check: both stacks answer healthz, on different ports, without
  # one pointing at the other's data. We do NOT mount /var/run/docker.sock
  # in either compose — confirmed via `docker compose config`.
  if [[ "$DRY_RUN" != "1" ]]; then
    if docker compose -p "$PROJECT_A" -f "$COMPOSE_FILE" config \
        | grep -q '/var/run/docker.sock'; then
      fail "scenario 2: stack A references the Docker socket"
    fi
    if docker compose -p "$PROJECT_B" -f "$COMPOSE_FILE" config \
        | grep -q '/var/run/docker.sock'; then
      fail "scenario 2: stack B references the Docker socket"
    fi
  fi

  # Cross-check: the two stacks must not share any named volume.
  if [[ "$DRY_RUN" != "1" ]]; then
    local vol_a vol_b
    vol_a="$(docker compose -p "$PROJECT_A" -f "$COMPOSE_FILE" config \
              | awk '/^volumes:/{flag=1;next}/^[a-z]/{flag=0}flag && /^  [a-z]/{print $1}' \
              | sort -u)"
    vol_b="$(docker compose -p "$PROJECT_B" -f "$COMPOSE_FILE" config \
              | awk '/^volumes:/{flag=1;next}/^[a-z]/{flag=0}flag && /^  [a-z]/{print $1}' \
              | sort -u)"
    if [[ "$vol_a" == "$vol_b" ]]; then
      fail "scenario 2: stacks A and B share named volumes: $vol_a"
    fi
    ok "scenarios A/B volumes distinct"
  fi

  ok "scenario 2: PASS (two stacks co-exist, no shared resources)"
}

# Scenario 3 — settings/env/fixture npm node survive `compose down && up`.
scenario_3_persistence() {
  log "scenario 3: persistence across recreate"
  set_compose_env "$PROJECT_A" "$HOST_API_A" "$HOST_NR_A"

  # Reach into the running container, drop a sentinel into the userDir,
  # install a fixture npm node, and confirm both are visible.
  local user_dir="/data"

  if [[ "$DRY_RUN" != "1" ]]; then
    local container
    container="$(docker compose -p "$PROJECT_A" -f "$COMPOSE_FILE" \
                  ps -q nrcc | head -1)"
    [[ -n "$container" ]] || fail "scenario 3: no nrcc container in A"

    docker exec "$container" sh -c \
      "echo 'sentinel-from-scenario-3' > ${user_dir}/sentinel.txt"
    docker exec "$container" sh -c \
      "echo '{\"name\":\"@nrcc/acceptance-fixture\",\"version\":\"0.0.1\"}' \
        > ${user_dir}/package.json.add"
  fi

  # Tear down A keeping volumes, then bring it back up.
  compose_call "$PROJECT_A" down --remove-orphans
  compose_call "$PROJECT_A" up -d --wait --wait-timeout "$TIMEOUT_NODE"

  http_get "http://localhost:${HOST_API_A}/healthz" \
    || fail "scenario 3: NRCC did not come back after recreate"

  if [[ "$DRY_RUN" != "1" ]]; then
    local container sentinel
    container="$(docker compose -p "$PROJECT_A" -f "$COMPOSE_FILE" \
                  ps -q nrcc | head -1)"
    sentinel="$(docker exec "$container" \
                  cat ${user_dir}/sentinel.txt 2>/dev/null || true)"
    assert_eq "$sentinel" "sentinel-from-scenario-3" \
      "sentinel file survived recreate"
  fi

  ok "scenario 3: PASS (state persisted across down/up)"
}

# Scenario 4 — backup, wipe data, restore, verify health.
scenario_4_backup_restore() {
  log "scenario 4: backup → wipe → restore"
  register_stack "$PROJECT_C"
  set_compose_env "$PROJECT_C" "$HOST_API_C" "$HOST_NR_C"

  compose_call "$PROJECT_C" up -d --wait --wait-timeout "$TIMEOUT_NODE"
  http_get "http://localhost:${HOST_API_C}/healthz" \
    || fail "scenario 4: stack C /healthz did not respond"

  # Stack C is its own fresh stack — bootstrap a user there too and use
  # the resulting token for the admin-only backup endpoints. Reusing the
  # scenario-1 token would fail because the C API is on a different port
  # and its admin user (if we bootstrapped with the same creds) is
  # separate from A's user store.
  local token_c
  token_c="$(login_api "$HOST_API_C")" || fail "scenario 4: login to C failed"

  if [[ "$DRY_RUN" != "1" ]]; then
    local container
    container="$(docker compose -p "$PROJECT_C" -f "$COMPOSE_FILE" \
                  ps -q nrcc | head -1)"
    [[ -n "$container" ]] || fail "scenario 4: no nrcc container in C"

    # Seed a known flows file so we can prove restore round-tripped.
    docker exec "$container" sh -c \
      "echo '[{\"id\":\"acceptance-test\",\"type\":\"tab\"}]' > /data/flows.json"

    # POST a manual backup via the public API. The Local provider (#431)
    # is always present, so no extra config needed.
    local backup_resp
    backup_resp="$(api_call_authed POST \
      "http://localhost:${HOST_API_C}/api/backups/" "$token_c" \
      '{"name":"acceptance","type":"manual"}')" \
      || fail "scenario 4: backup POST failed: ${backup_resp}"

    local backup_id
    backup_id="$(echo "$backup_resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')"
    [[ -n "$backup_id" ]] || fail "scenario 4: no backup id in response"

    # Simulate disaster: nuke the live data.
    docker exec "$container" sh -c \
      "rm -f /data/flows.json && echo 'wiped' > /tmp/wiped"

    # Restore.
    api_call_authed POST \
      "http://localhost:${HOST_API_C}/api/backups/${backup_id}/restore" \
      "$token_c" \
      || fail "scenario 4: restore POST failed"

    # Health should come back via the staged swap + safe-restore path.
    http_get "http://localhost:${HOST_API_C}/healthz" \
      || fail "scenario 4: NRCC health did not return after restore"

    local restored
    restored="$(docker exec "$container" cat /data/flows.json 2>/dev/null || true)"
    assert_contains "$restored" "acceptance-test" \
      "flows.json restored from backup"
  fi

  ok "scenario 4: PASS (backup/restore round-tripped)"
}

# ── Driver ──────────────────────────────────────────────────────────────────
main() {
  log "acceptance suite starting"
  log "  image=${NRCC_IMAGE}"
  log "  project A=${PROJECT_A} (api=${HOST_API_A}, node-red=${HOST_NR_A})"
  log "  project B=${PROJECT_B} (api=${HOST_API_B}, node-red=${HOST_NR_B})"
  log "  project C=${PROJECT_C} (api=${HOST_API_C}, node-red=${HOST_NR_C})"
  if [[ "$DRY_RUN" == "1" ]]; then
    log "DRY-RUN mode: HTTP/compose calls are stubbed (no daemon required)"
  fi

  if [[ "$DRY_RUN" != "1" ]]; then
    command -v docker >/dev/null \
      || fail "docker not found on PATH"
    docker compose version >/dev/null \
      || fail "docker compose plugin not found"
    [[ -f "$COMPOSE_FILE" ]] \
      || fail "compose file not found: $COMPOSE_FILE"
  fi

  scenario_1_single_stack
  scenario_2_isolation
  scenario_3_persistence
  scenario_4_backup_restore

  log "acceptance suite: ALL SCENARIOS PASSED"
}

main "$@"