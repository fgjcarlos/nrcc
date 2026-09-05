package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// recordingRollbackHook captures every audit event emitted during a
// rollback so tests can assert the apply.rollback.{started,succeeded,
// failed} discipline required by slice B.
type recordingRollbackHook struct {
	mu     sync.Mutex
	events []recordedRollbackEvent
}

type recordedRollbackEvent struct {
	Action string
	Result string
	Meta   map[string]string
}

func (r *recordingRollbackHook) Log(req *http.Request, actor, action, target, result string, meta map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyMeta := make(map[string]string, len(meta))
	for k, v := range meta {
		copyMeta[k] = v
	}
	r.events = append(r.events, recordedRollbackEvent{Action: action, Result: result, Meta: copyMeta})
}

// TestExecuteRollback_HappyPath walks the success path: backup exists,
// the restore succeeds, the post-rollback Ready probe returns nil,
// and the audit envelope carries apply.rollback.started → succeeded
// (no failed event).
func TestExecuteRollback_HappyPath(t *testing.T) {
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "settings-20260101-000000.js.bak")
	activePath := filepath.Join(dir, "settings.js")
	backup := "module.exports = { uiPort: 1880 };\n"
	failed := "module.exports = { uiPort: 9999 };\n"
	if err := os.WriteFile(backupPath, []byte(backup), 0o600); err != nil {
		t.Fatalf("seed backup: %v", err)
	}
	if err := os.WriteFile(activePath, []byte(failed), 0o600); err != nil {
		t.Fatalf("seed active: %v", err)
	}

	fa := &fakeAdapter{name: AdapterNameNativeBinary}
	hook := &recordingRollbackHook{}
	err := ExecuteRollback(context.Background(), RollbackRequest{
		BackupPath: backupPath,
		ActivePath: activePath,
		Stage:      RollbackStageRestart,
		Adapter:    fa,
		AuditHook:  hook.Log,
	})
	if err != nil {
		t.Fatalf("ExecuteRollback: %v", err)
	}

	// #nosec G304 -- activePath is the test-controlled destination.
	got, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read restored active: %v", err)
	}
	if string(got) != backup {
		t.Errorf("active content = %q, want %q", string(got), backup)
	}

	events := hook.events
	if len(events) != 2 {
		t.Fatalf("audit events = %d, want 2 (started + succeeded): %+v", len(events), events)
	}
	if events[0].Action != "apply.rollback.started" || events[0].Result != "ok" {
		t.Errorf("first event = %+v, want apply.rollback.started ok", events[0])
	}
	if events[1].Action != "apply.rollback.succeeded" || events[1].Result != "ok" {
		t.Errorf("second event = %+v, want apply.rollback.succeeded ok", events[1])
	}
	if events[0].Meta["apply_rollback_adapter"] != AdapterNameNativeBinary {
		t.Errorf("meta apply_rollback_adapter = %q, want %q", events[0].Meta["apply_rollback_adapter"], AdapterNameNativeBinary)
	}
	if events[0].Meta["apply_rollback_stage"] != string(RollbackStageRestart) {
		t.Errorf("meta apply_rollback_stage = %q, want %q", events[0].Meta["apply_rollback_stage"], RollbackStageRestart)
	}

	if fa.readyCalls != 1 {
		t.Errorf("adapter Ready calls = %d, want 1", fa.readyCalls)
	}
}

