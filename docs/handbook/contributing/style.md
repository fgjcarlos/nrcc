# Contributor style guide

> **One work-unit per commit. English source files. No silent
> regressions.** This page is the author-facing companion to
> [`parser-renderer-contract.md`](parser-renderer-contract.md) and
> [`testing-fixtures.md`](testing-fixtures.md). Read both before
> sending a patch.

## Branch and commit hygiene

- One work-unit per commit. A work-unit is a feature slice, a
  refactor that is reviewable on its own, or a chore that is
  small enough to read in one sitting.
- Conventional Commits: `feat(<area>): …`, `fix(<area>): …`,
  `docs(<area>): …`, `refactor(<area>): …`, `test(<area>): …`,
  `chore(<area>): …`. The `<area>` is the directory or
  responsibility being changed (`docs`, `service`, `frontend`,
  `workflows`, `parser`).
- ≤400 authored lines per commit on the feature branch. PRs that
  exceed this are sliced (see
  [`../governance/ownership.md`](../governance/ownership.md) for
  the chain).
- The first commit on the feature branch must `git checkout -b`
  off `origin/main` (no commits on `main` directly).

## Language

- English source files only. Issue #768 documents the policy and
  the scanner that enforces it (`docs/language-policy.md`).
- Spanish is acceptable in `frontend/src/locales/es/` catalog
  entries (i18n deferred to issue #767).
- Variable names, identifiers, comments, log messages, audit
  events, and error strings are all English.

## Testing

- TDD where the test harness is available. For Go:
  `go test ./...`. For frontend: `cd frontend && npm test -- --run`.
  For docs sites: `node scripts/check-pages-links.mjs`.
- A change to a parser/renderer rule ships with a fixture in
  `internal/service/testdata/` (see
  [`testing-fixtures.md`](testing-fixtures.md)).
- A change to a catalog entry ships with the matching row in
  [`../configuration/setting-catalog.md`](../configuration/setting-catalog.md)
  AND the entry in
  `internal/service/nodered_compatibility.go`. The
  `TestCatalogDocSync` test enforces this.
- No test is "obviously correct". Every test cites the contract
  it enforces (a one-line comment is enough).

## Forbidden tokens

The freshness gate from #770 forbids a list of legacy and stale
beta-era copy tokens in every tracked `*.md` and `*.html` file
under `docs/`. The full enumeration is curated in
[`odd/handbook/forbidden-tokens.md`](../../../odd/handbook/forbidden-tokens.md)
so the handbook itself can stay compliant.

When you need to *describe* a forbidden token in writing (changelog,
contributor guide, or migration note), put that description outside
`docs/` — the gate does not scan the rest of the repository.

The categories covered are:

- Legacy Signal Prime palette tokens (the design system is `ds-*`).
- Stale beta-era copy phrases (the project ships a 5.x catalog; the
  era when those phrases were accurate is closed).
- Incorrect promise copy that no longer matches the actual policy
  (catalog coverage gates, security scans, etc.).

The scan runs in `pages-freshness.yml` (PR + weekly cron). When it
fails, fix the token — do not silence the gate.

## Go conventions

- One package per directory; the directory name is the package
  name.
- Test files are in the same package as the code under test
  (`package service`, not `package service_test`).
- Every exported type and function carries a doc comment. Unexported
  helpers do not need doc comments but may have them.
- Every error path uses a typed error (`*ApplyError`,
  `*RevisionConflictError`). Generic `errors.New` is reserved for
  the boundary checks in `validateSettingsPath` and the audit
  emit failures.
- Every audit-emitting function takes the `ApplyRequest` (not
  just the path) so the audit record carries the request UUID.
- Every typed error implements `Unwrap()` so callers can use
  `errors.Is` and `errors.As`.

## Frontend conventions

- React 19 function components, no class components.
- Tailwind utility classes only; no inline `style` props.
- Component props are typed explicitly. `React.FC` is forbidden
  (it adds implicit children).
- i18n strings live in `frontend/src/locales/<locale>/common.json`.
  No hard-coded user-visible strings in components.
- Skeleton + Empty + Error states are mandatory on every list
  view (slice G acceptance).

## Documentation conventions

- Each top-level handbook page starts with a one-line quote that
  answers "what is this page?". Example: `> **Source preservation
  is the contract.** …`
- Every cross-reference uses a relative link
  (`../configuration/setting-catalog.md`). Absolute URLs to
  GitHub are forbidden.
- Forbidden tokens in the docs site match the freshness gate.
- Section headers (h2) use Title Case. Subsection headers (h3+)
  use sentence case.
- Each page ends with a `## Related` section that links to the
  three most-relevant handbook pages and the roadmap umbrella.

## Pull request hygiene

- One PR per work-unit branch. If the branch grew two unrelated
  features, split it before review.
- The PR description links the issue body, lists the work-units
  closed, and shows the test command(s) the author ran.
- The PR description includes the slice/cluster from
  `docs/control-plane.md` if the work closes one.
- Labels: `documentation` or the area label closest to the change
  (`area: devops`, `area: frontend`, etc.). `type:feature` for
  new functionality, `type:chore` otherwise.

## Related

- [`parser-renderer-contract.md`](parser-renderer-contract.md) —
  the parser/renderer pair rules.
- [`testing-fixtures.md`](testing-fixtures.md) — fixture
  authoring rules.
- [`../governance/ownership.md`](../governance/ownership.md) —
  who owns what.
