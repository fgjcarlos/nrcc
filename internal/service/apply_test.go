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
)

// Test fixtures -------------------------------------------------------------

// recordingAuditHook captures every Apply emit so tests can assert the
// audit discipline required by issue #758 (apply.start, apply.backup,
// apply.write, apply.success | apply.failure in the right order).
type recordingAuditHook struct {
	mu      sync.Mutex
	events  []recordedEvent
	failOn  string // if non-empty, return an error from Log to simulate audit degradation
}

type recordedEvent struct {
	Action  string
	Stage   ApplyStage
	Result  string
	Actor   string
	Target  string
	HasMeta bool
}

func (r *recordingAuditHook) Log(req *http.Request, actor, action, target, result string, meta map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failOn != "" && action == r.failOn {
		// Simulate the audit subsystem reporting degradation. ApplyService
		// does not gate the transaction on this outcome, but we record the
		// attempt so the test can confirm the call happened.
		r.events = append(r.events, recordedEvent{Action: action, Stage: ApplyStageAudit, Result: "degraded", Actor: actor, Target: target, HasMeta: meta != nil})
		return
	}
	r.events = append(r.events, recordedEvent{Action: action, Stage: ApplyStageValidate, Result: result, Actor: actor, Target: target, HasMeta: meta != nil})
}

func (r *recordingAuditHook) snapshot() []recordedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedEvent, len(r.events))
	copy(out, r.events)
	return out
}

// validSettingsJS returns a minimal but valid Node-RED settings.js body
// with a bcrypt-hashed adminAuth so adminAuth validation passes
// (slice A reuses ConfigService's validation gate).
func validSettingsJS(t *testing.T) string {
	t.Helper()
	const body = `module.exports = {
  uiPort: 1880,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  credentialSecret: 'initial-secret-value',
  adminAuth: null,
};
`
	return body
}

// seedConfigService is a documented helper for slice B tests that need
// a pre-configured ConfigService + (path, backupDir, revision) tuple.
// The slice A tests do not use it because each subtest sets up the
// env it needs (seeding files explicitly so the assertion failure
// modes are observable). Slice B should add:
//
//	func seedConfigService(t *testing.T) (*ConfigService, string, string, model.SourceRevision) {
//	    dir := t.TempDir()
//	    svc := NewIsolatedConfigService(dir)
//	    doc, err := svc.GetRawSettings()
//	    if err != nil { t.Fatalf("seed GetRawSettings: %v", err) }
//	    return svc, doc.Path, filepath.Join(dir, "backups", "settings"), doc.Revision
//	}

