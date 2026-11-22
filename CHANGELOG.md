# Changelog

## Unreleased

- **Behavior change:** NRCC no longer attaches to a Node-RED it did not
  start. With `NRCC_MANAGE_NODE_RED=true` (the default), start now fails with
  "node-red port is already bound by a foreign process" when another process
  binds the Node-RED port; set `NRCC_MANAGE_NODE_RED=false` to run beside a
  Node-RED another supervisor owns. The broken attach mode marked the runtime
  running without a process, so a later stop did nothing and a start crashed.
  The `running (external)` status and the `EXTERNAL_NODE_RED` error are gone.
- Node-RED's stdout and stderr now appear in the NRCC container log
  (`docker logs nrcc`), each line prefixed with `node-red: `, so a failed start
  (for example a syntax error in `settings.js`) explains itself. Previously the
  output was discarded.
- Installing a library now activates it in settings.js: `functionExternalModules`
  is enabled and plain npm libraries are exposed in `functionGlobalContext` under
  their alias (guarded `require`, so a module that fails to load cannot stop
  Node-RED). Uninstall removes the NRCC-managed entry; operator-authored entries
  are preserved. The install `alias` field is no longer ignored.
- Align the frontend on Tailwind CSS 4 with its Vite integration and daisyUI 5,
  preserving the corporate themes and setting the styling browser floor to Safari
  16.4, Chrome 111, and Firefox 128. The Tailwind 3 braces advisory route is removed.
- **Security — frontend dependencies:** updated PostCSS, fflate, Vitest and
  @vitest/mocker, and js-yaml to patched releases for five moderate package
  records. The separate HIGH advisory affecting `braces` remains unresolved.
- **Security — frontend dependencies:** raised the `undici` and
  `brace-expansion` override floors to address four HIGH advisories.
  The separate HIGH advisory affecting `braces` remains unresolved.
- Derive update capability from the managed Node-RED runtime, fail closed for
  unknown versions and unsupported installations, and explain image redeploys
  without suggesting npm as a whole-product upgrade.
- Coalesce session refreshes during authentication bootstrap and API 401 recovery,
  preventing concurrent refresh-cookie rotation from signing users out on reload.
- Normalize Node-RED CLI version output so version detection and the existing
  configuration compatibility gate recognize Node-RED 5 runtimes.
- Harden authentication setup with an atomic one-time recovery token,
  guarded re-bootstrap, normalized throttling, and TLS-aware cookies.
- **Security — refresh tokens (#669):** `sessions.json` no longer stores
  refresh tokens in cleartext. The persisted session id is now
  `sha256(token)`; the raw token is returned to the client and never
  written to disk. Validation and revocation hash the incoming token
  and compare with `subtle.ConstantTimeCompare`. **Upgrade note:** any
  sessions issued by a previous version become invalid; affected users
  will be signed out and must re-authenticate once. This is a one-time
  forced re-login.
- **Security — `/metrics` (#671):** the Prometheus endpoint now requires
  authentication by default. Set `NRCC_METRICS_PUBLIC=true` to opt back
  into the previous unauthenticated behaviour (e.g. when the metrics
  port is bound to a private network). Documented in `SECURITY.md`.
- **Security — backup encryption password (#670):** the passphrase used
  to wrap a backup archive is no longer accepted as a `?password=…`
  query parameter (which would have landed in proxy access logs and
  browser history). Pass it in the `X-Backup-Password` request header
  instead. The query-parameter form is still accepted for one release
  cycle and emits a deprecation warning in the server log.
