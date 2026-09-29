# Apply pipeline

> **Stub.** Landed as a skeleton in W1. Full content (import →
> edit → preview/diff → validation → atomic write → backup →
> readiness → rollback) lands in W4.

## Stages

1. **Import.** NRCC reads `settings.js` losslessly.
2. **Edit.** Operator changes a typed form field or opens an
   advanced escape hatch (#764).
3. **Preview / diff.** NRCC shows the redacted diff.
4. **Validation.** Schema and runtime checks.
5. **Atomic write.** New `settings.js` written via tmp + rename.
6. **Backup.** A snapshot is taken before write.
7. **Readiness.** NRCC restarts Node-RED and waits for ready.
8. **Rollback.** On readiness failure, the backup is restored and
   Node-RED is restarted again.

See [`glossary.md`](../glossary.md) for definitions of every
stage. The detailed contracts land in W4.