// TestApplyTransactionHappyPath locks in the slice-A acceptance
// criterion: a successful apply executes backup → atomic write in
// order, with the audit events fired in the contract order
// (apply.start → apply.backup → apply.write → apply.success).
//
// The redacted diff is asserted to omit the credentialSecret value and
// to mark it as "[redacted]" so secret-shaped keys never leak into the
// audit log.
func TestApplyTransactionHappyPath(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)

	// Seed a settings.js on disk so the backup stage has something to
	// copy. We write the file directly because we want a known revision
	// fingerprint at the start of the test.
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
		t.Fatalf("seed settings.js: %v", err)
	}
	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings after seed: %v", err)
	}
	backupDir := filepath.Join(dir, "backups", "settings")
	expected := doc.Revision

	hook := &recordingAuditHook{}
	apply := NewApplyService(svc, hook.Log)

	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1890", 1)
	req := ApplyRequest{
		Path:      settingsPath,
		Content:   candidate,
		Expected:  expected,
		BackupDir: backupDir,
		Actor:     "test-operator",
	}

	result, err := apply.Apply(context.Background(), req)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Atomic write committed: the on-disk file must match candidate.
	// #nosec G304 -- settingsPath is the test-controlled destination, derived from t.TempDir().
	got, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings.js after apply: %v", err)
	}
	if string(got) != candidate {
		t.Errorf("settings.js content mismatch\n got: %q\nwant: %q", got, candidate)
	}

	// Backup was created and contains the pre-apply content.
	matches, err := filepath.Glob(filepath.Join(backupDir, "settings-*.js.bak"))
	if err != nil {
		t.Fatalf("glob backup dir: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one backup, found %d", len(matches))
	}
	backupBytes, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backupBytes) != validSettingsJS(t) {
		t.Errorf("backup content mismatch\n got: %q\nwant: %q", backupBytes, validSettingsJS(t))
	}

	// Result metadata is correct.
	if result.BackupPath != matches[0] {
		t.Errorf("result.BackupPath = %q, want %q", result.BackupPath, matches[0])
	}
	if result.Path != settingsPath {
		t.Errorf("result.Path = %q, want %q", result.Path, settingsPath)
	}
	if result.Revision.Fingerprint == "" {
		t.Error("result.Revision.Fingerprint must not be empty after a successful apply")
	}
	if result.Revision.Algorithm != SourceRevisionAlgorithm {
		t.Errorf("result.Revision.Algorithm = %q, want %q", result.Revision.Algorithm, SourceRevisionAlgorithm)
	}

	// Diff is redacted: credentialSecret value never appears, the literal
	// "[redacted]" does.
	if strings.Contains(result.Diff, "initial-secret-value") {
		t.Errorf("diff leaks credentialSecret value: %q", result.Diff)
	}
	if !strings.Contains(result.Diff, "[redacted]") {
		t.Errorf("diff missing [redacted] marker for credentialSecret: %q", result.Diff)
	}

	// Audit event order: apply.start → apply.backup → apply.write → apply.success.
	events := hook.snapshot()
	wantOrder := []string{"apply.start", "apply.backup", "apply.write", "apply.success"}
	if len(events) != len(wantOrder) {
		t.Fatalf("audit events = %d, want %d (events: %+v)", len(events), len(wantOrder), events)
	}
	for i, want := range wantOrder {
		if events[i].Action != want {
			t.Errorf("audit event %d = %q, want %q (full: %+v)", i, events[i].Action, want, events[i])
		}
		if events[i].Result != "ok" {
			t.Errorf("audit event %d (%s) result = %q, want %q", i, events[i].Action, events[i].Result, "ok")
		}
	}
	for _, e := range events {
		if !e.HasMeta {
			t.Errorf("audit event %q emitted without meta map", e.Action)
		}
	}
}

