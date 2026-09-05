package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// helpers --------------------------------------------------------------------

// writeFileDirectly simulates an external actor (operator's editor, Node-RED
// itself, a second NRCC instance) mutating settings.js without going through
// the ConfigService. Tests use it to manufacture a stale-projection scenario:
// the caller has revision R captured from an earlier read, but the file on
// disk has moved on.
func writeFileDirectly(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir for direct write: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("direct write: %v", err)
	}
}

// Fingerprint ----------------------------------------------------------------

// TestFingerprintSource_Deterministic locks in the contract that identical
// bytes always produce identical revisions, and that the revision carries
// the algorithm identifier callers need to verify they are comparing like
// for like.
func TestFingerprintSource_Deterministic(t *testing.T) {
	const content = "module.exports = { uiPort: 1880 };\n"

	rev1 := FingerprintSource(content)
	rev2 := FingerprintSource(content)

	if rev1.Fingerprint == "" {
		t.Fatal("fingerprint must not be empty for non-empty content")
	}
	if rev1.Fingerprint != rev2.Fingerprint {
		t.Errorf("same bytes produced different fingerprints: %q vs %q", rev1.Fingerprint, rev2.Fingerprint)
	}
	if rev1.Algorithm != SourceRevisionAlgorithm {
		t.Errorf("algorithm = %q, want %q", rev1.Algorithm, SourceRevisionAlgorithm)
	}
}

// TestFingerprintSource_DetectsByteChange guards against a regression where
// the fingerprint accidentally normalises whitespace, comments or trailing
// newlines and therefore cannot tell two revisions apart. Every byte must
// participate in the digest.
func TestFingerprintSource_DetectsByteChange(t *testing.T) {
	base := FingerprintSource("module.exports = { uiPort: 1880 };\n")

	mutations := []struct {
		name string
		mut  func(string) string
	}{
		{"portChanged", func(s string) string { return strings.Replace(s, "1880", "1881", 1) }},
		{"newlineAdded", func(s string) string { return s + "\n" }},
		{"spaceInsideBlock", func(s string) string { return strings.Replace(s, " { ", "{\n  ", 1) }},
		{"trailingSemicolonRemoved", func(s string) string { return strings.TrimRight(s, ";\n") }},
		{"commentAppended", func(s string) string { return s + "// operator note\n" }},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			got := FingerprintSource(m.mut("module.exports = { uiPort: 1880 };\n"))
			if got.Fingerprint == base.Fingerprint {
				t.Errorf("mutation %q did not change fingerprint (%s)", m.name, got.Fingerprint)
			}
		})
	}
}

// TestFingerprintSource_EmptyContent documents that an empty content
// still has a stable, non-empty fingerprint so a never-touched file has a
// revision callers can pin against.
func TestFingerprintSource_EmptyContent(t *testing.T) {
	rev := FingerprintSource("")
	if rev.Fingerprint == "" {
		t.Fatal("empty content must still produce a fingerprint")
	}
	if FingerprintSource("").Fingerprint != rev.Fingerprint {
		t.Errorf("empty content fingerprints are not stable across calls")
	}
	if FingerprintSource("\n").Fingerprint == rev.Fingerprint {
		t.Errorf("a single newline must not collide with the empty content fingerprint")
	}
}

// RevisionMatches ------------------------------------------------------------

// TestRevisionMatches_EmptyExpectedAlwaysPasses documents the backward-
// compatibility contract: callers that have not yet captured a revision
// (legacy SaveRawSettings path) are never blocked.
func TestRevisionMatches_EmptyExpectedAlwaysPasses(t *testing.T) {
	current := FingerprintSource("module.exports = { uiPort: 1880 };\n")
	cases := []model.SourceRevision{
		{},                                              // zero value
		{Fingerprint: ""},                               // explicit empty fingerprint
		{Fingerprint: "", Algorithm: SourceRevisionAlgorithm},
	}
	for _, expected := range cases {
		if !RevisionMatches(expected, current) {
			t.Errorf("RevisionMatches(%+v, current) = false, want true (backward compat)", expected)
		}
	}
}

// TestRevisionMatches_Match verifies the happy path: identical fingerprints
// are accepted regardless of CapturedAt because equality is determined by
// (Algorithm, Fingerprint), not by capture time.
func TestRevisionMatches_Match(t *testing.T) {
	current := FingerprintSource("module.exports = { uiPort: 1880 };\n")
	expected := current
	expected.CapturedAt = "1970-01-01T00:00:00Z" // far in the past on purpose
	if !RevisionMatches(expected, current) {
		t.Errorf("RevisionMatches identical fingerprints = false, want true")
	}
}

