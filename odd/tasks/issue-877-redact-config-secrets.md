# Issue #877 — Redact credential and TLS secrets from config responses

## Objective

Issue #877 (`fix(security): redact credential and TLS secrets from viewer
config responses (P0)`) reports that `GET /api/config` returns
`credentialSecret` and `https.passphrase` in cleartext. The MEDIUM-015
redaction block in `internal/handler/config.go` covers admin-auth hashes
and environment values, but it omits these two secrets, so any
authenticated viewer can read the Node-RED credential-encryption secret
and the TLS private-key passphrase.

The fix makes both fields **write-only for every role**: no configuration
read or apply response carries the stored value, and a write that carries
an empty or redacted field keeps the stored secret instead of wiping it.

- Issue: <https://github.com/fgjcarlos/nrcc/issues/877>
- Branch: `fix/877-redact-config-secrets`
- Base: `origin/main` = `a0ce822563218cb1098ffe6cdfb4eec68db4601e`
- Worktree: `/home/composedof2/Dev/Codex/nrcc-877` (isolated: the primary
  checkout holds unrelated uncommitted work for #875)

## Problem and why it matters

`credentialSecret` encrypts `credentials.json` at rest; `https.passphrase`
unlocks an encrypted TLS private key. Both are operator secrets that must
not leave the host inside an API response. The frontend already treats
`credentialSecret` as write-only (issue #762): the field renders blank and
the help text says "Leave blank to keep current value". The backend
never enforced that contract, so the API response carried the value even
though the UI never used it.

Beyond the reported viewer leak, the same values travelled through every
config-bearing response (`GET /api/config`, the `POST /api/config` echo,
the `POST /api/config/apply` commit echo) and could be echoed back by a
client, silently overwriting or wiping the stored secret.

## Authorized policy and constraints

- **Redaction scope (user decision): every role.** Admin and viewer
  responses never carry `credentialSecret` or `https.passphrase`. The
  configuration API treats both as write-only values.
- **Write-back contract.** A write that carries an empty value or a
  redaction placeholder means "keep the stored value". An explicit new
  value rotates the secret; the literal `false` keeps its documented
  meaning for `credentialSecret` (disable credential encryption).
- **Out of scope by design.** `GET /api/settings/raw` (admin-only) returns
  the literal `settings.js` file for the raw editor; redacting it would
  corrupt the file the operator is editing. Node-RED on-disk paths
  (`https.key`, `https.cert`, `https.ca`) are not secrets and keep being
  returned.
- No frontend behaviour change is required: the configuration form already
  submits an empty `credentialSecret` / `httpsPassphrase` unless the
  operator types a new value, which is exactly the "keep current value"
  signal the backend now honours.
- Repository language rule: every artifact in this feature is written in
  English (see `AGENTS.md`).
- No push, no pull request, no merge, no issue-label change: those stay
  human decisions.

## Tasks

- [x] **T1 — Regression coverage (RED).** Added the issue's reproduction
      test verbatim (`TestReleaseAuditViewerMustNotReceiveTLSOrCredentialSecrets`),
      the admin-role variant, the "read does not modify persisted state"
      assertion, and the write-back preservation assertions. Observed the
      expected failures before implementing.
- [x] **T2 — Write-only secret contract in the model package.** Added the
      redaction marker, the "is this value redacted?" helper, and the
      non-mutating `RedactedSecrets()` projection on `model.NodeRedConfig`,
      with unit tests (`internal/model/config_secrets.go`).
- [x] **T3 — Write-back preservation in the service package.** `Save` and
      `CommitStructured` now preserve the stored `credentialSecret` and
      `https.passphrase` when a save carries an empty or redacted value.
      Rotation, the `false` opt-out, and the redacted-marker inputs are
      covered.
- [x] **T4 — Redaction at the response boundary.** Applied the write-only
      projection to every configuration response (`GET /api/config`,
      `GET /api/config/default`, the `POST /api/config` echo in both the
      legacy and the apply path, and the `POST /api/config/apply` commit
      echo) for every role.
- [x] **T5 — Export-surface hardening.** The AI provider request redacts
      secret-shaped flow keys; `passphrase` was not matched by that
      pattern, so a flow property named `passphrase` left the host in
      cleartext. The pattern now matches it, and a test covers it.
- [x] **T6 — Documentation and verification.** Updated the OpenAPI schema
      and the settings catalog for the write-only contract, ran the Go
      suite, `gofmt`, and `git diff --check`, and closed every task with a
      work-unit commit on the feature branch.

## Acceptance criteria

- [x] No configuration read or apply response contains `credentialSecret`
      or `https.passphrase`, for admin or viewer.
- [x] The persisted configuration and `settings.js` keep the stored secret
      after a redacted read and after a save that carries an empty or
      redacted field.
- [x] An explicit new value still rotates the secret; `credentialSecret:
      false` still disables credential encryption.
- [x] Non-secret TLS material (on-disk key/cert/CA paths, port) is still
      returned, so the configuration UI keeps working.
- [x] Regression tests assert the complete response body carries no
      fixture secret.
- [x] `go build ./...`, `go test ./...`, `gofmt` and `git diff --check`
      pass, with any skipped or failed check reported honestly.

## Verification evidence

**RED baseline (before the fix).**

```
$ go test ./internal/handler -run TestReleaseAuditViewerMustNotReceiveTLSOrCredentialSecrets -count=1
    viewer response includes audit-credential-fixture
    viewer response includes audit-tls-fixture