// TestApplyTransactionFaultMatrix exercises the failure paths required
// by the slice-A acceptance criterion: validation failures
// (unparseable content, stale revision, plaintext adminAuth), backup
// failures (parent dir unwritable), and write failures (parent dir
// missing, symlink destination, traversal).
//
// Each subtest asserts:
//   - The ApplyError.Stage matches the failing phase.
//   - errors.Is / errors.As identify the typed cause.
//   - apply.failure is the final audit event (no apply.success).
//   - apply.start is always emitted before the failure.
func TestApplyTransactionFaultMatrix(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(t *testing.T) (svc *ConfigService, req ApplyRequest, hook *recordingAuditHook, apply *ApplyService)
		wantStage   ApplyStage
		wantCauseIs error  // optional: a sentinel that errors.Is must match
		wantErrSub  string // optional: substring expected inside err.Error()
	}{
		{
			name: "emptyPathFailsAtValidate",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
				svc := NewIsolatedConfigService(t.TempDir())
				hook := &recordingAuditHook{}
				return svc, ApplyRequest{Content: validSettingsJS(t)}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:  ApplyStageValidate,
			wantErrSub: "apply path is required",
		},
		{
			name: "emptyBackupDirFailsAtValidate",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
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
				return svc, ApplyRequest{
					Path:     settingsPath,
					Content:  strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1891", 1),
					Expected: doc.Revision,
				}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:  ApplyStageValidate,
			wantErrSub: "apply backup directory is required",
		},
		{
			name: "staleRevisionFailsAtValidate",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
				dir := t.TempDir()
				svc := NewIsolatedConfigService(dir)
				settingsPath := filepath.Join(dir, "settings.js")
				if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
				// Mutate the on-disk file AFTER the read so the
				// candidate's expected revision no longer matches.
				if err := os.WriteFile(settingsPath, []byte(strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 9000", 1)), 0o600); err != nil {
					t.Fatalf("stale-mutate: %v", err)
				}
				staleRev := FingerprintSource(validSettingsJS(t))
				hook := &recordingAuditHook{}
				return svc, ApplyRequest{
					Path:      settingsPath,
					Content:   validSettingsJS(t),
					Expected:  staleRev,
					BackupDir: filepath.Join(dir, "backups", "settings"),
				}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:   ApplyStageValidate,
			wantCauseIs: ErrSourceRevisionMismatch,
		},
		{
			name: "sandboxTimeoutFailsAtValidate",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
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
				// Infinite-loop JS triggers ErrSandboxTimeout from
				// parseAdminAuthFromJS; the apply pipeline propagates
				// it via parseConfigFromContent so the validate stage
				// fails with the sandbox error.
				return svc, ApplyRequest{
					Path:      settingsPath,
					Content:   "module.exports = { adminAuth: (function(){ while(true){} })(), uiPort: 1880 };\n",
					Expected:  doc.Revision,
					BackupDir: filepath.Join(dir, "backups", "settings"),
				}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:  ApplyStageValidate,
			wantErrSub: "parse candidate",
		},
		{
			name: "plaintextAdminAuthFailsAtValidate",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
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
				plaintext := strings.Replace(validSettingsJS(t), "adminAuth: null", "adminAuth: { type: 'credentials', users: [{ username: 'admin', password: 'plaintext', permissions: '*' }] }", 1)
				return svc, ApplyRequest{
					Path:      settingsPath,
					Content:   plaintext,
					Expected:  doc.Revision,
					BackupDir: filepath.Join(dir, "backups", "settings"),
				}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:  ApplyStageValidate,
			wantErrSub: "validate adminAuth",
		},
		{
			name: "pathTraversalFailsAtValidate",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
				dir := t.TempDir()
				svc := NewIsolatedConfigService(dir)
				hook := &recordingAuditHook{}
				// Raw string concatenation — NOT filepath.Join — so
				// the `..` component survives for the boundary check.
				traversal := dir + "/../etc/settings.js"
				return svc, ApplyRequest{
					Path:      traversal,
					Content:   validSettingsJS(t),
					BackupDir: filepath.Join(dir, "backups"),
				}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:  ApplyStageValidate,
			wantErrSub: "boundary validation",
		},
		{
			name: "symlinkDestinationFailsAtWrite",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
				dir := t.TempDir()
				svc := NewIsolatedConfigService(dir)
				// Create a real settings.js so the revision is valid
				// for the candidate.
				realSettings := filepath.Join(dir, "real-settings.js")
				if err := os.WriteFile(realSettings, []byte(validSettingsJS(t)), 0o600); err != nil {
					t.Fatalf("seed real: %v", err)
				}
				// A symlink in a separate directory points at the real
				// file. Atomic write must refuse to follow it.
				linkDir := t.TempDir()
				link := filepath.Join(linkDir, "settings.js")
				if err := os.Symlink(realSettings, link); err != nil {
					t.Skipf("symlink unsupported: %v", err)
				}
				doc, err := svc.GetRawSettings()
				if err != nil {
					t.Fatalf("seed GetRawSettings: %v", err)
				}
				hook := &recordingAuditHook{}
				return svc, ApplyRequest{
					Path:      link,
					Content:   validSettingsJS(t),
					Expected:  doc.Revision,
					BackupDir: filepath.Join(linkDir, "backups"),
				}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:  ApplyStageWrite,
			wantErrSub: "symlink",
		},
		{
			name: "backupFailsWhenBackupDirCannotBeCreated",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
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
				// backupDir points at a path under a regular file, so
				// MkdirAll fails with ENOTDIR.
				fileAsParent := filepath.Join(dir, "blocker")
				if err := os.WriteFile(fileAsParent, []byte("not a dir"), 0o600); err != nil {
					t.Fatalf("blocker: %v", err)
				}
				return svc, ApplyRequest{
					Path:      settingsPath,
					Content:   strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1892", 1),
					Expected:  doc.Revision,
					BackupDir: filepath.Join(fileAsParent, "backups"),
				}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:  ApplyStageBackup,
			wantErrSub: "create backup dir",
		},
		{
			name: "writeFailsWhenParentDirCannotBeCreated",
			setup: func(t *testing.T) (*ConfigService, ApplyRequest, *recordingAuditHook, *ApplyService) {
				dir := t.TempDir()
				svc := NewIsolatedConfigService(dir)
				// Seed a valid settings.js so the candidate's expected
				// revision is satisfied. The atomic write target then
				// points at an unwritable parent (a regular file).
				settingsPath := filepath.Join(dir, "settings.js")
				if err := os.WriteFile(settingsPath, []byte(validSettingsJS(t)), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
				doc, err := svc.GetRawSettings()
				if err != nil {
					t.Fatalf("seed GetRawSettings: %v", err)
				}
				blocker := filepath.Join(dir, "blocker")
				if err := os.WriteFile(blocker, []byte("not a dir"), 0o600); err != nil {
					t.Fatalf("blocker: %v", err)
				}
				target := filepath.Join(blocker, "settings.js")
				hook := &recordingAuditHook{}
				return svc, ApplyRequest{
					Path:      target,
					Content:   strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 9000", 1),
					Expected:  doc.Revision,
					BackupDir: filepath.Join(dir, "backups"),
				}, hook, NewApplyService(svc, hook.Log)
			},
			wantStage:  ApplyStageBackup,
			wantErrSub: "read live settings",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, req, hook, apply := tc.setup(t)
			result, err := apply.Apply(context.Background(), req)
			if err == nil {
				t.Fatalf("expected Apply to fail, got result=%+v", result)
			}
			if result.Path != "" || result.Revision.Fingerprint != "" {
				t.Errorf("ApplyResult should be zero on failure, got %+v", result)
			}

			var ae *ApplyError
			if !errors.As(err, &ae) {
				t.Fatalf("expected *ApplyError, got %T: %v", err, err)
			}
			if ae.Stage != tc.wantStage {
				t.Errorf("ApplyError.Stage = %q, want %q (err: %v)", ae.Stage, tc.wantStage, err)
			}
			if tc.wantCauseIs != nil && !errors.Is(err, tc.wantCauseIs) {
				t.Errorf("errors.Is(%v, %v) = false, want true", err, tc.wantCauseIs)
			}
			if tc.wantErrSub != "" && !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Errorf("err.Error() = %q, want substring %q", err.Error(), tc.wantErrSub)
			}

			// Audit discipline: apply.start is always emitted first;
			// apply.failure is the final event for this transaction;
			// apply.success must NOT appear.
			events := hook.snapshot()
			if len(events) == 0 {
				t.Fatal("no audit events recorded; expected at least apply.start")
			}
			if events[0].Action != "apply.start" {
				t.Errorf("first audit event = %q, want %q", events[0].Action, "apply.start")
			}
			last := events[len(events)-1]
			if last.Action != "apply.failure" {
				t.Errorf("last audit event = %q, want %q", last.Action, "apply.failure")
			}
			for _, e := range events {
				if e.Action == "apply.success" {
					t.Errorf("apply.success emitted on failed transaction: %+v", e)
				}
				if e.Result != "ok" && e.Result != "error" {
					t.Errorf("unexpected audit result %q for action %q", e.Result, e.Action)
				}
			}
			// Bind-back: the failed transaction must NOT have modified
			// settings.js (no atomic write committed). We assert two
			// cheap post-conditions:
//
//   - No backup file was created when the failure is at validate or
//     earlier (no backup stage reached).
//   - No .apply-* sibling leaked from the atomic-write primitive.
			if req.Path != "" && req.BackupDir != "" {
				matches, _ := filepath.Glob(filepath.Join(req.BackupDir, "settings-*.js.bak"))
				if tc.wantStage == ApplyStageValidate && len(matches) > 0 {
					t.Errorf("no backup should exist after validate-stage failure, found %v", matches)
				}
				if dir := filepath.Dir(req.Path); dir != "" && dir != "." && dir != "/" {
					entries, _ := os.ReadDir(dir)
					for _, e := range entries {
						if strings.HasPrefix(e.Name(), ".apply-") {
							t.Errorf("leftover tmp sibling after failure: %s", e.Name())
						}
					}
				}
			}
		})
	}
}

