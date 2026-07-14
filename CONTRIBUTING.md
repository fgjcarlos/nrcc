# Contributing to nrcc

Thanks for your interest. This document covers how to get set up locally, what the workflow looks like, and how to send a change.

## Local setup

Requirements:

- Go 1.25+
- Node.js 22+ and pnpm 11+ (npm is intentionally not supported — see [`pnpm-workspace.yaml`](pnpm-workspace.yaml))
- Node-RED in `$PATH` (`npm install -g node-red`)

Get the code and install frontend dependencies:

```bash
git clone https://github.com/fgjcarlos/nrcc.git
cd nrcc
pnpm install   # resolves the frontend workspace
```

## Running locally

Two main flows:

```bash
make dev            # Go backend on :3001 + Vite HMR on :5173
make build          # Build the single binary with the embedded frontend
./nrcc              # Run the built binary
```

The Vite dev server proxies API calls to the Go backend, so you can iterate on either side independently.

## Running tests

```bash
make test                                 # Go test suite
pnpm --filter frontend test --run         # Vitest run (no watch)
pnpm --filter frontend lint               # ESLint
pnpm --filter frontend typecheck          # tsc --noEmit
```

CI runs the same commands on every PR — see [`.github/workflows/pr.yml`](.github/workflows/pr.yml). A PR that's red locally will be red in CI.

## Workflow

We use trunk-based development against `main` with short-lived branches and squash merges.

1. **Open an issue first** for anything that's not trivial. It's a chance to align on scope before code lands.
2. **Branch from `main`** with a descriptive prefix: `fix/`, `feat/`, `chore/`, `docs/`, `refactor/`, `test/`, `ci/`. Example: `fix/backups-listpaginated-query`.
3. **Write conventional commits** — `type(scope): subject` (e.g. `fix(auth): rotate session on password change`). The PR title follows the same format.
4. **Keep PRs focused.** One concern per PR. If a change starts spreading, split it (see [chained PR pattern](https://github.com/fgjcarlos/nrcc/issues?q=is%3Aissue+label%3A%22type%3Achore%22)).
5. **Reference issues** in the PR body (`Refs #N` for related, `Closes #N` for the one you're resolving).
6. **Let CI run.** Backend and frontend gates must be green before review.

## What to include in a PR description

- **Summary**: 1–3 bullets explaining what changed.
- **Why**: the motivation. If it's a bug, link to the issue and explain the root cause.
- **Test plan**: how to verify the change works. For UI changes, mention which page/flow to exercise manually.
- **Out of scope**: anything you noticed but deliberately didn't fix, with a link to the follow-up issue if you opened one.

## Reporting bugs

Open an issue with:

- What you expected to happen
- What actually happened
- Steps to reproduce
- Versions: `nrcc --version`, Node-RED version, OS

If the bug involves backups, configuration, or anything that touches stored state, attach the relevant file from `data/` (with secrets redacted) when possible.

## Code style

- **Go**: idiomatic Go, `gofmt`, `go vet`. The CI lint gate is `golangci-lint run --config=.golangci.yml ./...` and it is repo-wide (not a ratchet) — a green CI requires zero findings. Run it locally before pushing: `golangci-lint run --config=.golangci.yml ./...`. Tests follow table-driven patterns where useful.
- **TypeScript / React**: feature-folders under `frontend/src/features/`, shared primitives under `frontend/src/shared/`. Avoid `any` — if you reach for it, that's a signal to reshape the type. Tests use Vitest + Testing Library; mocks for `useAuth` go through the helpers in `features/auth/__test-utils__/`.
- **Installer script**: `scripts/install.sh` is the canonical copy. `docs/install.sh` is a byte-identical copy served by GitHub Pages at `get.nrcc.dev/install.sh`. When you change the installer, run `scripts/sync-install.sh --write` to refresh the copy and commit the result. CI runs `scripts/sync-install.sh --check` and fails the PR on drift.

## Security

If you find a security issue, please **don't open a public issue**. Email the maintainer (contact in the [GitHub profile](https://github.com/fgjcarlos)) or open a private security advisory in this repo.

## License

By contributing, you agree your contributions will be licensed under the [Apache License 2.0](LICENSE).
