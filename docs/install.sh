#!/bin/sh
set -e

# nrcc One-Liner Installer
# Usage: curl https://get.nrcc.dev | sh
# Or pinned version: NRCC_VERSION=v1.0.0 curl https://get.nrcc.dev | sh

REPO="composedof2/nrcc"
VERSION="${NRCC_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Step 1: Check dependencies
echo "→ Checking dependencies…"

# Check for curl
if ! command -v curl >/dev/null 2>&1; then
	echo "Error: curl is required but not installed. Install curl and retry."
	exit 1
fi

# Check for sha256sum (Linux) or shasum (macOS)
if command -v sha256sum >/dev/null 2>&1; then
	SHA256_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	SHA256_CMD="shasum -a 256"
else
	echo "Error: sha256sum or shasum not found. Install GNU coreutils or coreutils package to enable checksum verification."
	exit 1
fi

# Step 2: Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
	x86_64) ARCH="amd64" ;;
	aarch64) ARCH="arm64" ;;
	arm64) ARCH="arm64" ;;
	armv7l) ARCH="armv7" ;;
	*)
		echo "Error: Unsupported architecture: $ARCH. Please download manually from https://github.com/composedof2/nrcc/releases"
		exit 1
		;;
esac

# Normalize OS names and check for support
case "$OS" in
	linux) OS="linux" ;;
	darwin) OS="darwin" ;;
	*)
		echo "Error: Unsupported OS: $OS. Please download manually from https://github.com/composedof2/nrcc/releases"
		exit 1
		;;
esac

echo "→ Detected OS: $OS, Architecture: $ARCH"

# Step 3: Resolve version (latest or pinned)
if [ "$VERSION" = "latest" ]; then
	echo "→ Resolving latest version…"
	VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/' | head -1)
	if [ -z "$VERSION" ]; then
		echo "Error: Could not resolve latest version from GitHub API. Set NRCC_VERSION explicitly, e.g.: NRCC_VERSION=v1.0.0 curl -fsSL https://get.nrcc.dev/install.sh | sh"
		exit 1
	fi
fi

echo "Installing nrcc ${VERSION}…"

# Step 3a: Construct binary and checksum URLs
BINARY="nrcc-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
	BINARY="${BINARY}.exe"
fi

BINARY_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}.sha256"

# Step 3b: Validate install directory
echo "→ Installing to ${INSTALL_DIR}/nrcc"
if [ ! -d "$INSTALL_DIR" ]; then
	echo "Error: Install directory $INSTALL_DIR does not exist. Create it first or choose another directory."
	exit 1
fi

# Step 4: Download to temp directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "→ Downloading binary…"
curl -fsSL -o "$TMP_DIR/nrcc" "$BINARY_URL" || {
	echo "Error: Failed to download ${BINARY} from ${BINARY_URL}. Check your network connection and retry."
	exit 1
}

echo "→ Downloading checksum…"
curl -fsSL -o "$TMP_DIR/nrcc.sha256" "$CHECKSUM_URL" || {
	echo "Warning: SHA256SUMS not found for ${VERSION}. Skipping checksum verification. To verify manually, download from https://github.com/composedof2/nrcc/releases"
	# Continue with warning instead of failing - checksum files may not exist for legacy releases
}

# Step 5: Verify checksum (if file exists)
echo "→ Verifying checksum…"
cd "$TMP_DIR"
if [ -f "nrcc.sha256" ]; then
	if ! $SHA256_CMD -c nrcc.sha256 >/dev/null 2>&1; then
		echo "Error: Checksum verification failed for ${BINARY}. The downloaded file may be corrupt. Please retry or download manually from https://github.com/composedof2/nrcc/releases"
		exit 1
	fi
else
	if [ "${NRCC_STRICT_CHECKSUMS:-0}" = "1" ]; then
		echo "Error: SHA256SUMS not found for ${VERSION} and NRCC_STRICT_CHECKSUMS=1. Aborting."
		exit 1
	fi
fi

# Step 6: Make executable
chmod 0755 "$TMP_DIR/nrcc"

# Step 7: Install binary
echo "→ Installing to $INSTALL_DIR/nrcc…"
if [ "$(id -u)" -eq 0 ]; then
	# Running as root
	install -m 755 "$TMP_DIR/nrcc" "$INSTALL_DIR/nrcc"
else
	# Not root, use sudo
	if ! command -v sudo >/dev/null 2>&1; then
		echo "Error: Running as non-root user but 'sudo' is not available"
		exit 1
	fi
	sudo install -m 755 "$TMP_DIR/nrcc" "$INSTALL_DIR/nrcc"
fi

# Step 8: Print success and next steps
echo ""
echo "✓ nrcc ${VERSION} installed to ${INSTALL_DIR}/nrcc"
echo ""
echo "Next step — set up as a system service:"
echo "  sudo nrcc install"
echo ""
echo "Then open http://localhost:3001 in your browser."
