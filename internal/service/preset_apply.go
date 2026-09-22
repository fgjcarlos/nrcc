// Package service — slice 2 of issue #764.
//
// PresetApply is the apply path for a curated preset. It composes:
//
//   1. PresetRegistry lookup — fail closed on unknown preset ID.
//   2. Preset.BuildEdits — pure, deterministic, no I/O.
//   3. SourcePatch — the existing source-preserving batch patcher.
//      SourcePatch itself rolls back atomically: a single failed edit
//      returns the original content with ErrSourceNotExports.
//   4. RedactDiff — secret-shaped keys are collapsed to [redacted]
//      before the diff reaches the operator-facing preview or the
//      audit log.
//
// The contract called out by the issue:
//
//   - TestAdvancedPatchPreservesUnmanagedCode: an unrelated preset
//     apply leaves operator-owned code byte-stable.
//   - FunctionGlobalContextAndNodeDefaultsFixtureSuite: supported
//     structured cases round-trip; executable/unknown cases stay
//     preserved.

package service

import (
	"errors"
	"fmt"
)

// ErrUnknownPreset is returned by ApplyPreset when the requested preset
// ID is not in the registry. Callers should surface it as a 404-style
// "preset not available" rather than as a generic 500; the registry
// being defensive about unknown IDs is part of the contract.
var ErrUnknownPreset = errors.New("preset not registered")

// PresetApplyResult is the outcome of an ApplyPreset call. Before and
// After are the raw source (Before is the input the caller passed;
// After is the source after a successful apply). Preview is the
// redacted unified diff suitable for operator-facing display and for
// inclusion in the audit log; it never contains secret-shaped values.
//
// On failure Before == After and the error describes why. Slice 3 will
// render Preview in the AdvancedSettings UI; the apply handler in
// internal/handler will route Preview into the audit meta.
type PresetApplyResult struct {
	Before   string
	After    string
	Preview  string
	Inserted []string
	Replaced []string
}

// ApplyPreset runs the curated preset pipeline on content. The caller
// supplies the current settings.js source and the preset ID; values is
// a per-preset input map (slice 3 will populate it from the UI form;
// slice 2 ships a no-op values map because slice 1's BuildEdits are
// deterministic placeholders).
//
// On success: After is the patched source, Preview is the redacted
// diff, Inserted/Replaced describe which managed keys were touched.
// On failure (unknown preset, BuildEdits error, SourcePatch error):
// Before == After == the original content, and the returned error
// explains why. The "rollback" semantics are baked into SourcePatch —
// a single failed edit returns the original content verbatim — and
// PresetApply surfaces that to the caller without further mutation.
func (r *PresetRegistry) ApplyPreset(content, presetID string, values map[string]string) (PresetApplyResult, error) {
	// Defensive default so a nil-registry caller doesn't panic. The
	// happy path always uses a real registry from NewPresetRegistry.
	if r == nil {
		return PresetApplyResult{Before: content, After: content},
			fmt.Errorf("%w: nil registry", ErrUnknownPreset)
	}

	// Step 1 — registry lookup. An unknown ID is a 404, not a 500.
	preset, ok := r.Get(presetID)
	if !ok {
		return PresetApplyResult{Before: content, After: content},
			fmt.Errorf("%w: %q", ErrUnknownPreset, presetID)
	}

	// Step 2 — BuildEdits. The closure is required to be pure by the
	// slice 1 contract; a non-deterministic edit would break the
	// preview/audit reconciliation in slice 3.
	edits, err := preset.BuildEdits(content, values)
	if err != nil {
		return PresetApplyResult{Before: content, After: content},
			fmt.Errorf("build edits for %q: %w", presetID, err)
	}

	// Step 3 — SourcePatch. This is where the byte-stable unmanaged-
	// region guarantee lives. A failed edit here returns the original
	// content with ErrSourceNotExports; we propagate both.
	patched, err := SourcePatch(content, edits)
	if err != nil {
		// SourcePatch already returns the original content inside
		// patched.Content on failure. Mirror it into After so the
		// caller sees the rolled-back source verbatim.
		return PresetApplyResult{Before: content, After: patched.Content},
			fmt.Errorf("apply preset %q: %w", presetID, err)
	}

	// Step 4 — RedactDiff produces the operator-facing preview. The
	// redaction policy (apply_diff.go) is the same policy that audit
	// logs use, so an operator who sees the preview never sees
	// credentials they would not otherwise be allowed to see, and an
	// auditor who reads the log sees the same surface.
	preview := RedactDiff(content, patched.Content)

	return PresetApplyResult{
		Before:   content,
		After:    patched.Content,
		Preview:  preview,
		Inserted: patched.Inserted,
		Replaced: patched.Replaced,
	}, nil
}