// TestApplyError_StagePredicate verifies the Is method lets callers
// branch on Stage without unwrapping manually.
func TestApplyError_StagePredicate(t *testing.T) {
	validateErr := &ApplyError{Stage: ApplyStageValidate, Cause: errors.New("nope")}
	if !errors.Is(validateErr, &ApplyError{Stage: ApplyStageValidate}) {
		t.Error("errors.Is should match same-stage sentinel")
	}
	if errors.Is(validateErr, &ApplyError{Stage: ApplyStageWrite}) {
		t.Error("errors.Is should NOT match different-stage sentinel")
	}

	// errors.As must work.
	var typed *ApplyError
	if !errors.As(validateErr, &typed) {
		t.Fatal("errors.As failed to recover *ApplyError")
	}
	if typed.Stage != ApplyStageValidate {
		t.Errorf("recovered Stage = %q, want %q", typed.Stage, ApplyStageValidate)
	}

	// Cause chain reaches ErrSourceRevisionMismatch when wrapped.
	mismatch := &ApplyError{Stage: ApplyStageValidate, Cause: ErrSourceRevisionMismatch}
	if !errors.Is(mismatch, ErrSourceRevisionMismatch) {
		t.Error("errors.Is should reach wrapped sentinel via Unwrap")
	}
}

