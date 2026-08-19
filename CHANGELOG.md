# Changelog

## Unreleased

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
