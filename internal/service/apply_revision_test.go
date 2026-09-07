package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// recorder captures audit events emitted via AuditHookFunc so
// the slice C tests can assert the failure_stage envelope
// without depending on a real audit.Service. The recorder
// follows the same discipline as the slice A
// recordingAuditHook in apply_test.go.
type recorder struct {
	mu     sync.Mutex
	events []recordedAudit
}

type recordedAudit struct {
	Action       string
	Result       string
	FailureStage string
	HasMeta      bool
	MetaCopy     map[string]string
}

func (r *recorder) Log(req *http.Request, actor, action, target, result string, meta map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec := recordedAudit{
		Action:       action,
		Result:       result,
		FailureStage: meta["failure_stage"],
		HasMeta:      meta != nil,
	}
	if meta != nil {
		rec.MetaCopy = make(map[string]string, len(meta))
		for k, v := range meta {
			rec.MetaCopy[k] = v
		}
	}
	r.events = append(r.events, rec)
}

func (r *recorder) snapshot() []recordedAudit {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedAudit, len(r.events))
	copy(out, r.events)
	return out
}

// TestCheckRevisionPrecondition_MatchHappyPath covers the
// straightforward pass-through: the caller provided the same
// revision the live settings.js carries so the coordinator does
// not raise a conflict.
func TestCheckRevisionPrecondition_MatchHappyPath(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed settings.js: %v", err)
	}
	live, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}

	if err := CheckRevisionPrecondition(context.Background(), svc, ApplyRequest{
		Path:     settingsPath,
		Expected: live.Revision,
	}); err != nil {
		t.Errorf("CheckRevisionPrecondition returned %v, want nil", err)
	}
}

// TestCheckRevisionPrecondition_MismatchReturnsTypedConflict
// is the slice C acceptance criterion for the revision
// envelope: a stale expected revision surfaces a
// *RevisionConflictError carrying the live vs. provided
// fingerprints.
func TestCheckRevisionPrecondition_MismatchReturnsTypedConflict(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	live, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}

	// Fabricate a stale revision: same algorithm, different
	// fingerprint.
	stale := live.Revision
	stale.Fingerprint = strings.Repeat("0", 64)

	err = CheckRevisionPrecondition(context.Background(), svc, ApplyRequest{
		Path:     settingsPath,
		Expected: stale,
	})
	if err == nil {
		t.Fatal("CheckRevisionPrecondition accepted a stale revision; want *RevisionConflictError")
	}
	var conflict *RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v (%T); want *RevisionConflictError", err, err)
	}
	if conflict.Live.Fingerprint != live.Revision.Fingerprint {
		t.Errorf("conflict.Live.Fingerprint = %q, want %q",
			conflict.Live.Fingerprint, live.Revision.Fingerprint)
	}
	if conflict.Provided.Fingerprint != stale.Fingerprint {
		t.Errorf("conflict.Provided.Fingerprint = %q, want %q",
			conflict.Provided.Fingerprint, stale.Fingerprint)
	}
	if conflict.Path != settingsPath {
		t.Errorf("conflict.Path = %q, want %q", conflict.Path, settingsPath)
	}
	// errors.Is must match the slice B sentinel so legacy
	// handlers keep working.
	if !errors.Is(err, ErrSourceRevisionMismatch) {
		t.Errorf("errors.Is(err, ErrSourceRevisionMismatch) = false, want true")
	}
}

// TestCheckRevisionPrecondition_EmptyExpectedBypasses locks in
// the backward-compat bypass: an empty expected.Fingerprint
// skips the live read so a caller that has not yet captured the
// revision is never blocked.
func TestCheckRevisionPrecondition_EmptyExpectedBypasses(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	if err := CheckRevisionPrecondition(context.Background(), svc, ApplyRequest{
		Path:     filepath.Join(dir, "settings.js"),
		Expected: model.SourceRevision{},
	}); err != nil {
		t.Errorf("CheckRevisionPrecondition with empty expected returned %v, want nil", err)
	}
}

// TestCheckRevisionPrecondition_CtxCancelledShortCircuits makes
// sure a cancelled context does not perform any I/O. We use a
// pre-cancelled context to detect this without spinning up
// goroutines.
func TestCheckRevisionPrecondition_CtxCancelledShortCircuits(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := CheckRevisionPrecondition(ctx, svc, ApplyRequest{
		Path:     settingsPath,
		Expected: model.SourceRevision{},
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}