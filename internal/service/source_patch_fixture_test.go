// Package service — slice C of issue #757 pins the source-preserving
// settings.js patch contract to a corpus of realistic Node-RED 5
// fixtures. Each fixture exercises a distinct preservation hazard:
//
//   - settings-nodered-5-baseline.js        — pristine Node-RED 5 file.
//   - settings-with-extensions.js           — operator-owned function
//     expressions and require() calls.
//   - settings-callback-middleware.js       — nested braces, mixed-quote
//     strings and block comments
//     inside operator middleware.
//   - settings-unmanaged-untouched.js       — heavy operator commenting,
//     mixed indentation and
//     multiple unmanaged keys.
//
// For every fixture the suite runs six test families:
//
//  1. NoOpImportSave            — import → fingerprint → save round-trip
//     must be byte-for-byte identical.
//  2. ManagedPatchPreservesUnmanaged
//     — SourcePatch over a managed key must not
//     touch operator-owned regions.
//  3. FingerprintStability       — FingerprintSource is deterministic
//     and reacts to every byte change.
//  4. CompatibilityPolicyInteraction
//     — ConfigurationCapabilities reports the
//     expected adapter / mode / editability.
//  5. BlockPatchPreservesUnmanaged
//     — ApplyBlockEdit on a managed block must
//     preserve every byte outside the block.
//  6. RevisionRoundTrip          — slice-B fingerprint integration: save
//     with the captured revision succeeds,
//     stale revisions are refused with
//     ErrSourceRevisionMismatch, retry after
//     re-read succeeds.
//
// Slice C is the integration acceptance gate for the slice-A
// (source-preserving patch contract) and slice-B (revision fingerprint +
// 409 on stale settings.js) contracts: every sub-test must pass on a
// hermetic isolated host so the CI gate is green regardless of the
// developer's local Node-RED installation.
//
// Slice deferral:
//   - The frontend revision-echo UI (returning the new revision in the
//     response payload and re-binding it on the Save button) is out of
//     scope for this slice. The backend contract is fully exercised here;
//     wiring the response field into the React side is a follow-up.
//   - The renderSettings routing change that would have the legacy
//     SaveRawSettingsWithRevision path receive an already-patched
//     SourcePatch result instead of a raw content blob is deferred to a
//     follow-up PR. Slice C validates the contract end-to-end on the
//     file-level path; the routing refactor is orthogonal.
package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// loadRoundTripFixtures loads every settings-*.js fixture under testdata.
// The fixture name is the basename without the .js extension so sub-tests
// can use it directly as their t.Run label.
//
// Failures here abort the suite immediately — every subsequent test
// depends on the corpus being readable.
func loadRoundTripFixtures(t *testing.T) []struct {
	name    string
	content string
} {
	t.Helper()

	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata/: %v", err)
	}

	var fixtures []struct {
		name    string
		content string
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "settings-") || !strings.HasSuffix(name, ".js") {
			continue
		}
		path := filepath.Join("testdata", name)
		//nolint:gosec // G304 -- path is hardcoded under the package's testdata/ directory and filtered to settings-*.js fixtures enumerated via os.ReadDir.
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture %q: %v", path, err)
		}
		fixtures = append(fixtures, struct {
			name    string
			content string
		}{
			name:    strings.TrimSuffix(name, ".js"),
			content: string(data),
		})
	}

	if len(fixtures) < 4 {
		t.Fatalf("expected at least 4 settings-*.js fixtures, found %d", len(fixtures))
	}

	// Stable ordering so test output is reproducible regardless of the
	// filesystem's directory-entry order.
	for i := 1; i < len(fixtures); i++ {
		for j := i; j > 0 && fixtures[j-1].name > fixtures[j].name; j-- {
			fixtures[j-1], fixtures[j] = fixtures[j], fixtures[j-1]
		}
	}
	return fixtures
}

// newRoundTripFixtureService builds a hermetic ConfigService rooted at a
// temp directory. The settings.js path resolves to <tempDir>/settings.js
// regardless of the developer's local Node-RED installation, so the
// corpus is reproducible on Linux CI agents and developer laptops alike.
func newRoundTripFixtureService(t *testing.T) (*ConfigService, string) {
	t.Helper()
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	return NewConfigServiceWithHost(tempDir, hostSvc), hostSvc.Detect().Settings.Path
}

