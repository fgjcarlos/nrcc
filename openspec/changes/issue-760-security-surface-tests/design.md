# Authentication Surfaces Closure — Design

Slice 1 closes the Go half of Gap G2 for cluster #760 by adding a table-driven parser contract test and, only if that test exposes a defect, the smallest parser normalization needed to satisfy it. This is test closure, not an authentication-policy change. `SecuritySurfaceIsolationE2E` is explicitly deferred to chained slice 2.

## Decision summary

| Concern | Decision |
|---|---|
| Migration boundary | Normalize legacy auth fields while extracting settings through `ParseAuthenticationSurfacesViaSandbox` / `extractAuthenticationSurfaces` in `internal/service/settings_sandbox.go`, before the canonical model is consumed by `parseConfigFromContent` in `internal/service/config.go`. |
| Rendering | Keep `renderSettingsJS` / `generateSettingsJS` rendering canonical model fields only. Do not implement renderer-only alias substitution: it would leave validation and the structured config non-canonical. |
| Minimum production change | Add migration only if the new test proves a gap. Accept legacy aliases into the existing `model.AuthenticationSurfaces` fields, favor a present canonical destination over an alias, and do not add parallel public model aliases unless extraction requires them. If parser behavior already satisfies the test, use one GREEN test-only work-unit commit. |
| Unmanaged and unknown fields | Preserve unmanaged source under the existing source-patching contract. Do not carry unknown auth aliases into managed auth fields, `Extra`, diagnostics, logs, or rendered output. |
| Slice boundary | Slice 1 is `TestLegacyAuthFieldMigration` and a conditional minimal parser correction. `SecuritySurfaceIsolationE2E` is slice 2, deferred and not a slice-1 dependency. |
| Delivery | Strict TDD; keep the slice at or below 400 authored changed lines. The strategy is `ask-on-risk`; re-ask at slice-1 close before promoting slice 2. |

## Legacy-to-canonical migration map

| Legacy NRCC settings field / shape | Canonical Node-RED 5.x surface | Required handling |
|---|---|---|
| `adminAuth` legacy single-user shape | `adminAuth` | Normalize to one `AdminAuth.Users` entry. Preserve username, bcrypt hash bytes, and permissions. |
| `adminAuth` legacy multi-user shape | `adminAuth` | Preserve user order, count, hashes, and each `permissions` value. |
| `httpAuth` | `httpNodeAuth` | Populate `HTTPNodeAuth`; leave `HTTPStaticAuth` nil. |
| `nodeHttpAuth` or `staticAuth` | `httpStaticAuth` | Populate `HTTPStaticAuth`; leave `HTTPNodeAuth` nil. |
| Unrecognized legacy authentication keys | None | Drop from the managed projection without retaining or exposing their values. |

Canonical fields remain accepted unchanged. If a canonical destination and its alias are both present, canonical wins and the alias is discarded without exposing its value. This deterministic precedence avoids ambiguous overwrites and does not alter runtime authentication policy.

## Parser surface and minimum change

There is no `internal/service/auth_surfaces.go` in the current checkout. The actual parser boundary is `internal/service/settings_sandbox.go`; `internal/service/config.go` calls it via `parseAuthenticationSurfacesFromJS` from `parseConfigFromContent`. The current sandbox comment and implementation explicitly recognize only canonical `adminAuth`, `httpNodeAuth`, and `httpStaticAuth`; the aliases are not migrated implicitly. Existing canonical rendering is in `generateSettingsJS` / `renderHTTPBasicAuthBlock` in `config.go`.

The preferred minimal correction, if required by RED, is a focused extension to `extractAuthenticationSurfaces` in `settings_sandbox.go`, rather than a string rewrite in the renderer:

