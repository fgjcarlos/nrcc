#!/bin/sh
# nrcc docker-first installer (ADR 0003).
# Usage: curl -fsSL https://get.nrcc.dev/install.sh | sh
# Optional: NRCC_VERSION=<git ref> curl -fsSL … | sh

set -euo pipefail

REPO="${NRCC_REPO:-fgjcarlos/nrcc}"
VERSION="${NRCC_VERSION:-main}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.nrcc}"

for cmd in docker git; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		echo "Error: '$cmd' is required. Install it and retry."
		exit 1
	fi
done

if ! docker info >/dev/null 2>&1; then
	echo "Error: docker daemon is not reachable. Start Docker and retry."
	exit 1
fi

echo "→ Cloning $REPO @ $VERSION into $INSTALL_DIR"
mkdir -p "$(dirname "$INSTALL_DIR")"
if [ -d "$INSTALL_DIR" ]; then
	echo "Error: $INSTALL_DIR already exists. Remove it or set INSTALL_DIR to a new path."
	exit 1
fi
git clone --depth 1 --branch "$VERSION" "https://github.com/$REPO.git" "$INSTALL_DIR"

cd "$INSTALL_DIR"

echo "→ Pulling base image (nodered/node-red:4.1) and starting stack"
docker compose pull
docker compose up -d

echo ""
echo "✓ nrcc stack is up."
echo "  nrcc UI:   http://localhost:3001"
echo "  Node-RED:  http://localhost:1880"
echo ""
echo "Manage the stack from $INSTALL_DIR with 'docker compose [up|down|logs|ps]'."