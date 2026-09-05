// Package service — slice A of issue #758 introduces an apply pipeline
// that orchestrates validate → backup → atomic write → audit for every
// settings.js mutation. This file owns the orchestrator (ApplyService)
// and the typed error (ApplyError).
//
// Pipeline contract
// =================
//
// ApplyService.Apply(ctx, req) executes four stages in order:
//
//   1. validate    — parse the candidate, reject plaintext adminAuth
//                    passwords, refuse a stale Expected revision.
//   2. backup      — copy the current settings.js (if any) to a
//                    timestamped sibling inside the requested backup
//                    directory.
//   3. write       — atomic replace of settings.js (see apply_atomic.go).
//   4. audit       — emit apply.start, apply.backup, apply.write, and
//                    either apply.success or apply.failure carrying the
//                    typed ApplyError.Stage.
//
// On failure at any stage ApplyError{Stage, Cause} is returned. The
// Cause is wrapped via fmt.Errorf("%w", ...) so errors.Is / errors.As
// still match typed sentinels (ErrSourceRevisionMismatch, ErrSandboxTimeout).
//
// Slice B will reuse this orchestrator and add the ProcessManager
// restart + readiness verification between the write and success
// stages. Slice C will plug the HTTP handlers in. Slice A only delivers
// the core transaction and the audit-emission discipline required for
// observability of the core transaction.

package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// ApplyStage identifies the phase of an Apply transaction that failed.
// Callers use it with errors.As to branch on which step blew up.
type ApplyStage string

const (
	ApplyStageValidate ApplyStage = "validate"
	ApplyStageBackup   ApplyStage = "backup"
	ApplyStageWrite    ApplyStage = "write"
	ApplyStageAudit    ApplyStage = "audit"
)

// ApplyError is the typed error returned by ApplyService.Apply. Stage
// reports which phase failed; Cause is the wrapped underlying error.
// Use errors.As to recover ApplyError and inspect the Stage field; use
// errors.Is to match against typed sentinels (ErrSourceRevisionMismatch,
// ErrSandboxTimeout) that propagate through Cause.
type ApplyError struct {
	Stage   ApplyStage
	Cause   error
	Path    string
	Request ApplyRequest
}

// Error renders the typed error as "apply <stage>: <cause>". The cause
// carries the original error message; the stage names the failing phase.
func (e *ApplyError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("apply %s: %v", e.Stage, e.Cause)
}

