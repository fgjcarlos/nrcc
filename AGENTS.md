# Agent instructions

This file is the canonical entry point for AI coding agents (and
human contributors) working in this repository. It defines the
repository's language, testing, and review conventions.

## Language

All repository-maintained technical artifacts are written in English.
See CONTRIBUTING.md for the full rule, the scanner that enforces it,
and the exemption categories.

Concretely:

- English: source comments, log messages, dev-facing errors, scripts,
  CI workflow comments, configuration examples, API descriptions,
  documentation, issue/PR templates, contributor files.
- Localizable (deferred to i18n catalogs in issue #767): user-visible
  product copy. Until the catalog exists, user-facing strings live in
  frontend/src/** and may carry the existing Spanish copy.

When in doubt, write in English.

## Project conventions

- Frontend: React + TypeScript + Vite + Vitest + Playwright.
  Run `cd frontend && npm test -- --run` for unit tests and
  `cd frontend && npm run test:e2e` for e2e tests.
- Backend: Go. Run `go test ./...` for unit tests.
- Generated artifacts: never edit generated files (frontend/dist,
  vendor, mocks, etc.) by hand. Regenerate them.

## Plan / track before writing

For substantial work (multi-step implementation, multi-file changes,
or anything that needs recoverable state), follow the Organic Driven
Development (ODD) workflow. Track tasks under
`odd/tasks/<feature-name>.md` before the first write and mirror them in
a visible todo list.

## Commits

- Use Conventional Commit messages.
- One work-unit per commit (feature slice or chore).
- Keep work-unit commits under the review budget agreed for the slice
  (typically 400 authored lines).

## Review

- Do not claim a fix is verified without observed evidence (test
  output, lint run, build result).
- Surface failed, skipped, or pending checks explicitly in the final
  report.
