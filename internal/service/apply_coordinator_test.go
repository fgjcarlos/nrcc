package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// applyCoordinatorFixture wires the minimum graph the
// coordinator tests need: an isolated config service + apply
// service + recorder audit hook + coordinator. The recorder is
// returned so callers can assert the failure_stage meta key
// the slice C acceptance criteria pin.
type applyCoordinatorFixture struct {
	dir       string
	svc       *ConfigService
	apply     *ApplyService
	recorder  *coordinatorRecorder
	coordinator *ApplyCoordinator
	settingsPath string
}

type coordinatorRecorder struct {
	mu     sync.Mutex
	events []recordedAudit
}

func (r *coordinatorRecorder) Log(req *http.Request, actor, action, target, result string, meta map[string]string) {
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

func (r *coordinatorRecorder) snapshot() []recordedAudit {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedAudit, len(r.events))
	copy(out, r.events)
	return out
}

func newApplyCoordinatorFixture(t *testing.T) *applyCoordinatorFixture {
	t.Helper()
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	rec := &coordinatorRecorder{}
	apply := NewApplyService(svc, rec.Log)
	return &applyCoordinatorFixture{
		dir:          dir,
		svc:          svc,
		apply:        apply,
		recorder:     rec,
		coordinator:  NewApplyCoordinator(apply),
		settingsPath: settingsPath,
	}
}

func (f *applyCoordinatorFixture) liveRevision(t *testing.T) model.SourceRevision {
	t.Helper()
	doc, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}
	return doc.Revision
}