// Unwrap exposes Cause for errors.Is / errors.As traversal. A nil
// receiver returns nil so callers can use this in error chains safely.
func (e *ApplyError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is lets errors.Is distinguish ApplyErrors by stage. It only handles
// the target == *ApplyError case; other targets fall through to the
// default Unwrap chain.
func (e *ApplyError) Is(target error) bool {
	var other *ApplyError
	if !errors.As(target, &other) {
		return false
	}
	if other == nil || e == nil {
		return false
	}
	return e.Stage == other.Stage
}

// ApplyRequest is the input payload for ApplyService.Apply. The fields
// are normally populated by the HTTP handler from the live
// SettingsDocument returned by ConfigService.GetRawSettings:
//
//   - Path        : doc.Path (the on-disk settings.js location)
//   - BackupDir   : doc.Path's parent + "/backups/settings" (the
//                   established backup directory) or operator-configured
//                   override.
//   - Expected    : doc.Revision (the fingerprint at the time of read)
//   - Content     : the operator's edited settings.js body.
//
// Request is optional; when non-nil its IP and User-Agent feed the audit
// record. Actor populates the audit Actor field.
type ApplyRequest struct {
	Path         string
	Content      string
	Expected     model.SourceRevision
	BackupDir    string
	Actor        string
	Request      *http.Request
	Capabilities model.ConfigurationCapabilities
}

// ApplyResult captures the outcome of a successful Apply. Revision is
// the freshly-fingerprinted state of the bytes just persisted so the
// caller can use it as the next Expected value without re-reading the
// file. BackupPath is the location of the previous file (empty when no
// previous file existed). Diff is the redacted diff suitable for the
// audit log.
type ApplyResult struct {
	Revision   model.SourceRevision
	BackupPath string
	Diff       string
	Path       string
}

// AuditHookFunc is the function signature ApplyService uses to emit
// audit events. It mirrors audit.Service.Log but is defined locally to
// avoid an import cycle (audit imports middleware which imports
// service). Slice B will pass `(*audit.Service).Log` directly via a
// thin adapter; tests substitute a recording fake.
//
// The hook may be nil; ApplyService treats that as "audit disabled"
// and continues without error. Audit emission is observability, not a
// critical-path dependency — a failed Log never aborts the transaction.
type AuditHookFunc func(r *http.Request, actor, action, target, result string, meta map[string]string)

// ApplyService orchestrates the validate → backup → atomic write →
// audit transaction that rewrites Node-RED settings.js. Slice A of #758
// — the HTTP handlers and ProcessManager restart wiring belong to
// slices B and C.
type ApplyService struct {
	configSvc *ConfigService
	auditHook AuditHookFunc
	clock     func() time.Time
}

// NewApplyService constructs an ApplyService wired to configSvc. The
// audit hook may be nil; Apply tolerates that and emits no events.
// The clock defaults to time.Now().UTC and is overridable via WithClock
// for tests that pin the backup filename.
func NewApplyService(configSvc *ConfigService, auditHook AuditHookFunc) *ApplyService {
	return &ApplyService{
		configSvc: configSvc,
		auditHook: auditHook,
		clock:     func() time.Time { return time.Now().UTC() },
	}
}

// WithClock replaces the clock function. It returns the receiver so
// callers can chain configuration in tests.
func (s *ApplyService) WithClock(clock func() time.Time) *ApplyService {
	if clock != nil {
		s.clock = clock
	}
	return s
}

// Apply executes the settings.js apply transaction. See the file header
// for the four-stage contract.
//
// Boundary validation: an empty Path or BackupDir is rejected at stage
// "validate" with a typed ApplyError so the orchestrator's pre-conditions
// are observable to the caller.
//
// Audit emission: every stage emits at most one event. The start event
// is emitted BEFORE any stage work; failure events are emitted AFTER
// the typed error is constructed so the audit record reflects the
// exact Stage the caller receives. Success events are emitted AFTER
// the on-disk state is durable.
func (s *ApplyService) Apply(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	// Apply.start is emitted FIRST so observability captures every
	// transaction attempt — including boundary-validation failures.
	s.emit(req, "apply.start", ApplyStageValidate, "ok", nil)

	if req.Path == "" {
		err := &ApplyError{
			Stage: ApplyStageValidate,
			Cause: errors.New("apply path is required"),
			Path:  "",
		}
		s.emit(req, "apply.failure", ApplyStageValidate, "error", err.Cause)
		return ApplyResult{}, err
	}
	if req.BackupDir == "" {
		err := &ApplyError{
			Stage: ApplyStageValidate,
			Cause: errors.New("apply backup directory is required"),
			Path:  req.Path,
		}
		s.emit(req, "apply.failure", ApplyStageValidate, "error", err.Cause)
		return ApplyResult{}, err
	}
	if err := validateSettingsPath(req.Path); err != nil {
		ae := &ApplyError{Stage: ApplyStageValidate, Cause: err, Path: req.Path, Request: req}
		s.emit(req, "apply.failure", ApplyStageValidate, "error", err)
		return ApplyResult{}, ae
	}
	if err := validateSettingsPath(req.BackupDir); err != nil {
		ae := &ApplyError{Stage: ApplyStageValidate, Cause: err, Path: req.Path, Request: req}
		s.emit(req, "apply.failure", ApplyStageValidate, "error", err)
		return ApplyResult{}, ae
	}

	// --- Stage 1: validate -------------------------------------------------
	if err := s.applyValidate(ctx, req); err != nil {
		s.emit(req, "apply.failure", ApplyStageValidate, "error", err)
		return ApplyResult{}, &ApplyError{Stage: ApplyStageValidate, Cause: err, Path: req.Path, Request: req}
	}

	// --- Stage 2: backup ---------------------------------------------------
	backupPath, liveContent, err := s.applyBackup(req)
	if err != nil {
		s.emit(req, "apply.failure", ApplyStageBackup, "error", err)
		return ApplyResult{}, &ApplyError{Stage: ApplyStageBackup, Cause: err, Path: req.Path, Request: req}
	}
	s.emit(req, "apply.backup", ApplyStageBackup, "ok", nil)

	// --- Stage 3: atomic write --------------------------------------------
	if err := AtomicWriteSettings(ctx, req.Path, req.Content); err != nil {
		s.emit(req, "apply.failure", ApplyStageWrite, "error", err)
		return ApplyResult{}, &ApplyError{Stage: ApplyStageWrite, Cause: err, Path: req.Path, Request: req}
	}
	s.emit(req, "apply.write", ApplyStageWrite, "ok", nil)

	// --- Stage 4: success --------------------------------------------------
	result := ApplyResult{
		Revision:   FingerprintSource(req.Content),
		BackupPath: backupPath,
		Diff:       RedactDiff(liveContent, req.Content),
		Path:       req.Path,
	}
	s.emit(req, "apply.success", ApplyStageWrite, "ok", nil)
	return result, nil
}

// applyValidate runs the parse + adminAuth + revision checks. The
// revision check uses the same FingerprintSource / RevisionMatches pair
// delivered by #779 so this slice does not introduce a new fingerprint
// mechanism.
func (s *ApplyService) applyValidate(ctx context.Context, req ApplyRequest) error {
	// Honour the caller's context cancellation BEFORE we touch the
	// filesystem.
	if err := ctx.Err(); err != nil {
		return err
	}
	liveDoc, err := s.configSvc.GetRawSettings()
	if err != nil {
		return fmt.Errorf("read live settings: %w", err)
	}
	if !RevisionMatches(req.Expected, liveDoc.Revision) {
		return ErrSourceRevisionMismatch
	}
	parsed, err := s.configSvc.parseConfigFromContent(req.Content)
	if err != nil {
		// ErrSandboxTimeout propagates so the handler can map it to
		// SETTINGS_TIMEOUT. Other sandbox errors fall through to the
		// same return path as a parse failure.
		return fmt.Errorf("parse candidate settings: %w", err)
	}
	if err := validateAdminAuthPasswords(parsed); err != nil {
		return fmt.Errorf("validate adminAuth: %w", err)
	}
	return nil
}

// applyBackup copies the current settings.js (if any) to a timestamped
// sibling inside req.BackupDir. The returned liveContent is the bytes
// that were just backed up; the caller uses them to render the
// redacted diff. When the live file is missing the backup is skipped
// and the function returns "", "" so the diff collapses to "all lines
// are additions".
func (s *ApplyService) applyBackup(req ApplyRequest) (backupPath string, liveContent string, err error) {
	if err := os.MkdirAll(req.BackupDir, 0o750); err != nil {
		return "", "", fmt.Errorf("create backup dir %s: %w", req.BackupDir, err)
	}
	// Read the live content (for the diff). os.ReadFile errors only when
	// the file is missing or unreadable; we treat missing as the
	// first-write case and surface any other error.
	liveBytes, readErr := os.ReadFile(req.Path)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("read live settings: %w", readErr)
	}
	liveContent = string(liveBytes)

	// Build a timestamped backup filename. The format mirrors the one
	// used by ConfigService.backupSettingsFile so the existing backup
	// tooling continues to find the new files.
	stamp := s.clock().Format("20060102-150405")
	backupName := fmt.Sprintf("settings-%s.js.bak", stamp)
	target := filepath.Join(req.BackupDir, backupName)
	// #nosec G703 -- target is built from the operator-validated BackupDir (already passed validateSettingsPath) plus a constant filename; not request-derived.
	if err := os.WriteFile(target, liveBytes, 0o600); err != nil {
		return "", liveContent, fmt.Errorf("write backup %s: %w", target, err)
	}
	return target, liveContent, nil
}

// emit writes a single audit event. The function never returns an
// error; the audit subsystem reports its own degradation through
// Outcome, and ApplyService does not gate the transaction on the
// audit success (audit is observability, not a critical-path
// dependency).
func (s *ApplyService) emit(req ApplyRequest, action string, stage ApplyStage, result string, cause error) {
	if s.auditHook == nil {
		return
	}
	meta := RedactedCapabilityMeta(req.Capabilities)
	meta["apply_stage"] = string(stage)
	if req.Path != "" {
		meta["apply_path"] = filepath.Base(req.Path)
	}
	if req.BackupDir != "" {
		meta["apply_backup_dir"] = filepath.Base(req.BackupDir)
	}
	if cause != nil {
		meta["apply_error"] = cause.Error()
	}
	s.auditHook(req.Request, req.Actor, action, req.Path, result, meta)
}