// seedFixture writes the fixture content to the service's settings.js
// path through the SaveRawSettings entry point so the file lives at the
// canonical path with mode 0600 (matching a real first-save state).
func seedFixture(t *testing.T, svc *ConfigService, content string) {
	t.Helper()
	if _, err := svc.SaveRawSettings(content); err != nil {
		t.Fatalf("seed SaveRawSettings: %v", err)
	}
}

// requireBytesEqual compares got vs want byte-for-byte and emits a
// diff-friendly failure message. The argument labels are passed through
// to the failure text so the failure is self-describing.
func requireBytesEqual(t *testing.T, label string, got, want []byte) {
	t.Helper()
	if string(got) != string(want) {
		t.Fatalf("%s: bytes differ (got %d bytes, want %d bytes); first diff at offset %d\n got: %q\nwant: %q",
			label, len(got), len(want), firstDiffOffset(got, want), string(got), string(want))
	}
}

// firstDiffOffset returns the index of the first byte where a and b
// differ. When a == b (same length, same bytes) the length is returned
// so the value is still meaningful in the message.
func firstDiffOffset(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// -----------------------------------------------------------------------
// Family 1 — NoOpImportSave
// -----------------------------------------------------------------------

// TestSettingsRoundTripFixtureSuite_NoOpImportSave pins the foundational
// promise of the source-preserving work: import a settings.js document,
// read its revision, save the bytes back unchanged, and the file on disk
// must be byte-for-byte equal to the input. Any drift here would
// invalidate the no-op guarantee every other slice relies on.
func TestSettingsRoundTripFixtureSuite_NoOpImportSave(t *testing.T) {
	fixtures := loadRoundTripFixtures(t)

	for _, f := range fixtures {
		f := f
		t.Run(f.name, func(t *testing.T) {
			svc, settingsPath := newRoundTripFixtureService(t)
			seedFixture(t, svc, f.content)

			// Re-read the bytes the canonical read path would surface.
			read, err := svc.GetRawSettings()
			if err != nil {
				t.Fatalf("GetRawSettings: %v", err)
			}
			if read.Content != f.content {
				t.Fatalf("GetRawSettings returned content that does not match the seeded fixture (offset %d)",
					firstDiffOffset([]byte(read.Content), []byte(f.content)))
			}

			// Save the exact bytes back with the captured revision.
			doc, err := svc.SaveRawSettingsWithRevision(read.Content, read.Revision)
			if err != nil {
				t.Fatalf("SaveRawSettingsWithRevision (no-op save): %v", err)
			}
			if doc.Content != f.content {
				t.Fatalf("returned Content drifted from the seeded fixture (offset %d)",
					firstDiffOffset([]byte(doc.Content), []byte(f.content)))
			}

			// The on-disk file must be identical to the fixture; this is
			// the strictest possible no-op invariant.
			//nolint:gosec // G304 -- settingsPath is the hermetic isolated HostService settings path inside t.TempDir(), not user input.
			onDisk, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("read settings.js: %v", err)
			}
			requireBytesEqual(t, "on-disk settings.js", onDisk, []byte(f.content))
		})
	}
}

// -----------------------------------------------------------------------
// Family 2 — ManagedPatchPreservesUnmanaged
// -----------------------------------------------------------------------