// TestApplyCoordinator_HappyPath runs an apply through the
// coordinator and asserts the typed ApplyResult + audit
// failure_stage=ok envelope.
func TestApplyCoordinator_HappyPath(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	live := f.liveRevision(t)
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 7777", 1)

	result, err := f.coordinator.Apply(context.Background(), ApplyRequest{
		Path:      f.settingsPath,
		Content:   candidate,
		Expected:  live,
		BackupDir: filepath.Join(f.dir, "backups", "settings"),
		Actor:     "test",
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.Revision.Fingerprint == "" {
		t.Errorf("result.Revision.Fingerprint is empty")
	}
	if result.Revision.Fingerprint == live.Fingerprint {
		t.Errorf("result.Revision did not advance: %s", result.Revision.Fingerprint)
	}
	events := f.recorder.snapshot()
	if len(events) == 0 {
		t.Fatalf("no audit events recorded")
	}
	last := events[len(events)-1]
	if last.Action != "apply.success" {
		t.Errorf("last event action = %q, want apply.success", last.Action)
	}
	if last.FailureStage != "ok" {
		t.Errorf("last event failure_stage = %q, want ok", last.FailureStage)
	}
}

// TestApplyCoordinator_StaleRevisionRejected is the slice C
// acceptance criterion for the revision envelope: a stale
// expected revision surfaces a *RevisionConflictError with
// failure_stage=validate in the audit envelope.
func TestApplyCoordinator_StaleRevisionRejected(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	live := f.liveRevision(t)
	stale := live
	stale.Fingerprint = strings.Repeat("0", 64)
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 8888", 1)

	_, err := f.coordinator.Apply(context.Background(), ApplyRequest{
		Path:      f.settingsPath,
		Content:   candidate,
		Expected:  stale,
		BackupDir: filepath.Join(f.dir, "backups", "settings"),
		Actor:     "test",
	})
	if err == nil {
		t.Fatal("Apply accepted stale revision; want *RevisionConflictError")
	}
	var conflict *RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v (%T); want *RevisionConflictError", err, err)
	}
	if conflict.Live.Fingerprint != live.Fingerprint {
		t.Errorf("conflict.Live.Fingerprint = %q, want %q", conflict.Live.Fingerprint, live.Fingerprint)
	}

	events := f.recorder.snapshot()
	if len(events) == 0 {
		t.Fatalf("no audit events recorded for stale-revision path")
	}
	last := events[len(events)-1]
	if last.Action != "apply.failure" {
		t.Errorf("last event action = %q, want apply.failure", last.Action)
	}
	if last.FailureStage != "validate" {
		t.Errorf("last event failure_stage = %q, want validate", last.FailureStage)
	}

	// And the on-disk file MUST remain untouched — the rejected
	// apply must not have left a backup or modified settings.js.
	originalBytes, readErr := os.ReadFile(f.settingsPath)
	if readErr != nil {
		t.Fatalf("read settings.js after rejection: %v", readErr)
	}
	if string(originalBytes) != validSettingsJS(t) {
		t.Errorf("settings.js was modified despite rejection:\n got: %q\nwant: %q", string(originalBytes), validSettingsJS(t))
	}
	matches, _ := filepath.Glob(filepath.Join(f.dir, "backups", "settings", "*.js.bak"))
	if len(matches) != 0 {
		t.Errorf("rejected apply created backup(s) %v; expected none", matches)
	}
}

// TestConcurrentAndStaleApplyRejected is the slice C
// acceptance test from issue #758: overlapping applies are
// rejected with a typed error AND a stale revision is rejected
// with a typed error. We exercise both branches in a single
// test so a future regression that lets one path bypass the
// other is caught.
func TestConcurrentAndStaleApplyRejected(t *testing.T) {
	f := newApplyCoordinatorFixture(t)

	// Branch 1: overlapping transactions. Block the first apply
	// inside ApplyService by holding the per-path slot via a
	// tiny stub that wedges until the test releases it. We use
	// a custom ApplyService copy with a slow audit hook +
	// second Apply call to demonstrate the in-flight rejection
	// without needing to instrument the apply pipeline.
	//
	// The simplest reproducible overlap: launch one Apply, then
	// before it returns, fire a second Apply from a goroutine
	// and assert ErrApplyInFlight. Because ApplyService.Apply
	// is fast in the test environment, we instead swap in a
	// coordinator that has its per-path slot pre-acquired.
	flight, err := f.coordinator.acquire(f.settingsPath)
	if err != nil {
		t.Fatalf("acquire pre-held slot: %v", err)
	}
	releaseHeld := func() { f.coordinator.release(f.settingsPath, flight) }
	// Manually released in the test body to demonstrate the
	// second-branch (stale revision) scenario.

	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 7777", 1)
	_, err = f.coordinator.Apply(context.Background(), ApplyRequest{
		Path:      f.settingsPath,
		Content:   candidate,
		Expected:  f.liveRevision(t),
		BackupDir: filepath.Join(f.dir, "backups", "settings"),
		Actor:     "test",
	})
	if !errors.Is(err, ErrApplyInFlight) {
		t.Errorf("overlapping Apply returned %v, want ErrApplyInFlight", err)
	}
	// The in-flight rejection must emit a typed audit event
	// with failure_stage=start so observability can distinguish
	// "rejected before work" from "rejected after validate".
	events := f.recorder.snapshot()
	if len(events) == 0 {
		t.Fatalf("no audit events for in-flight rejection")
	}
	last := events[len(events)-1]
	if last.Action != "apply.failure" {
		t.Errorf("in-flight event action = %q, want apply.failure", last.Action)
	}
	if last.FailureStage != "start" {
		t.Errorf("in-flight event failure_stage = %q, want start", last.FailureStage)
	}

	// Branch 2: stale revision. Release the slot so the next
	// Apply can acquire it, then send a request with a stale
	// expected revision.
	releaseHeld()
	stale := f.liveRevision(t)
	stale.Fingerprint = strings.Repeat("f", 64)
	_, err = f.coordinator.Apply(context.Background(), ApplyRequest{
		Path:      f.settingsPath,
		Content:   candidate,
		Expected:  stale,
		BackupDir: filepath.Join(f.dir, "backups", "settings"),
		Actor:     "test",
	})
	var conflict *RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Errorf("stale Apply returned %v (%T); want *RevisionConflictError", err, err)
	}
}

