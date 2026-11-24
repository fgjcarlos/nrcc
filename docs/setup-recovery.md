# Setup and administrator recovery

## First boot

NRCC writes a random 256-bit token to `$DATA_DIR/.nrcc-setup-token`
(`/data/.nrcc-setup-token` in the container) with mode `0600`. Run
`docker compose logs nrcc` on the host and copy the value from the
`auth setup token:` line into the setup form. Keep the token and startup logs
private. API clients send it in `X-Setup-Reset-Token`; on an unconfigured
instance, a missing or invalid token returns `401`.

NRCC removes the token file before creating the administrator. If setup fails
after consumption and no administrator was created, restart NRCC to issue a
new first-run token. If the administrator was created, try signing in before
attempting recovery.

On an already-configured installation upgraded from a version that left its
first-run token behind, remove the stale token file while NRCC is stopped.
A token file on a configured installation authorizes recovery.

## Recover an administrator

1. Stop NRCC. On the host, set `DATA_DIR` to the actual NRCC data directory
   and run the following as the OS user that owns those data files. If a token
   file already exists, remove it first to revoke it; never reuse a token that
   was previously issued.

   ```bash
   (
     set -eu
     umask 077
     tmp=$(mktemp "${DATA_DIR:?Set DATA_DIR to the NRCC data directory}/.nrcc-recovery.XXXXXXXX")
     trap 'rm -f -- "$tmp"' EXIT
     token=$(openssl rand -hex 32)
     created_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
     printf '{"token":"%s","createdAt":"%s"}\n' "$token" "$created_at" > "$tmp"
     ln -- "$tmp" "$DATA_DIR/.nrcc-setup-token"
   )
   ```

2. Read the `token` field from the file privately. Start NRCC, then send the
   new administrator credentials to `/api/auth/setup` with
   `X-Setup-Reset-Token` set to that value. Recovery tokens do not appear in
   the startup log of a configured instance.
3. Verify the new administrator can sign in and the token file is absent.
   Revoke any unconsumed recovery token by stopping NRCC and removing the
   file. If NRCC consumed the token but did not create the administrator,
   issue a fresh recovery token before retrying.

Do not edit live user state. Store account backups separately from recovery
credentials.

## Run browser acceptance against a disposable stack

Start an uninitialized stack and pass its token to Playwright through the
environment:

```bash
E2E_SETUP_TOKEN="$(docker compose logs --no-color nrcc | sed -n 's/.*auth setup token: \([0-9a-f]\{64\}\).*/\1/p' | tail -1)" pnpm --dir frontend run test:e2e:stack
```

Set `E2E_BASE_URL` and `E2E_NODE_RED_URL` when the stack uses non-default ports.