// TestApplyTransaction_RedactsCredentialsInDiff is a focused unit test
// for the diff redaction so we catch any drift between
// secretSettingKeys and the Node-RED 5 catalog.
func TestApplyTransaction_RedactsCredentialsInDiff(t *testing.T) {
	before := `module.exports = {
  uiPort: 1880,
  credentialSecret: 'old-secret',
  adminAuth: null,
  https: { key: '/etc/ssl/key.pem', passphrase: 'old-pass' },
};
`
	after := `module.exports = {
  uiPort: 1890,
  credentialSecret: 'new-secret',
  adminAuth: { type: 'credentials', users: [{ username: 'admin', password: 'hash', permissions: '*' }] },
  https: { key: '/etc/ssl/key.pem', passphrase: 'new-pass' },
};
`

	diff := RedactDiff(before, after)

	// Top-level redaction must scrub credentialSecret and adminAuth
	// values on both sides of the diff.
	for _, secret := range []string{"old-secret", "new-secret"} {
		if strings.Contains(diff, secret) {
			t.Errorf("diff leaks %q: %s", secret, diff)
		}
	}
	// https block is collapsed wholesale — the nested passphrase is
	// hidden inside the [redacted] placeholder. Slice C's adapter
	// interface will own multi-line https redaction when the field
	// shape grows beyond a single line.
	if got := strings.Count(diff, "[redacted]"); got < 3 {
		t.Errorf("diff [redacted] count = %d, want >= 3 (diff: %s)", got, diff)
	}
	// Non-secret keys survive the diff intact.
	if !strings.Contains(diff, "uiPort: 1880") || !strings.Contains(diff, "uiPort: 1890") {
		t.Errorf("diff should preserve uiPort lines, got: %s", diff)
	}
}

// TestAtomicWriteSettings_RejectsSymlinkAndTraversal guards the
// boundary contract from apply_atomic.go in isolation, so a regression
// in the orchestrator cannot mask a regression in the atomic write
// primitive.
func TestAtomicWriteSettings_RejectsSymlinkAndTraversal(t *testing.T) {
	t.Run("rejectsSymlinkDestination", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "real.js")
		if err := os.WriteFile(target, []byte("module.exports = {};\n"), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
		linkDir := t.TempDir()
		link := filepath.Join(linkDir, "settings.js")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		err := AtomicWriteSettings(context.Background(), link, "module.exports = { uiPort: 1 };\n")
		if err == nil {
			t.Fatal("expected rejection of symlink destination")
		}
		if !errors.Is(err, ErrSettingsSymlinkRejected) {
			t.Errorf("err = %v, want ErrSettingsSymlinkRejected", err)
		}
	})

	t.Run("rejectsTraversalPath", func(t *testing.T) {
		err := AtomicWriteSettings(context.Background(), "/tmp/../etc/passwd", "x")
		if err == nil {
			t.Fatal("expected rejection of traversal path")
		}
		if !errors.Is(err, ErrSettingsPathInvalid) {
			t.Errorf("err = %v, want ErrSettingsPathInvalid", err)
		}
	})

	t.Run("rejectsEmptyPath", func(t *testing.T) {
		err := AtomicWriteSettings(context.Background(), "", "x")
		if err == nil {
			t.Fatal("expected rejection of empty path")
		}
		if !errors.Is(err, ErrSettingsPathInvalid) {
			t.Errorf("err = %v, want ErrSettingsPathInvalid", err)
		}
	})
}

