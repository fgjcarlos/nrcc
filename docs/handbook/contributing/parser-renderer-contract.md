# Parser/renderer contract

> **Stub.** Landed as a skeleton in W1. Full contract arrives in W5.

## Invariants

- Round-trip unmanaged regions verbatim.
- Preserve original whitespace and comments where possible.
- Refuse to coerce `ErrSourceNotExports`.
- Idempotent: applying the same edit twice is a no-op on disk.

## Where the code lives

- Parser: `internal/service/source_patch.go` (`ApplyScalarEdit`,
  `ApplyObjectEdit`, `ApplyArrayEdit`).
- Tests: `internal/service/source_patch_test.go`.
- Renderer form generation:
  `frontend/src/features/configuration/components/FormFields.tsx`.

Detailed contracts land in W5.
