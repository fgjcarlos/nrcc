# Setup recovery

On first boot NRCC writes a random 256-bit token to
`$DATA_DIR/.nrcc-setup-token` (`/data/.nrcc-setup-token` in the container).
The file is atomically published with mode `0600`; keep an encrypted,
operator-only backup on a separate protected volume.

If recovery is required:

1. Stop NRCC so user and token files cannot change.
2. Restore the backed-up token to `$DATA_DIR/.nrcc-setup-token` with mode `0600`.
3. Start NRCC and send it once as `X-Setup-Reset-Token` to `/api/auth/setup`.
4. Verify the administrator was added and the token file was removed.
5. Destroy every copy of the consumed backup.

If no protected token backup exists, keep NRCC stopped and restore
`$DATA_DIR/cc-users.json` plus the matching token from one atomic backup set.
Never edit live user state or generate an ad-hoc replacement token.