1. Read the supported legacy alias only when its canonical destination is absent; canonical values take precedence.
2. Route `httpAuth` to `HTTPNodeAuth`, and `nodeHttpAuth` / `staticAuth` to `HTTPStaticAuth`—never cross-map them.
3. Normalize supported legacy single-user `adminAuth` input into the existing `AdminAuth.Users` representation while retaining multi-user order and fields.
4. Ensure canonical extraction and `parseConfigFromContent` continue to expose only the existing model surface fields. Unknown auth aliases must not flow into `Extra` or diagnostics.
5. Return only non-sensitive structural errors if any are needed; never interpolate usernames, hashes, or raw settings text.

Do not broaden the public model, change handlers/middleware/persistence, or change authentication policy. If parsing already yields the required canonical structures, do not add an unused normalization helper or parser change.

## Data flow

1. `ConfigService.parseConfigFromContent` reads the settings source and invokes `parseAuthenticationSurfacesFromJS`.
2. `ParseAuthenticationSurfacesViaSandbox` executes the source in the existing bounded Goja sandbox; `extractAuthenticationSurfaces` obtains the supported auth values and is the appropriate point to map legacy aliases into canonical model fields.
3. `parseConfigFromContent` places those fields into `model.NodeRedConfig`; validation consumes that canonical structure. Existing redaction behavior is guarded by `TestHTTPAuthValidationRedactsPasswords` in `internal/service/auth_surfaces_test.go`.
4. `renderSettingsJS` / `generateSettingsJS` emit `adminAuth`, `httpNodeAuth`, and `httpStaticAuth` using already-hashed credential material. Legacy alias keys are not emitted by the canonical renderer.
5. Parse → render → parse preserves the canonical auth structure, including hash bytes, user order/count, and permissions.

## Test and fixture strategy

Add `TestLegacyAuthFieldMigration` to `internal/service/auth_surfaces_test.go`. Keep it table-driven with named `t.Run` sub-cases and inline fixtures. Use the existing `testBcryptHash` constant for every password fixture; do not include plaintext credentials or perform bcrypt work.

| Required sub-case | Contract assertions |
|---|---|
| Legacy single-user `adminAuth` | One `AdminAuth.Users` entry; username, `testBcryptHash`, and permissions preserved through render/parse. |
| Legacy multi-user `adminAuth` | User order and count preserved; distinct permissions preserved; hashes byte-identical; second parse equals the first normalized parse. |
| `httpAuth` → `httpNodeAuth` | `HTTPNodeAuth` contains the fixture user/hash; `HTTPStaticAuth` is nil; canonical rendered key exists. |
| Static alias → `httpStaticAuth` | Cover the supported `nodeHttpAuth` and/or `staticAuth` alias; `HTTPStaticAuth` contains the fixture user/hash; `HTTPNodeAuth` is nil; canonical rendered key exists. |

Exercise the same production path used by the existing tests (`parseConfigFromContent` and `renderSettingsJS` or `patchSettingsJS` as appropriate); no network or non-temporary filesystem state. Assert that rendered output does not contain `httpAuth`, `nodeHttpAuth`, or `staticAuth`, and does not introduce plaintext credentials. Retain the existing canonical-render and unmanaged-source-preservation tests as regression coverage. The existing password-redaction test remains the gate for error-path secrecy; add no secret-bearing diagnostic assertions.

## Strict-TDD work-unit sequence

1. **T1 RED:** Add only `TestLegacyAuthFieldMigration` and fixtures in `internal/service/auth_surfaces_test.go`. Run the focused service suite and confirm a clear failure identifies the missing canonical destination or round-trip invariant. Do not change production code in this work unit.
2. **T2 GREEN (conditional):** If RED demonstrates missing migration, make the minimum parser change in `settings_sandbox.go`, then rerun the focused service suite and required race-enabled command. If the existing parser already satisfies the test, a single GREEN test-only commit is acceptable; document the observed pre-change pass and why no RED failure or parser fix was appropriate. Do not manufacture a failing commit.
3. **T3 REFACTOR (optional):** Refactor only if it materially improves clarity. Preserve behavior and the exact test-case set; verify the same test matrix remains green.