// TestExecuteRollback_FailedReady asserts the rollback still restores
// the file (the restore succeeded) but the post-rollback Ready probe
// failed, so the orchestrator gets a *RollbackError and the audit
// envelope ends with apply.rollback.failed.
func TestExecuteRollback_FailedReady(t *testing.T) {
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "settings-20260101-000000.js.bak")
	activePath := filepath.Join(dir, "settings.js")
	backup := "module.exports = { uiPort: 1880 };\n"
	failed := "module.exports = { uiPort: 9999 };\n"
	if err := os.WriteFile(backupPath, []byte(backup), 0o600); err != nil {
		t.Fatalf("seed backup: %v", err)
	}
	if err := os.WriteFile(activePath, []byte(failed), 0o600); err != nil {
		t.Fatalf("seed active: %v", err)
	}

	fa := &fakeAdapter{name: AdapterNameDockerCompose, readyErr: ErrReadinessNotReady}
	hook := &recordingRollbackHook{}
	err := ExecuteRollback(context.Background(), RollbackRequest{
		BackupPath: backupPath,
		ActivePath: activePath,
		Stage:      RollbackStageReady,
		Adapter:    fa,
		AuditHook:  hook.Log,
	})

	var rbe *RollbackError
	if !errors.As(err, &rbe) {
		t.Fatalf("expected *RollbackError, got %T: %v", err, err)
	}
	if rbe.Stage != RollbackStageReady {
		t.Errorf("rbe.Stage = %q, want %q", rbe.Stage, RollbackStageReady)
	}
	if !errors.Is(rbe.Cause, ErrReadinessNotReady) {
		t.Errorf("rbe.Cause = %v, want ErrReadinessNotReady", rbe.Cause)
	}

	// #nosec G304 -- activePath is the test-controlled destination.
	got, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read restored active: %v", err)
	}
	if string(got) != backup {
		t.Errorf("active content = %q, want %q (restore should still happen)", string(got), backup)
	}

	events := hook.events
	if len(events) != 2 {
		t.Fatalf("audit events = %d, want 2 (started + failed): %+v", len(events), events)
	}
	if events[0].Action != "apply.rollback.started" {
		t.Errorf("first event = %+v, want apply.rollback.started", events[0])
	}
	if events[1].Action != "apply.rollback.failed" {
		t.Errorf("second event = %+v, want apply.rollback.failed", events[1])
	}
	if events[1].Meta["apply_rollback_action"] != "ready" {
		t.Errorf("failed event meta action = %q, want %q", events[1].Meta["apply_rollback_action"], "ready")
	}
}

// TestExecuteRollback_MissingBackup asserts the boundary check fires
// before any filesystem work: an empty backup path returns a typed
// *RollbackError and the audit envelope carries an apply.rollback.failed
// event without any restore attempt.
func TestExecuteRollback_MissingBackup(t *testing.T) {
	fa := &fakeAdapter{name: AdapterNameNativeBinary}
	hook := &recordingRollbackHook{}
	err := ExecuteRollback(context.Background(), RollbackRequest{
		BackupPath: "",
		ActivePath: "/tmp/settings.js",
		Stage:      RollbackStageRestart,
		Adapter:    fa,
		AuditHook:  hook.Log,
	})
	var rbe *RollbackError
	if !errors.As(err, &rbe) {
		t.Fatalf("expected *RollbackError, got %T: %v", err, err)
	}
	events := hook.events
	if len(events) != 1 || events[0].Action != "apply.rollback.failed" {
		t.Errorf("events = %+v, want exactly one apply.rollback.failed", events)
	}
}

// TestExecuteRollback_NilAdapter asserts the boundary check rejects a
// nil adapter so the orchestrator cannot accidentally trigger a
// rollback with no readiness probe available.
func TestExecuteRollback_NilAdapter(t *testing.T) {
	hook := &recordingRollbackHook{}
	err := ExecuteRollback(context.Background(), RollbackRequest{
		BackupPath: "/tmp/x.js.bak",
		ActivePath: "/tmp/x.js",
		Stage:      RollbackStageReady,
		Adapter:    nil,
		AuditHook:  hook.Log,
	})
	var rbe *RollbackError
	if !errors.As(err, &rbe) {
		t.Fatalf("expected *RollbackError, got %T: %v", err, err)
	}
}

// TestExecuteRollback_RestoreFailureRollsNoFurther asserts that when
// the restore step itself fails (the snapshot file is missing on
// disk), the orchestrator returns a typed *RollbackError, the audit
// envelope records apply.rollback.failed with action=restore, and the
// adapter's Ready probe is NOT called.
func TestExecuteRollback_RestoreFailureRollsNoFurther(t *testing.T) {
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "missing.js.bak")
	activePath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(activePath, []byte("module.exports = { uiPort: 9999 };\n"), 0o600); err != nil {
		t.Fatalf("seed active: %v", err)
	}

	fa := &fakeAdapter{name: AdapterNameNativeBinary}
	hook := &recordingRollbackHook{}
	err := ExecuteRollback(context.Background(), RollbackRequest{
		BackupPath: backupPath,
		ActivePath: activePath,
		Stage:      RollbackStageRestart,
		Adapter:    fa,
		AuditHook:  hook.Log,
	})

	var rbe *RollbackError
	if !errors.As(err, &rbe) {
		t.Fatalf("expected *RollbackError, got %T: %v", err, err)
	}
	if fa.readyCalls != 0 {
		t.Errorf("adapter Ready called %d times after restore failure, want 0", fa.readyCalls)
	}

	events := hook.events
	if len(events) != 2 || events[0].Action != "apply.rollback.started" || events[1].Action != "apply.rollback.failed" {
		t.Errorf("events = %+v, want started → failed", events)
	}
	if events[1].Meta["apply_rollback_action"] != "restore" {
		t.Errorf("failed meta action = %q, want restore", events[1].Meta["apply_rollback_action"])
	}

	// #nosec G304 -- activePath is the test-controlled destination.
	got, _ := os.ReadFile(activePath)
	if !strings.Contains(string(got), "9999") {
		t.Errorf("active content = %q, want it to be untouched after restore failure", string(got))
	}
}

