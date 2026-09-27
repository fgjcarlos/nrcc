# Authentication Surfaces Closure — Design

Slice 1 closes the Go half of Gap G2 for cluster #760 with a table-driven parser contract test and, only if that test exposes a defect, the smallest normalization change needed to satisfy it. It does not change authentication policy. Slice 2, `SecuritySurfaceIsolationE2E`, remains deferred.

## Decision summary

| Concern | Decision |
|---|---|
| Migration boundary | Normalize legacy auth aliases in the parsed `NodeRedConfig` before `Settings.ts` is re-rendered. Use a new `NormalizeLegacyAuthFields(in *NodeRedConfig) error` helper only if the current parser does not already produce the canonical model. |
| Rendering | Keep canonical rendering canonical. Do not make `RenderHTTPAuthCanonicalKeys` a return-only migration: it cannot correct the structured config used by validation, apply, and subsequent parsing. |
| Unknown legacy auth fields | Drop them from managed auth projection; never preserve their values in plaintext, diagnostics, logs, or round-trip output. |
| Slice | Slice 1 is the Go test and minimum parser correction. The Playwright E2E is deferred to slice 2. |
| Delivery gate | Re-`ask-on-risk` at slice-1 close before promoting slice 2; preserve the 400-authored-line budget. |

## Legacy-to-canonical migration map

| Legacy NRCC settings field / shape | Canonical Node-RED 5.x surface | Required handling |
|---|---|---|
| `adminAuth` (legacy single-user shape) | `adminAuth` | Normalize into `AdminAuth.Users` with one entry. Preserve username, bcrypt hash bytes, and permissions. |
| `adminAuth` (legacy multi-user shape) | `adminAuth` | Preserve user order, count, each bcrypt hash, and each `permissions` value. |
| `httpAuth` | `httpNodeAuth` | Populate `HTTPNodeAuth`; do not populate `HTTPStaticAuth`. |
| Static-auth alias (`nodeHttpAuth` or `staticAuth`) | `httpStaticAuth` | Populate `HTTPStaticAuth`; do not populate `HTTPNodeAuth`. |
| Unrecognised legacy authentication keys | None | Redaction-safe drop. Do not retain values in a model field, diagnostic, log, or rendered round trip. |

Canonical inputs remain accepted unchanged. If both a canonical field and its alias are present, canonical wins; the alias is discarded without exposing its value. This deterministic precedence avoids an ambiguous overwrite and does not alter authentication policy.

## Parser surface and minimum change

`internal/service/auth_surfaces.go` is the boundary for sandbox extraction and canonical rendering. Its exported helpers (`ParseAuthenticationSurfacesViaSandbox`, `ParseAdminAuthViaSandbox`, and `RenderHTTPAuthCanonicalKeys`) should retain their canonical responsibilities. `ParseAuthenticationSurfacesViaSandbox` currently extracts the canonical Node-RED names; the migration belongs between extraction of settings/model values and the structured settings renderer, not as a renderer-only string substitution.

The preferred minimal implementation is a new `NormalizeLegacyAuthFields(in *model.NodeRedConfig) error` step invoked immediately after the relevant legacy fields have been parsed and before the `Settings.ts` / settings-JS rendering path. It should:

1. Move supported legacy values into their canonical fields only when the canonical destination is absent.
2. Convert supported single-user `adminAuth` input into the existing `AdminAuth.Users` representation without changing credential bytes or permissions; preserve existing multi-user order and fields.
3. Clear recognized alias fields after migration and discard unknown legacy auth fields rather than carrying them into `Extra` or source output.
4. Return a non-sensitive error only for structurally invalid recognized input; errors must not interpolate usernames, hashes, or raw settings text.

Do not add parallel aliases to the public model unless parsing proves they are required to carry legacy values to normalization. Do not alter handler, middleware, or policy behavior. If the current parse → canonical model → render path already satisfies these invariants, add only the test and document the single-GREEN rationale in the PR body; do not add an unused helper.

## Data flow

1. `ConfigService.parseConfigFromContent` reads settings input and extracts auth surfaces through the existing sandbox/parser boundary.
2. Legacy field names/shapes are normalized into `NodeRedConfig` canonical fields before validation or re-render.
3. Validation operates on canonical auth structures; its errors remain redacted, guarded by `TestHTTPAuthValidationRedactsPasswords` (`internal/service/auth_surfaces_test.go:73`).
4. The existing renderer emits only `adminAuth`, `httpNodeAuth`, and `httpStaticAuth` with pre-hashed values. No legacy aliases or plaintext credentials flow to round-trip output.
5. A subsequent parse yields a struct equal to the normalized first parse for the auth surfaces covered by the contract.

## Test and fixture strategy

Add `TestLegacyAuthFieldMigration` to `internal/service/auth_surfaces_test.go` as a table-driven test with named `t.Run` sub-cases:

| Sub-case | Input and assertions |
|---|---|
| Legacy single-user `adminAuth` | Exactly one canonical user; username, `testBcryptHash`, and permissions preserved. |
| Legacy multi-user `adminAuth` | User order/count and distinct permissions preserved; every password equals `testBcryptHash`; second parse equals normalized first parse. |
| `httpAuth` → `httpNodeAuth` | Canonical node-auth populated with the same username/hash; static-auth remains nil; rendered settings use `httpNodeAuth` only. |
| Static alias → `httpStaticAuth` | Test `nodeHttpAuth` and/or `staticAuth` aliases; static-auth populated; node-auth remains nil; rendered settings use `httpStaticAuth` only. |