// TestRevisionMatches_Mismatch verifies a different fingerprint is rejected
// — the core conflict-detection behaviour that drives the 409 response.
func TestRevisionMatches_Mismatch(t *testing.T) {
	current := FingerprintSource("module.exports = { uiPort: 1880 };\n")
	expected := FingerprintSource("module.exports = { uiPort: 1881 };\n")
	if RevisionMatches(expected, current) {
		t.Errorf("RevisionMatches different fingerprints = true, want false")
	}
}

// TestRevisionMatches_AlgorithmMismatch protects against a future algorithm
// roll-out producing accidental fingerprint collisions: two revisions that
// agree on the hex digest but disagree on algorithm must not be treated
// as equal.
func TestRevisionMatches_AlgorithmMismatch(t *testing.T) {
	current := model.SourceRevision{Fingerprint: "abc123", Algorithm: "sha256"}
	expected := model.SourceRevision{Fingerprint: "abc123", Algorithm: "blake2b"}
	if RevisionMatches(expected, current) {
		t.Errorf("RevisionMatches across algorithms = true, want false")
	}
}

// ConfigService integration --------------------------------------------------

// TestSaveRawSettingsWithRevision_FreshRevisionAccepted is the happy path:
// the caller captured the current revision before composing their edit, the
// save succeeds and the returned document carries the next revision so the
// client can pin a subsequent save against it without re-reading.
func TestSaveRawSettingsWithRevision_FreshRevisionAccepted(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	read, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}
	if read.Revision.Fingerprint == "" {
		t.Fatal("GetRawSettings must populate Revision even on first read")
	}

	const updated = "module.exports = { uiPort: 1880, httpAdminRoot: '/' };\n"
	doc, err := svc.SaveRawSettingsWithRevision(updated, read.Revision)
	if err != nil {
		t.Fatalf("SaveRawSettingsWithRevision: %v", err)
	}
	if doc.Revision.Fingerprint == "" {
		t.Fatal("returned document must carry the next revision")
	}
	if doc.Revision.Fingerprint == read.Revision.Fingerprint {
		t.Errorf("revision did not advance after a real edit: %s", doc.Revision.Fingerprint)
	}
	if doc.Content != updated {
		t.Errorf("Content = %q, want %q", doc.Content, updated)
	}
}

// TestSaveRawSettingsWithRevision_NoExpectedAccepted keeps the legacy
// SaveRawSettings signature's promise that callers without a captured
// revision are not blocked. The handler routes the empty-revision case
// through this same code path on every request until the frontend learns
// to echo the revision back.
func TestSaveRawSettingsWithRevision_NoExpectedAccepted(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	const content = "module.exports = { uiPort: 1880 };\n"
	doc, err := svc.SaveRawSettingsWithRevision(content, model.SourceRevision{})
	if err != nil {
		t.Fatalf("SaveRawSettingsWithRevision: %v", err)
	}
	if doc.Content != content {
		t.Errorf("Content = %q, want %q", doc.Content, content)
	}
}