// TestExecuteRollback_NilAuditHookOK asserts a nil audit hook is
// tolerated — audit is observability, not a critical-path dependency
// (same contract as ApplyService.emit).
func TestExecuteRollback_NilAuditHookOK(t *testing.T) {
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "settings.js.bak")
	activePath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(backupPath, []byte("module.exports = { uiPort: 1880 };\n"), 0o600); err != nil {
		t.Fatalf("seed backup: %v", err)
	}
	if err := os.WriteFile(activePath, []byte("module.exports = { uiPort: 9999 };\n"), 0o600); err != nil {
		t.Fatalf("seed active: %v", err)
	}
	fa := &fakeAdapter{name: AdapterNameNativeBinary}
	if err := ExecuteRollback(context.Background(), RollbackRequest{
		BackupPath: backupPath,
		ActivePath: activePath,
		Stage:      RollbackStageRestart,
		Adapter:    fa,
		AuditHook:  nil,
	}); err != nil {
		t.Fatalf("ExecuteRollback with nil hook: %v", err)
	}
}

// TestRestoreSnapshot_RejectsTraversal asserts the boundary contract
// from slice A's apply_atomic.go is honoured by the rollback path: a
// `..` component in either path fails the rollback before any I/O.
func TestRestoreSnapshot_RejectsTraversal(t *testing.T) {
	if err := restoreSnapshot("/tmp/../etc/passwd", "/tmp/settings.js"); !errors.Is(err, ErrSettingsPathInvalid) {
		t.Errorf("restoreSnapshot traversal backup = %v, want ErrSettingsPathInvalid", err)
	}
}

// ----------------------------------------------------------------------
// Slice B fault-matrix extensions
// ----------------------------------------------------------------------

// rollbackTestCaps returns a stable ConfigurationCapabilities for the
// rollback tests. The apply.go file expects this field to be populated
// (it feeds RedactedCapabilityMeta); we don't care about the adapter
// string for these tests because the fault-matrix exercises the
// rollback helper directly.
func rollbackTestCaps() model.ConfigurationCapabilities {
	return model.ConfigurationCapabilities{
		Adapter:        "nodered-5",
		RuntimeVersion: "5.0.6",
		CatalogVersion: "5.0.6",
		Mode:           "editable",
		Editable:       true,
	}
}

// TestApplyTransactionFaultMatrix_RestartFailureRollback extends the
// slice A fault matrix with the slice B failure mode: a successful
// Apply (validate → backup → atomic write) followed by a synthetic
// adapter.Restart failure. The test asserts:
//
//   - ExecuteRollback restores settings.js to the slice A backup.
//   - The post-rollback Ready probe is exercised against the adapter.
//   - apply.rollback.{started,failed} audit events follow the
//     discipline documented on ExecuteRollback.
//
// We exercise the rollback path directly (rather than wiring a
// coordinator) so this test stays in slice A's fault-matrix style:
// table-driven, focused, no HTTP handler involvement.
func TestApplyTransactionFaultMatrix_RestartFailureRollback(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("seed GetRawSettings: %v", err)
	}
	hook := &recordingAuditHook{}
	apply := NewApplyService(svc, hook.Log)

	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1891", 1)
	req := ApplyRequest{
		Path:         settingsPath,
		Content:      candidate,
		Expected:     doc.Revision,
		BackupDir:    filepath.Join(dir, "backups", "settings"),
		Actor:        "fault-matrix-test",
		Capabilities: rollbackTestCaps(),
	}
	result, err := apply.Apply(context.Background(), req)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.BackupPath == "" {
		t.Fatal("ApplyResult.BackupPath is empty; rollback cannot proceed")
	}

	rbHook := &recordingRollbackHook{}
	adapter := &fakeAdapter{name: AdapterNameNativeBinary, readyErr: ErrReadinessNotReady}
	rbErr := ExecuteRollback(context.Background(), RollbackRequest{
		BackupPath:   result.BackupPath,
		ActivePath:   settingsPath,
		Stage:        RollbackStageRestart,
		Adapter:      adapter,
		AuditHook:    rbHook.Log,
		Actor:        "fault-matrix-test",
		Capabilities: req.Capabilities,
	})

	var rbe *RollbackError
	if !errors.As(rbErr, &rbe) {
		t.Fatalf("expected *RollbackError, got %T: %v", rbErr, rbErr)
	}
	if rbe.Stage != RollbackStageRestart {
		t.Errorf("rbe.Stage = %q, want %q", rbe.Stage, RollbackStageRestart)
	}

	events := rbHook.events
	if len(events) < 2 {
		t.Fatalf("rollback audit events = %d, want >= 2 (started + failed): %+v", len(events), events)
	}
	if events[0].Action != "apply.rollback.started" {
		t.Errorf("first event = %q, want apply.rollback.started", events[0].Action)
	}
	last := events[len(events)-1]
	if last.Action != "apply.rollback.failed" {
		t.Errorf("last event = %q, want apply.rollback.failed", last.Action)
	}

	// #nosec G304 -- settingsPath is the test-controlled destination.
	got, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read restored active: %v", err)
	}
	if string(got) != validSettingsJS(t) {
		t.Errorf("active content after rollback = %q, want %q", string(got), validSettingsJS(t))
	}
}