Keep fixtures inline in the existing test file. Use the existing `testBcryptHash` constant exclusively for password material; fixtures must contain no plaintext credential values and must not perform real bcrypt work. Assert canonical positive output, forbidden alias absence (`httpAuth`, `nodeHttpAuth`, `staticAuth`), hash byte equality, and parse-render-parse stability where applicable. Reuse the redaction gate `TestHTTPAuthValidationRedactsPasswords` and add an assertion only if a new error path is introduced; never assert by exposing a secret in test diagnostics.

## Strict-TDD work-unit sequence

- **RED:** Add only `TestLegacyAuthFieldMigration`; run the focused service tests and record a clear failure naming the missing canonical destination. If current parser behavior already passes, skip a manufactured failing commit and use one GREEN test-only commit; state the observed passing behavior and reason in the PR body.
- **GREEN:** If RED demonstrates missing migration, add the minimal normalization/parser change in a separate immediately following work-unit commit. Rerun the focused tests and the required race-enabled command.
- **REFACTOR (optional):** Separate only if useful. It must not change behavior or add/remove test cases.

Each commit remains independently reviewable and uses a Conventional Commit message. Keep test and parser behavior paired within the prescribed RED/GREEN sequence. Forecast the complete authored diff against the 400-line budget; `ask-on-risk` applies if the actual slice creates a delivery decision.

## Slice boundaries and exclusions

| Slice | Scope | Status |
|---|---|---|
| 1 | Go table-driven unit test in `internal/service/auth_surfaces_test.go`; minimum parser normalization only if needed. | This design. |
| 2 | `SecuritySurfaceIsolationE2E` Playwright spec in `frontend/e2e/security-center.spec.ts`, using the stack fixture. | Deferred; orchestrator re-`ask-on-risk` at slice-1 close. |

Explicitly out of scope for slice 1:

- Handler or middleware changes.
- Authentication-policy changes or new cross-surface enforcement.
- The Playwright spec and any frontend/e2e changes.
- Modifications to the five existing capability specs: `dashboard-runtime-metrics`, `metrics-endpoint`, `user-management-ui`, `user-update-api`, and `user-role-editing`.
- Settings persistence changes beyond the minimum legacy-field normalization required by the parser contract.
- Rewriting `odd/audits/issue-765/gap-report.md` or changing OpenSpec configuration.

## Files and contracts

| Path | Planned relationship |
|---|---|
| `internal/service/auth_surfaces_test.go` | Add named table-driven migration test using `testBcryptHash`; retain the existing redaction test. |
| `internal/service/auth_surfaces.go` | Change only if required to expose/apply the minimum normalization at the correct parser boundary; preserve canonical rendering semantics. |
| `internal/service/config.go` | Wire normalization before validation/re-render only if needed; avoid unrelated settings parser changes. |
| `internal/service/settings_sandbox.go` | Keep sandbox extraction bounded and redacted; extend alias extraction only if the parser path requires it. |
| `openspec/specs/authentication-surfaces-closure/spec.md` | Contract source; not modified by implementation. |
| `frontend/e2e/security-center.spec.ts` | Deferred slice 2; untouched in slice 1. |

The test demonstrates the Go acceptance item from Gap G2 and documents that `SecuritySurfaceIsolationE2E` remains outstanding. The slice-1 PR must identify Gap G2, name `TestLegacyAuthFieldMigration`, state that `SecuritySurfaceIsolationE2E` is deferred, and link the #760 cluster issue as required by the repository PR process.

## Verification and rollout

- Focused: `go test -race -count=1 -timeout 20m ./internal/service/...` — required; race detector must report no races.
- Repo lint: `golangci-lint run --config=.golangci.yml ./...` — required zero findings.
- Build constraint: run on Linux; the project is not buildable on macOS/Windows.
- Review budget: keep authored slice-1 additions plus deletions at or below 400 lines. Do not include slice 2 to fit or exceed the budget.
- Rollout: merge slice 1 only after checks pass. Treat its merge as Go-half closure evidence for Gap G2, not full #760 closure. Pause and `ask-on-risk` again before deciding whether to promote slice 2.

## Risks

| Risk | Mitigation |
|---|---|
| Linux-only build constraint | Run the Go verification on Linux; no cross-platform build claim. |
| Repo-wide zero-finding lint | Run the exact `golangci-lint` command; fix only findings introduced by this slice. |
| Race detector regression | Run the required service test command with `-race -count=1`; keep the test self-contained and deterministic. |
| Fork-PR CI gate | Confirm the repository's `safe-to-test` requirement where heavy CI is skipped for untrusted forks; do not change CI workflow in this slice. |
| Parser already implements migration | Use the contract test as the single GREEN commit and document evidence in the PR body; do not add a needless parser change. |
| Plaintext leakage or unknown-field retention | Use bcrypt-only fixtures, canonical-only output assertions, generic errors, and the existing password-redaction test as the gate. |
| Ambiguous legacy/canonical duplicates | Canonical destination wins; discard alias data without logging or retaining it. |
