#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# scripts/sync-install.sh — keep docs/install.sh byte-identical with
# scripts/install.sh so GitHub Pages (which serves docs/) keeps publishing
# the canonical installer at get.nrcc.dev/install.sh.
#
# Usage (from repo root):
#   scripts/sync-install.sh             # copies and reports diff exit
#   scripts/sync-install.sh --check     # exits 1 if drift, no write
#   scripts/sync-install.sh --write     # copies and stage the change
#
# Background:
#   scripts/install.sh is the canonical installer. It is also published
#   at https://fgjcarlos.github.io/nrcc/install.sh because GitHub Pages
#   serves the docs/ directory of main. Keeping two copies manually is
#   error-prone; this script makes the relationship explicit and CI
#   fails if they ever drift.
# ─────────────────────────────────────────────────────────────────────────────

set -euo pipefail

src="scripts/install.sh"
dst="docs/install.sh"

case "${1:-}" in
  --check)
    if cmp -s "$src" "$dst"; then
      echo "✓ $dst is in sync with $src"
      exit 0
    fi
    echo "✗ $dst differs from $src" >&2
    diff -u "$src" "$dst" >&2 || true
    exit 1
    ;;
  --write)
    if ! cmp -s "$src" "$dst"; then
      cp "$src" "$dst"
      git add "$dst"
      echo "✓ Synced $dst from $src and staged the change"
    else
      echo "✓ $dst already in sync, nothing to do"
    fi
    exit 0
    ;;
  "")
    cp "$src" "$dst"
    echo "✓ Copied $src -> $dst"
    ;;
  *)
    echo "Usage: $0 [--check|--write]" >&2
    exit 64
    ;;
esac