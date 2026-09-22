# Issue 768 — Enforce English across source and repository artifacts

## Objective

Establish English as the canonical language for repository-maintained
technical artifacts (source comments, dev messages, scripts, configs,
docs, CI workflows) and prevent regressions through an advisory CI gate.
User-visible localized strings stay in place — they migrate to i18n
catalogs under issue #767.

## Scope (per user decision 2026-09-22)

- Migrate non-English artefacts in **non-frontend files**:
  `*.go`, `*.ts`/`*.tsx` (NON-UI strings), `*.md`, `*.yml`,
  scripts, `.github/`, `Dockerfile`, `LICENSE`, etc.
- **Do NOT** migrate user-visible Spanish UI strings — those belong
  in the i18n catalog introduced by #767.
- **Do NOT** migrate locale files (when they exist).
- Add an **advisory** CI gate that warns on new violations but does
  not block PRs. Migration is gradual.

## Detector rules (slice 1)

The scanner distinguishes legitimate technical Unicode from
violations:

- **Whitelisted Unicode** (NOT violations): math symbols (`±`, `×`,
  `§`), Greek (`α`, `β`), arrows, box-drawing, etc.
- **Spanish content** (violations when in non-frontend files):
  Spanish sentence patterns (multiple words + Spanish-specific
  characters like `á`, `é`, `í`, `ó`, `ú`, `ñ`, `¿`, `¡`).
- **Excluded paths**: `frontend/src/**/locales/**`,
  `frontend/src/i18n/**`, generated files.

## Slice plan (feature-branch-chain)

### Slice 1 — Policy + audit tool
- Branch: `chore/issue-768-policy-and-audit`
- `CONTRIBUTING.md` updated with the English-first policy section.
- `AGENTS.md` created (does not exist in repo) with the language rule
  for AI assistants.
- `tools/langscan/main.go` — Go scanner that walks the repo and emits
  a JSON + markdown report of suspected non-English artefacts in
  non-frontend files. Whitelist logic for technical Unicode.
- `tools/langscan/main_test.go` — table-driven tests with fixtures
  (positive cases: real Spanish in comments; negative cases: Spanish
  UI strings, technical Unicode, locale files, generated files).
- `odd/tasks/issue-768-audit.md` — initial findings from the
  scanner, grouped by directory.
- Acceptance: scanner runs `go run ./tools/langscan` and the report
  matches the manual audit. All tests pass.

### Slice 2 — Migration
- Branch: `chore/issue-768-migrate-source-and-docs`
- Migrate the high-confidence findings from slice 1:
  - Go source comments and log messages → English.
  - TS source comments (NON-UI) → English.
  - `.github/` workflow comments, error messages → English.
  - CONTRIBUTING, LICENSE (already mostly English), docs fragments
    that drift → English.
- **Do NOT touch** `frontend/src/**` UI strings (out of scope per
  #767). The scanner is updated to mark these explicitly as
  "deferred to #767".
- Commit per area (Go, .github, docs) to keep review slices small.
- Acceptance: scanner report shrinks to only the deferred-to-#767
  findings. No new violations.

### Slice 3 — CI gate + contributing guide
- Branch: `chore/issue-768-ci-gate`
- `.github/workflows/language-policy.yml` — runs the scanner on PRs
  and on push to main. Advisory: posts a comment with the report but
  does not block.
- `CONTRIBUTING.md` already updated by slice 1; this slice cross-
  references the workflow and adds the "what to do when the
  advisory fires" section.
- Tests for the workflow: run the scanner in a fixture repo with
  both clean and dirty states; verify advisory behaviour.
- Acceptance: PR with a new Spanish comment shows the advisory
  comment. Existing violations do not block.

### Tracker
- `chore/issue-768-english-enforcement` (from main) merges the three
  slices in order. Single PR to main.

## Out of scope (explicit)

- User-visible UI strings in `frontend/src/**` — handled by #767.
- Locale catalogs (when added) — handled by #767.
- Generated files (vendor, mocks) — always exempt.
- Commit messages — left to author preference for now (no policy).

## Stop conditions

- If slice 1 finds >500 suspected findings, narrow the detector
  rules before proceeding (avoid noisy reports).
- If slice 2's per-area commits exceed 400 lines, split per file.
- If the CI workflow breaks an unrelated check, refactor before merge.