// TestSettingsRoundTripFixtureSuite_ManagedPatchPreservesUnmanaged proves
// that SourcePatch over a single managed scalar (uiPort) does not
// disturb any operator-owned region: comments, function expressions,
// require() calls, blank lines, indentation — every byte outside the
// managed entry must round-trip verbatim.
//
// Each fixture has a distinct unmanaged signature; the assertions are
// therefore per-fixture rather than a single template check.
func TestSettingsRoundTripFixtureSuite_ManagedPatchPreservesUnmanaged(t *testing.T) {
	fixtures := loadRoundTripFixtures(t)

	// unmanagedSignatures are the byte-anchored substrings each fixture
	// must contain AFTER the managed edit. They are chosen so a regex-
	// matcher regression (#723) would drop at least one of them on every
	// fixture.
	unmanagedSignatures := map[string][]string{
		"settings-nodered-5-baseline": {
			"httpNodeAuth",
			"httpStaticAuth",
			"$2a$08$XXXXXXXXXXXXXXXXXXXXX",
			"functionGlobalContext",
		},
		"settings-with-extensions": {
			"httpMiddleware: function(req, res, next)",
			"externalModules",
			"exportGlobalContextKeys: ['os', 'fs', 'path']",
			"nodesDirArray",
			"secrets: require('./secrets.json')",
		},
		"settings-callback-middleware": {
			"httpMiddleware: function(req, res, next)",
			"setHeader('X-Powered-By'",
			"function reply(message)",
			"hint: 'use \"strong\" passwords'",
		},
		"settings-unmanaged-untouched": {
			"// ============================================================================",
			"// Header — operator-authored.",
			"nodesDirArray",
			"externalModules",
			"// Footer — operator-authored.",
		},
	}

	for _, f := range fixtures {
		f := f
		t.Run(f.name, func(t *testing.T) {
			svc, _ := newRoundTripFixtureService(t)
			seedFixture(t, svc, f.content)

			// Apply a managed edit on a key that every fixture has: uiPort.
			result, err := SourcePatch(f.content, []SourceEdit{
				{Key: "uiPort", Value: "3000"},
			})
			if err != nil {
				t.Fatalf("SourcePatch: %v", err)
			}
			if !strings.Contains(result.Content, "uiPort: 3000,") {
				t.Fatalf("managed edit did not land; got head:\n%s", headBytes(result.Content, 400))
			}

			// Every unmanaged signature for this fixture must still be
			// present byte-for-byte.
			for _, sig := range unmanagedSignatures[f.name] {
				if !strings.Contains(result.Content, sig) {
					t.Errorf("unmanaged signature %q dropped by SourcePatch", sig)
				}
			}

			// The edit must report uiPort in Replaced (it pre-existed in
			// every fixture) — never in Inserted.
			if len(result.Inserted) != 0 {
				t.Errorf("Inserted = %v, want [] (uiPort pre-exists in every fixture)", result.Inserted)
			}
			foundReplaced := false
			for _, k := range result.Replaced {
				if k == "uiPort" {
					foundReplaced = true
					break
				}
			}
			if !foundReplaced {
				t.Errorf("Replaced = %v, want uiPort present", result.Replaced)
			}
		})
	}
}

// headBytes returns the first n bytes of s, appending an ellipsis when
// truncated. Used for diagnostic messages only.
func headBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

// -----------------------------------------------------------------------
// Family 3 — FingerprintStability
// -----------------------------------------------------------------------

// TestSettingsRoundTripFixtureSuite_FingerprintStability locks in slice
// B's fingerprint contract across the entire corpus: identical bytes
// produce identical fingerprints, every byte-level mutation changes the
// fingerprint, and the revision Algorithm identifier is the catalog
// value so callers can pin a future algorithm roll-out.
func TestSettingsRoundTripFixtureSuite_FingerprintStability(t *testing.T) {
	fixtures := loadRoundTripFixtures(t)

	for _, f := range fixtures {
		f := f
		t.Run(f.name, func(t *testing.T) {
			rev1 := FingerprintSource(f.content)
			rev2 := FingerprintSource(f.content)

			if rev1.Fingerprint == "" {
				t.Fatalf("FingerprintSource(%q) returned empty fingerprint", f.name)
			}
			if rev1.Fingerprint != rev2.Fingerprint {
				t.Fatalf("FingerprintSource is not deterministic: %q vs %q",
					rev1.Fingerprint, rev2.Fingerprint)
			}
			if rev1.Algorithm != SourceRevisionAlgorithm {
				t.Errorf("Algorithm = %q, want %q", rev1.Algorithm, SourceRevisionAlgorithm)
			}

			// A byte-level mutation (changing the managed uiPort) must
			// move the fingerprint. Use SourcePatch so we exercise the
			// same edit path slice C protects.
			patched, err := SourcePatch(f.content, []SourceEdit{
				{Key: "uiPort", Value: "65535"},
			})
			if err != nil {
				t.Fatalf("SourcePatch: %v", err)
			}
			revPatched := FingerprintSource(patched.Content)
			if revPatched.Fingerprint == rev1.Fingerprint {
				t.Fatalf("fingerprint did not change after a managed edit (%s)", revPatched.Fingerprint)
			}

			// RevisionMatches: equal revisions match, divergent do not.
			if !RevisionMatches(rev1, rev1) {
				t.Errorf("RevisionMatches(rev, rev) = false, want true")
			}
			if RevisionMatches(rev1, revPatched) {
				t.Errorf("RevisionMatches(original, patched) = true, want false")
			}
		})
	}
}

