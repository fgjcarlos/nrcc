#!/bin/sh
# dev-entrypoint.sh — installs Go + air if not present, then starts air
set -e

GO_VERSION="${GO_VERSION:-1.22.12}"
GO_ARCH="amd64"
GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
GO_URL="https://go.dev/dl/${GO_TAR}"
GO_INSTALL_DIR="/usr/local"
AIR_BIN="/usr/local/bin/air"

# ── Install Go if needed ──────────────────────────────────────────────────────
if ! command -v go >/dev/null 2>&1; then
    echo "→ Installing Go ${GO_VERSION}..."
    apt-get update -qq && apt-get install -y --no-install-recommends wget ca-certificates 2>/dev/null
    wget -qO "/tmp/${GO_TAR}" "${GO_URL}"
    tar -C "${GO_INSTALL_DIR}" -xzf "/tmp/${GO_TAR}"
    rm "/tmp/${GO_TAR}"
    export PATH="${GO_INSTALL_DIR}/go/bin:${PATH}"
    echo "✓ Go $(go version) installed"
else
    export PATH="${GO_INSTALL_DIR}/go/bin:${PATH}"
    echo "✓ Go $(go version) already installed"
fi

export GOPATH="/root/go"
export PATH="${GOPATH}/bin:${PATH}"

# ── Install air if needed ─────────────────────────────────────────────────────
if ! command -v air >/dev/null 2>&1 && [ ! -f "${AIR_BIN}" ]; then
    echo "→ Installing air (Go hot reload)..."
    go install github.com/air-verse/air@latest
    # Link to /usr/local/bin for easy access
    ln -sf "${GOPATH}/bin/air" "${AIR_BIN}" 2>/dev/null || true
    echo "✓ air installed"
else
    echo "✓ air already installed"
fi

# ── Download Go modules if needed ────────────────────────────────────────────
echo "→ Syncing Go modules..."
go mod download

# ── Start air (hot reload) ────────────────────────────────────────────────────
echo "→ Starting nrcc with hot reload (air)..."
exec air -c .air.toml