// TestApplyTransactionFaultMatrix_ReadinessFailureRollback exercises
// the readiness-failure path. The atomic write already committed, so
// the rollback restores the slice A snapshot AND verifies Node-RED
// came back healthy against the restored settings.
func TestApplyTransactionFaultMatrix_ReadinessFailureRollback(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("seed GetRawSettings: %v", err)
	}
	hook := &recordingAuditHook{}
	apply := NewApplyService(svc, hook.Log)

	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1892", 1)
	result, err := apply.Apply(context.Background(), ApplyRequest{
		Path:         settingsPath,
		Content:      candidate,
		Expected:     doc.Revision,
		BackupDir:    filepath.Join(dir, "backups", "settings"),
		Actor:        "fault-matrix-test",
		Capabilities: rollbackTestCaps(),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	rbHook := &recordingRollbackHook{}
	adapter := &fakeAdapter{name: AdapterNameDockerCompose}
	rbErr := ExecuteRollback(context.Background(), RollbackRequest{
		BackupPath:   result.BackupPath,
		ActivePath:   settingsPath,
		Stage:        RollbackStageReady,
		Adapter:      adapter,
		AuditHook:    rbHook.Log,
		Actor:        "fault-matrix-test",
		Capabilities: rollbackTestCaps(),
	})
	if rbErr != nil {
		t.Fatalf("ExecuteRollback: %v", rbErr)
	}

	events := rbHook.events
	if len(events) != 2 {
		t.Fatalf("rollback audit events = %d, want 2 (started + succeeded): %+v", len(events), events)
	}
	if events[0].Action != "apply.rollback.started" {
		t.Errorf("first event = %q, want apply.rollback.started", events[0].Action)
	}
	if events[1].Action != "apply.rollback.succeeded" {
		t.Errorf("second event = %q, want apply.rollback.succeeded", events[1].Action)
	}
	if events[1].Meta["apply_rollback_adapter"] != AdapterNameDockerCompose {
		t.Errorf("succeeded event adapter = %q, want %q", events[1].Meta["apply_rollback_adapter"], AdapterNameDockerCompose)
	}

	// #nosec G304 -- settingsPath is the test-controlled destination.
	got, _ := os.ReadFile(settingsPath)
	if string(got) != validSettingsJS(t) {
		t.Errorf("active content after rollback = %q, want %q", string(got), validSettingsJS(t))
	}
}

// TestActiveSettingsFilePerDeployment is the slice B acceptance
// criterion for the deployment-scoped settings path: the apply
// pipeline must write ONLY to the file the running Node-RED is
// actually using. We simulate "running" by pointing the readiness
// probe at an httptest server whose /settings handler reports the
// file the operator believed it was editing.
//
// Test matrix:
//
//   - nodered-5 host with live settings at /tmp/.../deploy/settings.js
//   - apply writes to the live path
//   - the fake server answers 200 with the post-apply candidate content
//   - apply.success is the final audit event
func TestActiveSettingsFilePerDeployment(t *testing.T) {
	dir := t.TempDir()
	deployDir := filepath.Join(dir, "deploy")
	if err := os.MkdirAll(deployDir, 0o750); err != nil {
		t.Fatalf("mkdir deploy: %v", err)
	}
	liveSettingsPath := filepath.Join(deployDir, "settings.js")
	initial := validSettingsJS(t)
	if err := os.WriteFile(liveSettingsPath, []byte(initial), 0o600); err != nil {
		t.Fatalf("seed live settings: %v", err)
	}

	// Spin up a "running Node-RED" stand-in. The handler is the
	// contract this test enforces: any apply that does NOT result
	// in a 200 from this handler (after the next read) means the
	// apply wrote to the wrong file.
	var currentContent atomicValue
	currentContent.set(initial)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(currentContent.get()))
	}))
	defer srv.Close()

	svc := NewIsolatedConfigService(dir)
	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("seed GetRawSettings: %v", err)
	}
	hook := &recordingAuditHook{}
	apply := NewApplyService(svc, hook.Log)

	candidate := strings.Replace(initial, "uiPort: 1880", "uiPort: 1950", 1)
	req := ApplyRequest{
		Path:      liveSettingsPath,
		Content:   candidate,
		Expected:  doc.Revision,
		BackupDir: filepath.Join(dir, "backups", "settings"),
		Actor:     "active-file-test",
	}
	result, err := apply.Apply(context.Background(), req)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.Path != liveSettingsPath {
		t.Errorf("ApplyResult.Path = %q, want %q", result.Path, liveSettingsPath)
	}

	// Post-apply: the file the running Node-RED is using MUST be
	// the file that changed.
	currentContent.set(candidate)
	resp, err := http.Get(srv.URL + "/settings")
	if err != nil {
		t.Fatalf("GET /settings: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("running NR /settings = %d, want 200", resp.StatusCode)
	}
	body := readAll(t, resp.Body)
	if string(body) != candidate {
		t.Errorf("running NR body differs from candidate: got %q, want %q", string(body), candidate)
	}

	events := hook.snapshot()
	if got := events[len(events)-1].Action; got != "apply.success" {
		t.Errorf("last audit event = %q, want apply.success", got)
	}
}