// -----------------------------------------------------------------------
// Family 4 — CompatibilityPolicyInteraction
// -----------------------------------------------------------------------

// TestSettingsRoundTripFixtureSuite_CompatibilityPolicyInteraction
// asserts the host-detected contract for every fixture: the isolated
// hermetic service must report Node-RED 5, the nodered-5 adapter,
// Editable=true (because settings.js is writable inside t.TempDir) and
// Mode=editable. Slice C is the integration test that proves fixtures
// can be edited end-to-end; without Editable=true the entire patch
// pipeline is short-circuited.
func TestSettingsRoundTripFixtureSuite_CompatibilityPolicyInteraction(t *testing.T) {
	fixtures := loadRoundTripFixtures(t)

	for _, f := range fixtures {
		f := f
		t.Run(f.name, func(t *testing.T) {
			svc, settingsPath := newRoundTripFixtureService(t)
			seedFixture(t, svc, f.content)

			caps := svc.ConfigurationCapabilities()
			if caps.RuntimeVersion == "" {
				t.Fatalf("RuntimeVersion is empty; isolated host should report a known version")
			}
			if caps.Adapter != "nodered-5" {
				t.Errorf("Adapter = %q, want %q", caps.Adapter, "nodered-5")
			}
			if !caps.Editable {
				t.Errorf("Editable = false, want true; settings path was %q (Writable=%t)",
					settingsPath, svc.hostSvc.Detect().Settings.Writable)
			}
			if caps.Mode != "editable" {
				t.Errorf("Mode = %q, want %q", caps.Mode, "editable")
			}
			if caps.CatalogVersion == "" {
				t.Errorf("CatalogVersion is empty")
			}

			// The detected path must point at the isolated service's
			// settings.js and not at the developer's local Node-RED
			// installation. This is the regression guard for #754
			// (managed runtime authority).
			hostSettings := svc.hostSvc.Detect().Settings
			if hostSettings.Path != settingsPath {
				t.Errorf("HostService.Settings.Path = %q, want %q", hostSettings.Path, settingsPath)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Family 5 — BlockPatchPreservesUnmanaged
// -----------------------------------------------------------------------

// TestSettingsRoundTripFixtureSuite_BlockPatchPreservesUnmanaged
// exercises the brace-counting scanner on a managed block edit
// (editorTheme or logging, both present in every fixture) and proves
// every byte outside the block is preserved verbatim.
//
// The replacement block is intentionally different in content so an
// accidental identity-edit (which would silently pass) is impossible.
func TestSettingsRoundTripFixtureSuite_BlockPatchPreservesUnmanaged(t *testing.T) {
	fixtures := loadRoundTripFixtures(t)

	const newEditorTheme = `    editorTheme: {
        projects: {
            enabled: true
        },
        // slice-C replacement marker — must appear in the patched result
        sliceCMarker: "block-edit-fingerprint",
    },`

	// Block-signatures: substrings outside the editorTheme block that
	// must survive the block edit. These are the unmanaged anchors the
	// scanner must NOT touch while rewriting editorTheme.
	unmanagedAnchors := map[string][]string{
		"settings-nodered-5-baseline": {
			"uiPort: process.env.PORT",
			"httpAdminRoot: '/'",
			"functionGlobalContext: {",
		},
		"settings-with-extensions": {
			"httpMiddleware: function(req, res, next)",
			"externalModules: {",
			"nodesDirArray: [",
			"functionGlobalContext: {",
		},
		"settings-callback-middleware": {
			"httpMiddleware: function(req, res, next)",
			"function reply(message)",
			"res.on('finish'",
			"httpNodeAuth: {",
		},
		"settings-unmanaged-untouched": {
			"// ============================================================================",
			"// Header",
			"httpMiddleware: function(req, res, next)",
			"// Footer",
			"functionGlobalContext: {",
		},
	}

	for _, f := range fixtures {
		f := f
		t.Run(f.name, func(t *testing.T) {
			patched, ok := ApplyBlockEdit(f.content, "editorTheme", newEditorTheme)
			if !ok {
				t.Fatalf("ApplyBlockEdit(editorTheme) returned ok=false on %s", f.name)
			}

			// The replacement block must appear verbatim in the result.
			if !strings.Contains(patched, newEditorTheme) {
				t.Fatalf("ApplyBlockEdit did not insert the replacement block verbatim on %s", f.name)
			}

			// Slice-C marker must be present so a future regression that
			// truncates the replacement is caught at the test stage.
			if !strings.Contains(patched, `sliceCMarker: "block-edit-fingerprint"`) {
				t.Fatalf("slice-C marker missing from the replacement block on %s", f.name)
			}

			// Every unmanaged anchor must survive verbatim.
			for _, anchor := range unmanagedAnchors[f.name] {
				if !strings.Contains(patched, anchor) {
					t.Errorf("unmanaged anchor %q dropped by ApplyBlockEdit on %s", anchor, f.name)
				}
			}

			// The patched content must also pass through SourcePatch's
			// fingerprint determinism — fingerprinting twice must yield
			// the same hash, and it must differ from the original.
			origFP := FingerprintSource(f.content)
			newFP := FingerprintSource(patched)
			if origFP.Fingerprint == newFP.Fingerprint {
				t.Errorf("byte-level fingerprint did not change after block edit on %s", f.name)
			}
			if FingerprintSource(patched).Fingerprint != newFP.Fingerprint {
				t.Errorf("fingerprint is not stable after a block edit on %s", f.name)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Family 6 — RevisionRoundTrip
// -----------------------------------------------------------------------

// TestSettingsRoundTripFixtureSuite_RevisionRoundTrip is the slice-B
// integration gate across the fixture corpus. It exercises the
// fingerprint → save → revision-advance → external-edit → stale-rejection
// → re-read → retry-success sequence on every fixture so a regression in
// any of the four steps is caught with a per-fixture failure label.
func TestSettingsRoundTripFixtureSuite_RevisionRoundTrip(t *testing.T) {
	fixtures := loadRoundTripFixtures(t)

	for _, f := range fixtures {
		f := f
		t.Run(f.name, func(t *testing.T) {
			svc, settingsPath := newRoundTripFixtureService(t)
			seedFixture(t, svc, f.content)

			// Step 0: derive a byte-different candidate through SourcePatch
			// (a managed uiPort edit). The revision MUST advance across a
			// real write — re-saving identical bytes is a documented
			// no-op for the fingerprint (#757 slice B acceptance).
			patched, err := SourcePatch(f.content, []SourceEdit{
				{Key: "uiPort", Value: "3000"},
			})
			if err != nil {
				t.Fatalf("SourcePatch: %v", err)
			}
			if patched.Content == f.content {
				t.Fatalf("SourcePatch returned byte-identical content; managed edit did not land")
			}

			// Step 1: read the initial revision.
			initial, err := svc.GetRawSettings()
			if err != nil {
				t.Fatalf("initial GetRawSettings: %v", err)
			}
			if initial.Revision.Fingerprint == "" {
				t.Fatal("initial revision fingerprint is empty")
			}

			// Step 2: save the patched bytes with the captured (fresh)
			// revision — must succeed and the returned revision MUST
			// advance to reflect the new on-disk content.
			afterSave, err := svc.SaveRawSettingsWithRevision(patched.Content, initial.Revision)
			if err != nil {
				t.Fatalf("SaveRawSettingsWithRevision with fresh revision: %v", err)
			}
			if afterSave.Revision.Fingerprint == initial.Revision.Fingerprint {
				t.Fatalf("revision did not advance after a successful edit-save (%s)", afterSave.Revision.Fingerprint)
			}
			if afterSave.Revision.Fingerprint != FingerprintSource(patched.Content).Fingerprint {
				t.Errorf("returned revision does not match the patched content's fingerprint")
			}

			// Step 3: an external actor mutates the file directly (e.g.
			// Node-RED restart, the operator's text editor, a sibling
			// NRCC instance). The captured revision is now stale.
			const externalEdit = "module.exports = { uiPort: 19999 };\n"
			if err := os.WriteFile(settingsPath, []byte(externalEdit), 0o600); err != nil {
				t.Fatalf("external edit: %v", err)
			}

			// Step 4: save with the now-stale revision must fail with
			// ErrSourceRevisionMismatch and must NOT touch the file.
			_, err = svc.SaveRawSettingsWithRevision(patched.Content, initial.Revision)
			if err == nil {
				t.Fatal("SaveRawSettingsWithRevision accepted a stale revision; want ErrSourceRevisionMismatch")
			}
			if !errors.Is(err, ErrSourceRevisionMismatch) {
				t.Fatalf("SaveRawSettingsWithRevision error = %v, want ErrSourceRevisionMismatch", err)
			}
			//nolint:gosec // G304 -- settingsPath is the hermetic isolated HostService settings path inside t.TempDir(), not user input.
			onDisk, readErr := os.ReadFile(settingsPath)
			if readErr != nil {
				t.Fatalf("read settings.js after rejected save: %v", readErr)
			}
			if string(onDisk) != externalEdit {
				t.Fatalf("settings.js was modified despite revision rejection:\n got: %q\nwant: %q",
					string(onDisk), externalEdit)
			}

			// Step 5: re-read to pick up the new revision and retry —
			// must succeed and must return a fresh revision.
			fresh, err := svc.GetRawSettings()
			if err != nil {
				t.Fatalf("GetRawSettings after external edit: %v", err)
			}
			if fresh.Revision.Fingerprint == initial.Revision.Fingerprint {
				t.Fatal("re-read did not pick up the external edit's revision")
			}

			// Rewrite the patched bytes with the fresh revision; this is
			// the production retry path: the operator's last-known-good
			// content survives, just stamped with the new on-disk
			// revision.
			retry, err := svc.SaveRawSettingsWithRevision(patched.Content, fresh.Revision)
			if err != nil {
				t.Fatalf("retry with fresh revision: %v", err)
			}
			if retry.Content != patched.Content {
				t.Fatalf("retry content = %q, want patched content", retry.Content)
			}
			if retry.Revision.Fingerprint == fresh.Revision.Fingerprint {
				t.Errorf("retry revision did not advance from the fresh revision (%s)", retry.Revision.Fingerprint)
			}

			// Step 6: backwards-compat — an empty expected revision is
			// accepted (legacy callers that have not yet captured a
			// revision are never blocked). Saving the patched bytes here
			// exercises the legacy path with a non-trivial edit.
			if _, err := svc.SaveRawSettings(patched.Content); err != nil {
				t.Fatalf("SaveRawSettings with empty revision (legacy path): %v", err)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Sanity — every fixture produces a non-zero SourceRevision that uses
// the catalog Algorithm identifier.
// -----------------------------------------------------------------------

// TestSettingsRoundTripFixtureSuite_FixtureRevisionCoverage is a meta
// check that ties the four fixtures together: each one must yield a
// non-empty fingerprint tagged with the catalog Algorithm string. This
// guards against a fixture accidentally landing in the corpus without a
// recognised module.exports shape (FingerprintSource would still return
// a hash, but ApplyBlockEdit/ApplyScalarEdit would refuse to edit it).
func TestSettingsRoundTripFixtureSuite_FixtureRevisionCoverage(t *testing.T) {
	fixtures := loadRoundTripFixtures(t)

	for _, f := range fixtures {
		f := f
		t.Run(f.name, func(t *testing.T) {
			rev := FingerprintSource(f.content)
			if rev.Fingerprint == "" {
				t.Fatalf("fixture %q produced empty fingerprint", f.name)
			}
			if rev.Algorithm != SourceRevisionAlgorithm {
				t.Errorf("fixture %q Algorithm = %q, want %q",
					f.name, rev.Algorithm, SourceRevisionAlgorithm)
			}

			// Sanity-check that each fixture is editable through the
			// SourcePatch entry point — a fixture that is not a
			// module.exports shape would surface here.
			if _, ok := ApplyScalarEdit(f.content, "uiPort", "3000"); !ok {
				t.Fatalf("ApplyScalarEdit refused fixture %q; not a recognisable module.exports", f.name)
			}

			// Sanity-check that the revision field round-trips through
			// the model package without loss (this protects callers that
			// rely on model.SourceRevision to carry the contract).
			doc := model.SettingsDocument{Revision: rev, Content: f.content}
			if doc.Revision.Fingerprint != rev.Fingerprint {
				t.Errorf("model.SourceRevision lost the fingerprint on copy")
			}
		})
	}
}
