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

# Re-installing over an existing nrcc install keeps the operator's .env (and
# docker-compose.override.yml) and replaces the checkout, so the new revision
# is never merged into the old one. The checkout is never deleted: a failed
# reinstall rolls back from the .previous.$$ copy, and a verified one is kept
# at INSTALL_DIR.previous for the operator to delete (a preserved compose
# override may bind-mount data beneath it).
if [ -d "$INSTALL_DIR" ]; then
	if [ ! -f "$INSTALL_DIR/.env" ] || [ ! -f "$INSTALL_DIR/docker-compose.yml" ]; then
		echo "Error: $INSTALL_DIR already exists and is not an nrcc install. Set INSTALL_DIR to a new path." >&2
		exit 1
	fi
	if [ ! -f "$INSTALL_DIR/.nrcc-install" ] && [ "${NRCC_REPLACE_EXISTING:-0}" != "1" ]; then
		echo "Error: $INSTALL_DIR has no .nrcc-install marker, so it was not created by this installer." >&2
		echo "Refusing to replace it. Re-run with NRCC_REPLACE_EXISTING=1 to replace it anyway (the replaced checkout is kept at ${INSTALL_DIR}.previous for rollback)." >&2
		exit 1
	fi
	CLONE_DIR="${INSTALL_DIR}.clone.$$"
	PREVIOUS_DIR="${INSTALL_DIR}.previous.$$"
	echo "→ Cloning $REPO @ $VERSION into $INSTALL_DIR (replacing the previous checkout)"
	git clone --depth 1 --branch "$VERSION" "https://github.com/$REPO.git" "$CLONE_DIR"
	mv "$INSTALL_DIR" "$PREVIOUS_DIR"
	mv "$CLONE_DIR" "$INSTALL_DIR"
	cp -p "$PREVIOUS_DIR/.env" "$INSTALL_DIR/.env"
	if [ -f "$PREVIOUS_DIR/docker-compose.override.yml" ]; then
		cp -p "$PREVIOUS_DIR/docker-compose.override.yml" "$INSTALL_DIR/docker-compose.override.yml"
	fi
	printf 'nrcc install\n' >"$INSTALL_DIR/.nrcc-install"
else
	echo "→ Cloning $REPO @ $VERSION into $INSTALL_DIR"
	mkdir -p "$(dirname "$INSTALL_DIR")"
	git clone --depth 1 --branch "$VERSION" "https://github.com/$REPO.git" "$INSTALL_DIR"
	printf 'nrcc install\n' >"$INSTALL_DIR/.nrcc-install"
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
		# The key must land on its own line: when the file does not end with a
		# newline, append one first or the assignment merges into the previous
		# value and Compose interpolation breaks.
		if [ -s "$env_file" ] && [ -n "$(tail -c 1 "$env_file")" ]; then
			printf '\n' >>"$env_file"
		fi
		printf 'NRCC_ENCRYPTION_KEY=%s\n' "$key" >>"$env_file"
	fi
	chmod 600 "$env_file"
	unset key
}

if [ -z "${NRCC_ENCRYPTION_KEY:-}" ]; then
	ensure_encryption_key .env
fi

# A failed reinstall keeps the previous checkout on disk until the replacement
# is verified, so this hint can always recover the working install.
restore_previous() {
	if [ -n "${PREVIOUS_DIR:-}" ] && [ -d "$PREVIOUS_DIR" ]; then
		echo "The previous checkout was kept at $PREVIOUS_DIR." >&2
		echo "To roll back: rm -rf '$INSTALL_DIR' && mv '$PREVIOUS_DIR' '$INSTALL_DIR'" >&2
	fi
}

echo "→ Building and starting stack"
if ! docker compose up -d --build; then
	restore_previous
	docker compose ps || true
	docker compose logs --tail=50 || true
	exit 1
fi

# Report success only once the stack is ready: every container's health first,
# the published health endpoint as fallback when no healthcheck is exposed.
READY_TIMEOUT="${NRCC_READY_TIMEOUT:-300}"
case "$READY_TIMEOUT" in
	''|*[!0-9]*)
		echo "Error: NRCC_READY_TIMEOUT must be a number of seconds, got '$READY_TIMEOUT'." >&2
		exit 1
		;;
esac

# Aggregate health over every container Compose reports: a stopped container
# fails immediately (an exited service must not look ready just because it is
# invisible to a running-only listing), then unhealthy wins, then all healthy,
# then anything still starting, and nohealth when no container exposes a
# health status at all.
compose_health() {
	saw_status=0
	all_healthy=1
	# Container ids are word-split deliberately. --all includes stopped
	# containers, which a plain `ps -q` listing would hide.
	# shellcheck disable=SC2046
	for id in $(docker compose ps -q --all 2>/dev/null); do
		state=$(docker inspect --format '{{.State.Status}}' "$id" 2>/dev/null || true)
		if [ "$state" != running ]; then
			echo unhealthy
			return 0
		fi
		status=$(docker inspect --format '{{.State.Health.Status}}' "$id" 2>/dev/null || true)
		case "$status" in
			'')
				continue
				;;
			unhealthy)
				echo unhealthy
				return 0
				;;
			healthy)
				saw_status=1
				;;
			*)
				saw_status=1
				all_healthy=0
				;;
		esac
	done
	if [ "$saw_status" -eq 0 ]; then
		echo nohealth
	elif [ "$all_healthy" -eq 1 ]; then
		echo healthy
	else
		echo pending
	fi
}

ELAPSED=0
READY=0
while [ "$ELAPSED" -le "$READY_TIMEOUT" ]; do
	STATUS=$(compose_health)
	case "$STATUS" in
		healthy)
			READY=1
			break
			;;
		unhealthy)
			break
			;;
	esac
	if [ "$STATUS" = nohealth ] && command -v curl >/dev/null 2>&1 &&
		curl --connect-timeout 5 --max-time 10 -fsS "http://127.0.0.1:${NRCC_HOST_PORT:-3001}/healthz" >/dev/null 2>&1; then
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
	restore_previous
	echo "Error: nrcc did not become ready within ${READY_TIMEOUT} seconds." >&2
	docker compose ps || true
	docker compose logs --tail=50 || true
	exit 1
fi

# The replacement is verified. The previous checkout is not deleted: readiness
# alone cannot prove the operator no longer needs it (a preserved compose
# override may bind-mount a data directory beneath the old checkout that the
# new checkout lacks). It moves under a stable name, replacing only older
# .previous copies created by this installer.
if [ -n "${PREVIOUS_DIR:-}" ] && [ -d "$PREVIOUS_DIR" ]; then
	rm -rf "${INSTALL_DIR}.previous"
	mv "$PREVIOUS_DIR" "${INSTALL_DIR}.previous"
	echo "Previous checkout kept at ${INSTALL_DIR}.previous; delete it once the upgrade is confirmed."
fi

echo ""
echo "✓ nrcc stack is up."
echo "  nrcc UI:   http://localhost:${NRCC_HOST_PORT:-3001}"
echo "  Node-RED:  http://localhost:1880"
echo ""
echo "Manage the stack from $INSTALL_DIR with 'docker compose [up|down|logs|ps]'."