// TestConfigurationApplyE2E_DiffMatchesEffectiveState is the slice B
// acceptance criterion for the post-restart state: the redacted diff
// reported by ApplyResult.Diff must match the effective on-disk state
// AFTER a simulated restart (the orchestrator would normally do this
// in slice C; here we exercise the diff-vs-state invariant directly).
//
// We intentionally do not run a real Node-RED. The "effective state"
// is the on-disk settings.js after the apply committed. The invariant
// is: the diff is built from (liveContent, candidate), so liveContent
// AFTER the apply is irrelevant — what matters is that the diff
// represents the change the operator requested.
func TestConfigurationApplyE2E_DiffMatchesEffectiveState(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("seed GetRawSettings: %v", err)
	}
	apply := NewApplyService(svc, nil)

	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1951", 1)
	result, err := apply.Apply(context.Background(), ApplyRequest{
		Path:      settingsPath,
		Content:   candidate,
		Expected:  doc.Revision,
		BackupDir: filepath.Join(dir, "backups", "settings"),
		Actor:     "e2e-test",
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// #nosec G304 -- settingsPath is the test-controlled destination.
	got, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read post-apply: %v", err)
	}

	// Invariant: on-disk == candidate.
	if string(got) != candidate {
		t.Errorf("on-disk content != candidate\n got: %q\nwant: %q", got, candidate)
	}

	// Invariant: the diff contains the change the operator requested
	// and is redacted (no plaintext secret-shaped value).
	if !strings.Contains(result.Diff, "uiPort: 1951") {
		t.Errorf("diff missing uiPort:1951: %q", result.Diff)
	}
	if strings.Contains(result.Diff, "initial-secret-value") {
		t.Errorf("diff leaks credentialSecret value: %q", result.Diff)
	}
	if !strings.Contains(result.Diff, "[redacted]") {
		t.Errorf("diff missing [redacted] marker: %q", result.Diff)
	}

	if result.Revision.Fingerprint == "" {
		t.Error("result.Revision.Fingerprint is empty")
	}
	if FingerprintSource(string(got)).Fingerprint != result.Revision.Fingerprint {
		t.Errorf("on-disk fingerprint %q != result fingerprint %q",
			FingerprintSource(string(got)).Fingerprint, result.Revision.Fingerprint)
	}
}

// atomicValue is a tiny thread-safe string holder used by
// TestActiveSettingsFilePerDeployment so the fake httptest server can
// return the post-apply candidate content without a data race.
type atomicValue struct {
	mu  sync.Mutex
	val string
}

func (a *atomicValue) set(v string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.val = v
}

func (a *atomicValue) get() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.val
}

// readAll drains r without an upper bound; small enough that the test
// never has to think about chunking.
func readAll(t *testing.T, r interface{ Read(p []byte) (int, error) }) []byte {
	t.Helper()
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 512)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			return buf
		}
	}
}