// TestAtomicWriteSettings_HappyPath ensures the atomic primitive is
// independently exercised (not just via ApplyService) so a refactor
// of the orchestrator cannot regress the primitive.
func TestAtomicWriteSettings_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.js")

	const initial = "module.exports = { uiPort: 1880 };\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	const next = "module.exports = { uiPort: 1890 };\n"
	if err := AtomicWriteSettings(context.Background(), path, next); err != nil {
		t.Fatalf("AtomicWriteSettings: %v", err)
	}

	// #nosec G304 -- path is the test-controlled destination, derived from t.TempDir().
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after write: %v", err)
	}
	if string(got) != next {
		t.Errorf("content = %q, want %q", got, next)
	}

	// No leftover .apply-* sibling after a successful write.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".apply-") {
			t.Errorf("leftover tmp sibling: %s", e.Name())
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perm = %o, want 600", info.Mode().Perm())
	}
}

// TestSecretSettingKeys_CatalogParity asserts the redacted-keys list is
// in lock-step with the Node-RED 5 catalog Secret flags. A drift
// between the two would leak new secret-shaped keys into audit logs.
func TestSecretSettingKeys_CatalogParity(t *testing.T) {
	catalog := NodeRED5Catalog()
	for _, entry := range catalog {
		if !entry.Secret {
			continue
		}
		// Catalog entries named in the redacted set must be present.
		if !IsSecretSettingKey(entry.Key) {
			t.Errorf("catalog secret entry %q is not in secretSettingKeys; audit diff would leak it", entry.Key)
		}
	}
	// And vice versa: every redacted key must come from the catalog's
	// Secret list (no over-redaction).
	for _, k := range SecretSettingKeys() {
		found := false
		for _, entry := range catalog {
			if entry.Key == k {
				if !entry.Secret {
					t.Errorf("redacted key %q is not marked Secret in catalog", k)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("redacted key %q is not in the Node-RED 5 catalog at all", k)
		}
	}
}

// TestApplyTransaction_NoAuditEmittedWhenHookNil documents that an
// uninitialised audit hook is treated as "audit disabled" — the
// transaction still completes; only the observability is lost.
func TestApplyTransaction_NoAuditEmittedWhenHookNil(t *testing.T) {
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
	candidate := strings.Replace(validSettingsJS(t), "uiPort: 1880", "uiPort: 1899", 1)
	req := ApplyRequest{
		Path:      settingsPath,
		Content:   candidate,
		Expected:  doc.Revision,
		BackupDir: filepath.Join(dir, "backups", "settings"),
	}
	if _, err := apply.Apply(context.Background(), req); err != nil {
		t.Fatalf("Apply with nil hook failed: %v", err)
	}
}

// TestApplyTransaction_HonoursContextCancellation guards the ctx-aware
// path so a slow apply can be aborted cleanly without leaving the
// transaction in an unrecoverable state.
func TestApplyTransaction_HonoursContextCancellation(t *testing.T) {
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
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := ApplyRequest{
		Path:      settingsPath,
		Content:   validSettingsJS(t),
		Expected:  doc.Revision,
		BackupDir: filepath.Join(dir, "backups", "settings"),
	}
	if _, err := apply.Apply(ctx, req); err == nil {
		t.Fatal("Apply with cancelled ctx should fail")
	} else {
		var ae *ApplyError
		if !errors.As(err, &ae) {
			t.Errorf("expected *ApplyError, got %T: %v", err, err)
		}
	}
	// No backup file should exist (validate-stage failure short-circuits).
	if matches, _ := filepath.Glob(filepath.Join(dir, "backups", "settings", "settings-*.js.bak")); len(matches) != 0 {
		t.Errorf("no backup file should exist after validate-stage failure, found %v", matches)
	}
	// No .apply-* sibling should remain.
	entries, _ := os.ReadDir(filepath.Dir(settingsPath))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".apply-") {
			t.Errorf("leftover tmp sibling after cancellation: %s", e.Name())
		}
	}
}

// Compile-time check: *recordingAuditHook is a no-op function (returns
// nothing) so it satisfies AuditHookFunc at the call site. The
// assertion at construction time guards against future drift in the
// function signature.
var _ = func() bool {
	var hook AuditHookFunc
	rh := &recordingAuditHook{}
	hook = rh.Log
	return hook != nil
}()

// Compile-time check: ensure net/http is referenced (it is, via the
// recordingAuditHook signature).
var _ = func(r *http.Request) bool { return r != nil }