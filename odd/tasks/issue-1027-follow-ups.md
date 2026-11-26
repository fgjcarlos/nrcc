# Issue 1027: follow-up work

These findings were not closed by the small fixes in the [issue 1027 task record](issue-1027-v1-hardening.md). Each row is a separate follow-up, not an implementation claim. No GitHub issues have been created for these rows.

## Security and compatibility

| Spec | What remains | Boundary for the follow-up |
| --- | --- | --- |
| S3 | Structured saves intentionally fill in `httpStatic: "uploads"` when no static mount is configured (`internal/service/config.go`). Exposure depends on the active Node-RED access policy and was not tested on a live port. | Define an explicit publication policy for Dashboard assets. Keep private upload/download behavior and operator-authored mounts distinct. |
| S4 | JWT claims remain usable until expiry; naively looking up the user in every middleware invocation reads and decodes the full JSON user store. | Define an authoritative role/revocation lookup and bounded invalidation. Verify deletion and demotion take effect without adding an unbounded per-request I/O path. |
| S5 | TOTP login validates a code but does not track which time step was consumed. | Design an atomic per-enrollment time-step check that respects clock skew and restart continuity; keep recovery codes and previously enrolled accounts compatible. |
| S8 | Registry-only selector validation is fixed; npm still runs lifecycle scripts for registry packages. | Define a script policy separately from selector validation. Verify supported Node-RED nodes that require native build or install steps before changing the default. |
| S10 | The in-process JavaScript settings parser has an interrupt deadline but no independent memory limit. | Use an isolation boundary with enforceable resource budgets and retain support for the currently accepted settings shapes. |
| S11 | Existing encrypted environment values use the legacy SHA-256-derived key scheme. | Version new writes, retain a read path for existing values, and define key requirements and a documented upgrade path before changing the encryption contract. |

## Runtime and storage

| Spec | What remains | Boundary for the follow-up |
| --- | --- | --- |
| S12 | Both apply endpoints pass the request context into the transaction. Simply dropping its cancellation also drops its deadline and disconnects the operation from shutdown. | Assign admitted applies a bounded service lifecycle, preserve revision checks and serialized commits, and coordinate completion or rollback with Node-RED shutdown. Test both disconnect and real shutdown. |
| S17 | The 100 MiB multipart body limit exceeds the Compose temporary filesystem budget, and the server's 15-second read timeout applies to the full body. | Decide the supported upload budget. If keeping 100 MiB, stage bounded requests without exhausting temporary storage, preserve existing files on failed uploads, and test the real Compose limits. |
| S18 | Settings snapshots and failed-apply copies are retained indefinitely. They are distinct from scheduled backup archives and may be recovery or diagnostic artifacts. | Define retention and a safe startup/maintenance point. Preserve all artifacts referenced by active recovery and document how existing files are treated before enabling deletion. |
