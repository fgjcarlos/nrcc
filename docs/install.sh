#!/bin/sh
# nrcc docker-first installer (ADR 0003).
# Usage: curl -fsSL https://get.nrcc.dev/install.sh | sh
# Explicit version: curl -fsSL https://get.nrcc.dev/install.sh | sh -s -- v1.2.3
#
# Portable POSIX sh: no `set -o pipefail`, no bashisms. The script needs
# git, a reachable Docker daemon and (optionally) curl.

set -eu

REPO="${NRCC_REPO:-fgjcarlos/nrcc}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.nrcc}"
VERSION="${1:-${NRCC_VERSION:-}}"

for cmd in docker git; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		echo "Error: '$cmd' is required. Install it and retry." >&2
		exit 1
	fi
done
if ! docker info >/dev/null 2>&1; then
	echo "Error: docker daemon is not reachable. Start Docker and retry." >&2
	exit 1
fi

# Version selection: positional argument, then NRCC_VERSION, then the latest
# published release. `main` is the last resort and is always announced.
if [ -z "$VERSION" ]; then
	if command -v curl >/dev/null 2>&1; then
		VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null |
			sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1 || true)
	fi
	if [ -z "$VERSION" ]; then
		VERSION=main
		echo "Warning: could not resolve the latest release for $REPO; falling back to main." >&2
	fi
fi

# Re-installing over an existing nrcc install keeps the operator's .env and
# replaces the checkout, so the new revision is never merged into the old one.
if [ -d "$INSTALL_DIR" ]; then
	if [ ! -f "$INSTALL_DIR/.env" ] || [ ! -f "$INSTALL_DIR/docker-compose.yml" ]; then
		echo "Error: $INSTALL_DIR already exists and is not an nrcc install. Set INSTALL_DIR to a new path." >&2
		exit 1
	fi
	CLONE_DIR="${INSTALL_DIR}.clone.$$"
	PREVIOUS_DIR="${INSTALL_DIR}.previous.$$"
	echo "→ Cloning $REPO @ $VERSION into $INSTALL_DIR (replacing the previous checkout)"
	git clone --depth 1 --branch "$VERSION" "https://github.com/$REPO.git" "$CLONE_DIR"
	mv "$INSTALL_DIR" "$PREVIOUS_DIR"
	mv "$CLONE_DIR" "$INSTALL_DIR"
	cp -p "$PREVIOUS_DIR/.env" "$INSTALL_DIR/.env"
	rm -rf "$PREVIOUS_DIR"
else
	echo "→ Cloning $REPO @ $VERSION into $INSTALL_DIR"
	mkdir -p "$(dirname "$INSTALL_DIR")"
	git clone --depth 1 --branch "$VERSION" "https://github.com/$REPO.git" "$INSTALL_DIR"
fi

cd "$INSTALL_DIR"

# Returns a 32-byte random key as 64 hexadecimal characters.
generate_key() {
	if command -v openssl >/dev/null 2>&1; then
		openssl rand -hex 32
	else
		od -An -N32 -tx1 /dev/urandom | tr -d ' \n'
	fi
}

# Encryption material must exist before Compose interpolates it. An exported
# NRCC_ENCRYPTION_KEY wins; an operator key in .env is preserved verbatim; an
# empty key is replaced; otherwise a fresh key is generated once and stored
# with restrictive permissions. The value is never printed.
ensure_encryption_key() {
	env_file="$1"
	if [ -f "$env_file" ] &&
		grep -Eq '^[[:space:]]*(export[[:space:]]+)?NRCC_ENCRYPTION_KEY=.+$' "$env_file"; then
		return 0
	fi
	key=$(generate_key)
	case "$key" in
		*[!0-9a-f]*|'')
			echo "Error: could not generate an encryption key." >&2
			exit 1
			;;
	esac
	if [ "${#key}" -ne 64 ]; then
		echo "Error: generated encryption key has an invalid length." >&2
		exit 1
	fi
	umask 077
	if [ -f "$env_file" ] &&
		grep -Eq '^[[:space:]]*(export[[:space:]]+)?NRCC_ENCRYPTION_KEY=[[:space:]]*$' "$env_file"; then
		env_tmp="$env_file.tmp.$$"
		awk -v k="$key" '
			/^[[:space:]]*(export[[:space:]]+)?NRCC_ENCRYPTION_KEY=[[:space:]]*$/ && !seen {
				print "NRCC_ENCRYPTION_KEY=" k
				seen = 1
				next
			}
			{ print }
		' "$env_file" >"$env_tmp"
		mv "$env_tmp" "$env_file"
	else
		touch "$env_file"
		printf 'NRCC_ENCRYPTION_KEY=%s\n' "$key" >>"$env_file"
	fi
	chmod 600 "$env_file"
	unset key
}

if [ -z "${NRCC_ENCRYPTION_KEY:-}" ]; then
	ensure_encryption_key .env
fi

echo "→ Building and starting stack"
if ! docker compose up -d --build; then
	docker compose ps || true
	docker compose logs --tail=50 || true
	exit 1
fi

# Report success only once the stack is ready: container health first, the
# published health endpoint as fallback when no healthcheck is exposed.
READY_TIMEOUT="${NRCC_READY_TIMEOUT:-300}"
ELAPSED=0
READY=0
while [ "$ELAPSED" -le "$READY_TIMEOUT" ]; do
	CONTAINER_ID=$(docker compose ps -q 2>/dev/null | head -n 1)
	STATUS=""
	if [ -n "$CONTAINER_ID" ]; then
		STATUS=$(docker inspect --format '{{.State.Health.Status}}' "$CONTAINER_ID" 2>/dev/null || true)
	fi
	case "$STATUS" in
		healthy)
			READY=1
			break
			;;
		unhealthy)
			break
			;;
	esac
	if [ -z "$STATUS" ] && command -v curl >/dev/null 2>&1 &&
		curl -fsS "http://127.0.0.1:${NRCC_HOST_PORT:-3001}/healthz" >/dev/null 2>&1; then
		READY=1
		break
	fi
	if [ "$ELAPSED" -ge "$READY_TIMEOUT" ]; then
		break
	fi
	sleep 1
	ELAPSED=$((ELAPSED + 1))
done
if [ "$READY" -ne 1 ]; then
	echo "Error: nrcc did not become ready within ${READY_TIMEOUT} seconds." >&2
	docker compose ps || true
	docker compose logs --tail=50 || true
	exit 1
fi

echo ""
echo "✓ nrcc stack is up."
echo "  nrcc UI:   http://localhost:${NRCC_HOST_PORT:-3001}"
echo "  Node-RED:  http://localhost:1880"
echo ""
echo "Manage the stack from $INSTALL_DIR with 'docker compose [up|down|logs|ps]'."