This ordering follows `strict_tdd.enabled: true`, `requires_failing_test_first: true`, and `red_green_refactor: true`. Each work unit should remain reviewable and use a Conventional Commit message. Pause and ask on any risk trigger, including auth/secrets scope expansion or a projected over-budget diff; do not infer an exception or chain strategy.

## Slice boundaries

| Slice | Scope | Status |
|---|---|---|
| 1 | Go table-driven `TestLegacyAuthFieldMigration`; minimum parser correction only if the test demonstrates a defect. | This change. |
| 2 | Playwright `SecuritySurfaceIsolationE2E` in `frontend/e2e/security-center.spec.ts`, exercising surface isolation against the stack. | Deferred chained follow-up; requires slice 1 GREEN on `origin/main` and a renewed `ask-on-risk` decision. |

Slice 1 must not touch frontend files, handlers, middleware, persistence, existing capability specs, or the audit report. The deferred E2E is not bundled into slice 1 under any circumstance.

## Files and contracts

| Path | Planned relationship |
|---|---|
| `internal/service/auth_surfaces_test.go` | Add the table-driven migration test using `testBcryptHash`; retain existing redaction and canonical-render tests. |
| `internal/service/settings_sandbox.go` | Actual sandbox extraction boundary; extend only if required to migrate aliases before model projection. |
| `internal/service/config.go` | Existing `parseConfigFromContent` integration and canonical renderer; no renderer-only alias substitution or unrelated parser changes. |
| `internal/model` auth types | Existing canonical `NodeRedConfig` / `AuthenticationSurfaces` contract; avoid public model expansion unless extraction proves necessary. |
| `frontend/e2e/security-center.spec.ts` | Deferred slice 2; untouched in slice 1. |
| `odd/audits/issue-765/gap-report.md` | Read-only audit source; Gap G2 is referenced, not edited. |

The PR evidence for slice 1 must identify Gap G2, name `TestLegacyAuthFieldMigration`, explicitly defer `SecuritySurfaceIsolationE2E` to chained slice 2, and link the #760 cluster issue. This slice closes only the Go half of the audit gap, not the full cluster.

## Verification and rollout

Run verification on Linux, consistent with the repository's Linux-only build constraint:

- Focused required suite: `go test -race -count=1 -timeout 20m ./internal/service/...` — migration test and existing service tests pass with no race reports.
- Repository lint gate: `golangci-lint run --config=.golangci.yml ./...` — zero findings.
- Review workload: authored additions plus deletions remain at or below 400 lines. If risk exceeds the threshold or auth/security scope expands, stop and ask under `ask-on-risk`.

Merge slice 1 only after its checks pass. Treat the merge as Go-half closure evidence for Gap G2, not full #760 closure. Before promoting slice 2, pause for the required delivery decision and ensure the follow-up remains a separate chained slice.

## Risks

| Risk | Mitigation |
|---|---|
| Parser already implements the migration, so RED is not attainable. | Accept one GREEN test-only commit with the observed passing behavior and rationale documented; never manufacture a failure. |
| Alias extraction accidentally maps `httpAuth` to the static surface or a static alias to the node surface. | Pin both destination and opposite-surface-nil assertions in named sub-cases; use the explicit migration map. |
| Credential hashes or permissions change during normalization or round-trip. | Assert byte-identical `testBcryptHash`, user count/order, permissions, and parse-render-parse stability. |
| Credential material leaks through errors or output. | Use only the hash constant, retain the redaction test, emit canonical output, and keep structural errors generic. |
| Parser fix broadens beyond the minimum or alters auth policy. | Limit production changes to sandbox extraction/model normalization; no handler, middleware, persistence, or policy changes. |
| The slice exceeds the review budget or absorbs E2E work. | Keep slice 1 Go-only and under 400 authored lines; stop and ask on risk; defer E2E to slice 2. |
| Linux-only assumptions invalidate verification elsewhere. | Run the required Go verification on Linux; make no macOS/Windows build claim. |