// TestStaleProjectionRejected is the acceptance test for slice B of #757:
// when settings.js is mutated externally between the operator's read and
// their save, the save must be rejected with ErrSourceRevisionMismatch
// and the on-disk file must remain untouched (no backup is created either,
// so a follow-up retry against the new revision does not see a phantom
// "previous version").
func TestStaleProjectionRejected(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	paths := hostSvc.Detect().Settings
	if paths.Path == "" {
		t.Fatal("isolated host service did not resolve a settings.js path")
	}

	// 1. Initial write so a settings.js exists and has a stable revision.
	const initial = "module.exports = { uiPort: 1880 };\n"
	if _, err := svc.SaveRawSettings(initial); err != nil {
		t.Fatalf("initial SaveRawSettings: %v", err)
	}

	// 2. Operator reads → captures revision R0.
	read, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}
	if read.Revision.Fingerprint == "" {
		t.Fatal("GetRawSettings must populate Revision on subsequent reads")
	}
	expected := read.Revision

	// 3. External actor mutates the file directly (Node-RED restart, the
	//    operator's text editor, a sibling NRCC instance, …). The file is
	//    now at revision R1 != R0 even though the caller's expected
	//    revision is still R0.
	const externallyEdited = "module.exports = { uiPort: 19999 };\n"
	writeFileDirectly(t, paths.Path, externallyEdited)

	// 4. Operator tries to save with their captured (now stale) revision.
	const candidate = "module.exports = { uiPort: 2000 };\n"
	_, err = svc.SaveRawSettingsWithRevision(candidate, expected)
	if err == nil {
		t.Fatal("SaveRawSettingsWithRevision accepted a stale revision; want ErrSourceRevisionMismatch")
	}
	if !errors.Is(err, ErrSourceRevisionMismatch) {
		t.Errorf("error = %v, want ErrSourceRevisionMismatch", err)
	}

	// 5. The external edit MUST survive untouched — the rejected save must
	//    not overwrite it and must not create a backup file (creating one
	//    would imply the previous version was the candidate, not the
	//    external edit, and would mislead a subsequent retry).
	got, readErr := os.ReadFile(paths.Path)
	if readErr != nil {
		t.Fatalf("read settings.js after rejected save: %v", readErr)
	}
	if string(got) != externallyEdited {
		t.Errorf("settings.js was modified despite rejection:\n got: %q\nwant: %q", string(got), externallyEdited)
	}
	backupGlob := filepath.Join(tempDir, "backups", "settings", "*.js.bak")
	matches, _ := filepath.Glob(backupGlob)
	if len(matches) != 0 {
		t.Errorf("rejected save created backup(s) %v; expected none", matches)
	}
}

// TestStaleProjectionRejected_RetrySucceedsAfterRevalidation is the
// companion to TestStaleProjectionRejected: once the caller re-reads and
// re-renders against the new revision, the next save must succeed and the
// returned document must carry the next revision so the client keeps
// pinning correctly.
func TestStaleProjectionRejected_RetrySucceedsAfterRevalidation(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	if _, err := svc.SaveRawSettings("module.exports = { uiPort: 1880 };\n"); err != nil {
		t.Fatalf("initial save: %v", err)
	}
	stale, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}

	settingsPath := hostSvc.Detect().Settings.Path
	writeFileDirectly(t, settingsPath, "module.exports = { uiPort: 19999 };\n")

	if _, err := svc.SaveRawSettingsWithRevision("module.exports = { uiPort: 1881 };\n", stale.Revision); !errors.Is(err, ErrSourceRevisionMismatch) {
		t.Fatalf("first save should fail with ErrSourceRevisionMismatch, got %v", err)
	}

	fresh, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings after external edit: %v", err)
	}
	if fresh.Revision.Fingerprint == stale.Revision.Fingerprint {
		t.Fatalf("re-read did not pick up the external edit's revision")
	}

	const candidate = "module.exports = { uiPort: 1881 };\n"
	doc, err := svc.SaveRawSettingsWithRevision(candidate, fresh.Revision)
	if err != nil {
		t.Fatalf("retry with fresh revision: %v", err)
	}
	if doc.Revision.Fingerprint == fresh.Revision.Fingerprint {
		t.Errorf("revision did not advance after retry: %s", doc.Revision.Fingerprint)
	}
	if doc.Content != candidate {
		t.Errorf("Content = %q, want %q", doc.Content, candidate)
	}
}

// TestGetRawSettings_CarriesRevision ensures the read path populates the
// revision so the operator (or the structured-edit UI) can capture it for
// the next save without an extra round trip.
func TestGetRawSettings_CarriesRevision(t *testing.T) {
	tempDir := t.TempDir()
	hostSvc := NewIsolatedHostService(tempDir)
	svc := NewConfigServiceWithHost(tempDir, hostSvc)

	const content = "module.exports = { uiPort: 3000 };\n"
	if _, err := svc.SaveRawSettings(content); err != nil {
		t.Fatalf("save: %v", err)
	}

	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}
	if doc.Revision.Fingerprint == "" {
		t.Fatal("Revision.Fingerprint is empty after SaveRawSettings")
	}
	if doc.Revision.Algorithm != SourceRevisionAlgorithm {
		t.Errorf("Revision.Algorithm = %q, want %q", doc.Revision.Algorithm, SourceRevisionAlgorithm)
	}
	if FingerprintSource(doc.Content).Fingerprint != doc.Revision.Fingerprint {
		t.Errorf("Revision.Fingerprint does not match the content's fingerprint")
	}
}
