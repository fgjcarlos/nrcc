#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$ROOT_DIR/dist/release}"
VERSION="${VERSION:-dev}"
APP_NAME="${APP_NAME:-nrcc}"
TARGETS="${TARGETS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"
FRONTEND_INSTALL_MODE="${FRONTEND_INSTALL_MODE:-auto}"
shopt -s nullglob

build_frontend() {
	case "$FRONTEND_INSTALL_MODE" in
		auto)
			if [[ ! -d "$ROOT_DIR/frontend/node_modules" ]]; then
				npm ci
			fi
			;;
		always)
			npm ci
			;;
		never)
			;;
		*)
			echo "Unsupported FRONTEND_INSTALL_MODE: $FRONTEND_INSTALL_MODE" >&2
			exit 1
			;;
	esac

	npm run build
}

package_target() {
	local goos="$1"
	local goarch="$2"
	local archive_base="${APP_NAME}_${VERSION}_${goos}_${goarch}"
	local stage_dir="$OUTPUT_DIR/$archive_base"
	local binary_name="$APP_NAME"

	if [[ "$goos" == "windows" ]]; then
		binary_name+=".exe"
	fi

	mkdir -p "$stage_dir"
	GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$stage_dir/$binary_name" .
	cp "$ROOT_DIR/README.md" "$stage_dir/README.md"

	if [[ "$goos" == "windows" ]]; then
		(
			cd "$OUTPUT_DIR"
			zip -qr "${archive_base}.zip" "$archive_base"
		)
	else
		(
			cd "$OUTPUT_DIR"
			tar -czf "${archive_base}.tar.gz" "$archive_base"
		)
	fi

	rm -rf "$stage_dir"
}

rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

(
	cd "$ROOT_DIR/frontend"
	build_frontend
)

for target in $TARGETS; do
	IFS=/ read -r goos goarch <<<"$target"
	package_target "$goos" "$goarch"
done

(
	cd "$OUTPUT_DIR"
	artifacts=(./*.tar.gz ./*.zip)
	if [[ ${#artifacts[@]} -eq 0 ]]; then
		echo "No packaged artifacts were created" >&2
		exit 1
	fi
	sha256sum "${artifacts[@]}" > checksums.txt
)

echo "Release artifacts written to $OUTPUT_DIR"
