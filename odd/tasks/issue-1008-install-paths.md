# Issue 1008 — make the documented install paths work
Branch: `fix/1008-install-paths`; base: `origin/main` (`76f3712`); delivery: PR after verification; forecast: ~300 authored lines.
Checks: `go test ./internal/installer/ ./internal/service/`, `shellcheck -s sh docs/install.sh`, `docker compose config`, `node --test docs/sections/index.test.mjs`, langscan. No tag, no release: running the commands against the release candidate is the maintainer's step after `v1.0.0-rc.1` is published.

## Specs
S1. "Add `image: ghcr.io/fgjcarlos/nrcc:${NRCC_VERSION:-latest}` beside `build:` in the compose file, so `pull` works and `--build` stays available for contributors." Decision (L2): the variable is `NRCC_IMAGE_TAG`, because published image tags carry no `v` prefix (`scripts/release/metadata.py:182`) while installer versions do.
S2. "Make the installer pull the released image instead of building." Includes: the installer no longer "falls back to `main` when no release exists" and no longer runs `docker compose up -d --build`.
S3. Decision (L2): the installer always pins the exact installed version as `NRCC_IMAGE_TAG` in `.env`; re-running the installer upgrades a pinned stack.
S4. "Remove `get.nrcc.dev` from the README until DNS and Pages are live, or bring them up. Point to the raw GitHub URL in the meantime."
S5. "Then drop `continue-on-error` from the `smoke-install-url` job."
S6. Acceptance: "On a machine with Docker and no Go or Node toolchain, the README quick start reaches a healthy stack." / "`docker compose pull && docker compose up -d` upgrades an existing stack." / "Every install command in the README and in `docs/` has been run once against the release candidate."

## Tasks
- T1 [S1,S2,S3] compose `image:` + installer pulls and pins the released image, fails without a resolvable release; installer test; env-contract doc. Route: inline, test-first. Risk: item 5. Commit: pending.
- T2 [S4,S6] docs: README, quick-start, docker-stack upgrade, production guide, CONTRIBUTING, openspec project map, installer header use the raw GitHub URL and the pull/upgrade flow. Route: inline. Commit: pending.
- T3 [S5,S6] release workflow: `smoke-install-url` runs the raw-URL installer end to end against the published tag as a hard gate. Route: inline. Risk: item 5. Commit: pending.
- T4 [S1-S6] independent verification (`gentle-ai-verify`, Risk: item 5). Pending.

## Log
L1. User: "Resuelve la issue #1008"
L2. User chose: "Fijar siempre la versión exacta en .env (recomendado)". Variable named `NRCC_IMAGE_TAG` (no `v` prefix) to match published image tags.