// TestApplyCoordinator_DifferentPathsDoNotBlock documents the
// per-path keying contract: overlapping applies on different
// settings.js paths must NOT block each other. We exercise the
// contract by acquiring two distinct slots manually and
// asserting neither acquire blocks on the other.
func TestApplyCoordinator_DifferentPathsDoNotBlock(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	pathA := f.settingsPath
	pathB := filepath.Join(f.dir, "settings-other.js")
	if err := os.WriteFile(pathB, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed pathB: %v", err)
	}
	flightA, err := f.coordinator.acquire(pathA)
	if err != nil {
		t.Fatalf("acquire pathA: %v", err)
	}
	defer f.coordinator.release(pathA, flightA)

	// pathB's slot must be independently available even when
	// pathA's slot is held. The coordinator's per-path
	// keying is the contract under test here.
	flightB, err := f.coordinator.acquire(pathB)
	if err != nil {
		t.Fatalf("acquire pathB with pathA held: %v; want nil (different keys should not block)", err)
	}
	f.coordinator.release(pathB, flightB)
}

// TestApplyCoordinator_CancellationPropagates covers the
// cancellation contract: a cancelled ctx returns an error
// before doing any work. We use a pre-cancelled ctx to avoid
// racing the apply pipeline.
func TestApplyCoordinator_CancellationPropagates(t *testing.T) {
	f := newApplyCoordinatorFixture(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.coordinator.Apply(ctx, ApplyRequest{
		Path:      f.settingsPath,
		Content:   "module.exports = { uiPort: 1880 };",
		Expected:  f.liveRevision(t),
		BackupDir: filepath.Join(f.dir, "backups", "settings"),
		Actor:     "test",
	})
	if err == nil {
		t.Fatal("Apply on cancelled ctx returned nil; want non-nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Apply err = %v; want context.Canceled in chain", err)
	}
	// Cancellation must NOT leave a slot acquired — a
	// follow-up Apply on the same path must succeed.
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1234", 1)
	if _, err := f.coordinator.Apply(context.Background(), ApplyRequest{
		Path:      f.settingsPath,
		Content:   candidate,
		Expected:  f.liveRevision(t),
		BackupDir: filepath.Join(f.dir, "backups", "settings"),
		Actor:     "test",
	}); err != nil {
		t.Errorf("follow-up Apply after cancellation: %v", err)
	}
}

// TestApplyCoordinator_EmptyPathRejected guards the boundary
// contract: an empty Path is rejected at the validate stage
// before any I/O.
func TestApplyCoordinator_EmptyPathRejected(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	_, err := f.coordinator.Apply(context.Background(), ApplyRequest{
		Path: "",
	})
	if err == nil {
		t.Fatal("Apply with empty path returned nil; want *ApplyError")
	}
	var ae *ApplyError
	if !errors.As(err, &ae) {
		t.Errorf("err = %v (%T); want *ApplyError", err, err)
	} else if ae.Stage != ApplyStageValidate {
		t.Errorf("ae.Stage = %q, want validate", ae.Stage)
	}
}

// TestApplyCoordinator_ReleaseAfterCompletionUnlocks confirms
// the in-flight slot is released on completion so subsequent
// applies on the same path are not blocked.
func TestApplyCoordinator_ReleaseAfterCompletionUnlocks(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1111", 1)
	live := f.liveRevision(t)

	for i := 0; i < 3; i++ {
		c := strings.Replace(candidate, "uiPort: 1111", fmt.Sprintf("uiPort: %d", 2000+i), 1)
		if _, err := f.coordinator.Apply(context.Background(), ApplyRequest{
			Path:      f.settingsPath,
			Content:   c,
			Expected:  live,
			BackupDir: filepath.Join(f.dir, "backups", "settings"),
			Actor:     "test",
		}); err != nil {
			t.Fatalf("sequential Apply %d returned %v", i, err)
		}
		// Refresh live revision after each successful apply so
		// the next iteration's Expected matches the new on-disk
		// fingerprint.
		live = f.liveRevision(t)
	}
}

// recordedAudit is shared with apply_revision_test.go; Go's
// test compilation model treats each _test.go file as part of
// the same package, so the type declared in
// apply_revision_test.go is visible here without re-import.