$ go test ./internal/handler -run TestConfigHandler_ -count=1
    admin response includes audit-credential-fixture
    stored credentialSecret = "", want it preserved
    stored https.passphrase must be preserved
    save response includes rotated-credential-fixture
$ go test ./internal/service -run TestConfigService_ -count=1   # without the service fix
    rendered settings.js must keep the stored passphrase; got ... https: { key, cert, port }
$ go test ./internal/service -run TestRedactSecretsRedactsPassphraseKeys -count=1   # without the AI fix
    redacted output leaked the passphrase: {"nodes":[{"id":"tls-1","passphrase":"audit-tls-fixture",...}]}
```

**GREEN (after the fix).**

```
$ go build ./...                                     # OK
$ go vet ./...                                       # OK
$ go test ./...                                      # 14 packages ok, 1 without tests, 0 failures
$ go test -race -count=1 -timeout 20m ./...         # all packages ok (CI parity)
$ golangci-lint run --config=.golangci.yml ./...    # 0 issues
$ npx --yes @redocly/cli lint docs/openapi.yaml     # valid, same 9 pre-existing warnings as main
$ gofmt -l <changed .go files>                       # no output
$ git diff --check                                   # clean
$ go run ./tools/langscan .                          # 0 findings
$ go run ./scripts/cite_check ./docs                 # OK: no broken internal links
$ go test ./internal/handler -run TestReleaseAuditViewerMustNotReceiveTLSOrCredentialSecrets -count=1   # ok
```

The repo-wide `golangci-lint` gate is green. The invented credential and
passphrase fixtures are annotated with the repository's existing
`//nolint:gosec` test convention (G101) and the test-only settings path
carries the matching G304 annotation, so the gate stays clean without
weakening the rule.

The repository is not gofmt-clean at `origin/main` (for example
`internal/model/response.go` tag alignment), so only the files touched by
this feature were checked; none of them contributes a new gofmt diff.

## Known boundaries

- `GET /api/settings/raw` and the raw apply echo still return the literal
  `settings.js` for the admin raw editor, by design (redacting it would
  corrupt the file being edited).
- The preservation source is the NRCC configuration store. When the
  passphrase exists only in a hand-edited `settings.js` that NRCC has
  never saved, it stays in that file untouched, but NRCC cannot restore
  it into its own store.
- A `nil` `https` block still removes the TLS configuration, including its
  passphrase. That is the established #762 contract
  (`TestSave_RemovesHttpsBlockWhenCleared`,
  `TestPatchSettingsJS_RemovesHttpsBlockWhenNil`), and the frontend sends
  the block whenever the operator keeps any TLS field set.
- A pre-existing renderer limitation stays out of scope: the settings.js
  `https` block is only patched in place when the document already
  contains it (the block-append path does not match a document ending in
  `};`).

## Next step

Every task is closed. The branch carries five work-unit commits and the
pull request is open: <https://github.com/fgjcarlos/nrcc/pull/921>.

- `5112684` `fix(security): keep stored config secrets on write-back`
- `b9d0cbb` `fix(security): redact credential and TLS secrets from config responses`
- `df5f954` `fix(security): redact passphrase keys in AI provider requests`
- `1fd3744` `docs: document the write-only config secret contract`
- `2e0b9cf` `test(security): annotate secret fixtures for the gosec gate`

Delivery decision (maintainer, human-controlled gate): the slice is 953
changed lines, above the 400-line review budget for a PR slice, so the pull
request carries the `size:exception` label with the rationale documented in
its body. The fix is atomic (redaction without write-back preservation
would wipe the stored secrets), and the branch still splits it into
reviewable commits. The pull request closes #877 on merge.

Remaining human decisions: review and merge the pull request.
