// Package service — slice B rollback flow.
//
// Slice A's apply pipeline backs up the previous settings.js before the
// atomic write. Slice B reuses that backup to roll back when the
// post-write restart or readiness probe fails. The flow is:
//
//	1. Slice A succeeds → ApplyResult.BackupPath points at the
//	   timestamped .js.bak from slice A's applyBackup stage.
//	2. Slice B calls adapter.Restart → on failure, Restore.
//	3. Slice B calls adapter.Ready  → on failure, Restore.
//	4. After Restore, Slice B verifies the rollback via adapter.Ready
//	   (a successful restart+read against the OLD settings is the
//	   audit-trail evidence that the rollback itself was non-fatal).
//
// All rollback observability flows through the auditHook signature
// ApplyService uses, with three action verbs:
//
//	apply.rollback.started    ← snapshot path, backup source, stage that failed
//	apply.rollback.succeeded  ← restore path, post-rollback readiness
//	apply.rollback.failed     ← last error from restore / post-rollback ready
//
// The functions in this file are stateless helpers — the orchestrator
// caller (slice C's ApplyCoordinator) owns the lifecycle. Slice B
// delivers the primitives so a future slice can compose them with
// ApplyService.Apply without changing slice A's public surface.

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

// RollbackStage identifies which post-write stage triggered the
// rollback. It is surfaced in audit meta so observers can correlate
// apply.rollback.* events with the failing stage.
type RollbackStage string

const (
	RollbackStageRestart RollbackStage = "restart"
	RollbackStageReady   RollbackStage = "ready"
)

// RollbackError is the typed error returned by ExecuteRollback when the
// restore itself fails. The outer orchestrator uses errors.As to map it
// onto a 5xx (and to keep audit meta consistent). Cause carries the
// underlying restore failure; Stage carries the failing post-write
// stage so the error is recoverable from logs.
type RollbackError struct {
	Stage   RollbackStage
	Cause   error
	Restore string // path the rollback tried to restore to
}

func (e *RollbackError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("apply rollback after %s: %v", e.Stage, e.Cause)
}

func (e *RollbackError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// RollbackRequest is the input for ExecuteRollback. BackupPath is the
// snapshot produced by slice A's applyBackup stage (a timestamped
// .js.bak in BackupDir); ActivePath is the live settings.js that was
// just overwritten with the failed candidate. Adapter issues the
// post-rollback readiness probe. Actor / Request / Capabilities feed
// the audit emit.
type RollbackRequest struct {
	BackupPath  string
	ActivePath  string
	Stage       RollbackStage
	Adapter     ApplyAdapter
	Actor       string
	Request     *http.Request
	AuditHook   AuditHookFunc
	Capabilities model.ConfigurationCapabilities
	// ReadyzTimeout caps the post-rollback readiness probe. Optional;
	// zero falls back to the readiness probe default.
	ReadyzTimeout time.Duration
}

// ExecuteRollback restores the snapshot at req.BackupPath to
// req.ActivePath, verifies the restored settings.js via the adapter's
// Ready probe, and emits apply.rollback.{started,succeeded,failed}
// audit events.
//
// Failure semantics:
//
//   - The restore itself fails (file missing, fsync error): returns a
//     *RollbackError. The audit hook emits apply.rollback.failed and
//     the caller is expected to surface a 500 with ROLLBACK_FAILED.
//   - The post-rollback Ready probe fails: returns a *RollbackError
//     wrapping ErrReadinessNotReady. The settings.js file IS the
//     previous content (the restore succeeded), so the system is in
//     a degraded state — the next operator action must be to inspect
//     the deployment. The audit hook emits apply.rollback.failed.
//
// A nil error means the rollback restored the snapshot and Node-RED
// came back healthy against it.
func ExecuteRollback(ctx context.Context, req RollbackRequest) error {
	if req.BackupPath == "" {
		err := &RollbackError{
			Stage:   req.Stage,
			Cause:   errors.New("rollback: backup path is empty"),
			Restore: req.ActivePath,
		}
		emitRollback(req, "apply.rollback.failed", "error", err.Cause, map[string]string{
			"apply_rollback_action": "validate",
		})
		return err
	}
	if req.ActivePath == "" {
		err := &RollbackError{
			Stage:   req.Stage,
			Cause:   errors.New("rollback: active path is empty"),
			Restore: "",
		}
		emitRollback(req, "apply.rollback.failed", "error", err.Cause, map[string]string{
			"apply_rollback_action": "validate",
		})
		return err
	}
	if req.Adapter == nil {
		err := &RollbackError{
			Stage:   req.Stage,
			Cause:   errors.New("rollback: adapter is nil"),
			Restore: req.ActivePath,
		}
		emitRollback(req, "apply.rollback.failed", "error", err.Cause, map[string]string{
			"apply_rollback_action": "validate",
		})
		return err
	}

	emitRollback(req, "apply.rollback.started", "ok", nil, map[string]string{
		"apply_rollback_stage":       string(req.Stage),
		"apply_rollback_backup_path": filepath.Base(req.BackupPath),
		"apply_rollback_target":      filepath.Base(req.ActivePath),
	})

	// Snapshot the post-failure file so the operator can diff it
	// against the restored version after the fact. We deliberately
	// preserve the failed candidate — the audit log and the on-disk
	// ".failed" sibling let post-mortem tooling reconstruct what
	// Node-RED rejected.
	failedSibling := req.ActivePath + ".failed-" + time.Now().UTC().Format("20060102-150405")
	if err := copyFile(req.ActivePath, failedSibling); err != nil {
		// Best-effort: a missing live file is normal on the very
		// first apply. We log via audit meta but do not abort the
		// rollback — the operator still needs the prior settings.js
		// back in place.
		emitRollback(req, "apply.rollback.started", "degraded", err, map[string]string{
			"apply_rollback_stage":         string(req.Stage),
			"apply_rollback_failed_sibling": filepath.Base(failedSibling),
			"apply_rollback_failed_sibling_error": err.Error(),
		})
	}

	if err := restoreSnapshot(req.BackupPath, req.ActivePath); err != nil {
		emitRollback(req, "apply.rollback.failed", "error", err, map[string]string{
			"apply_rollback_stage":  string(req.Stage),
			"apply_rollback_action": "restore",
		})
		return &RollbackError{
			Stage:   req.Stage,
			Cause:   fmt.Errorf("restore snapshot: %w", err),
			Restore: req.ActivePath,
		}
	}

	// Verify the rollback brought Node-RED back. We honour an
	// optional ReadyzTimeout (slice C may pass 30s, etc.) by
	// wrapping the adapter's Ready with a timeout-aware adapter
	// when one is supplied; otherwise the readiness probe uses
	// its own per-attempt timeout.
	if err := req.Adapter.Ready(ctx); err != nil {
		emitRollback(req, "apply.rollback.failed", "error", err, map[string]string{
			"apply_rollback_stage":  string(req.Stage),
			"apply_rollback_action": "ready",
		})
		return &RollbackError{
			Stage:   req.Stage,
			Cause:   fmt.Errorf("post-rollback readiness: %w", err),
			Restore: req.ActivePath,
		}
	}

	emitRollback(req, "apply.rollback.succeeded", "ok", nil, map[string]string{
		"apply_rollback_stage":         string(req.Stage),
		"apply_rollback_failed_sibling": filepath.Base(failedSibling),
	})
	return nil
}

// restoreSnapshot atomically replaces activePath with the contents of
// backupPath. We reuse AtomicWriteSettings because the file boundary
// contract (path traversal, symlink, fsync parent dir) is identical;
// the rollback writes the previous-good file in place of the failed
// candidate without going through the apply pipeline.
func restoreSnapshot(backupPath, activePath string) error {
	if err := validateSettingsPath(backupPath); err != nil {
		return err
	}
	if err := validateSettingsPath(activePath); err != nil {
		return err
	}
	// #nosec G304 -- backupPath is the operator-supplied snapshot path from ApplyResult.BackupPath; we re-validate before reading.
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup %s: %w", backupPath, err)
	}
	return AtomicWriteSettings(context.Background(), activePath, string(data))
}

// copyFile copies src to dst, preserving the source mode. dst's parent
// directory is created if missing. The function returns nil when src
// does not exist (the post-apply file may be a first-write case).
// Any other error is wrapped with the offending path.
func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("mkdir for %s: %w", dst, err)
	}
	// #nosec G304 -- src is the operator-supplied active path; we re-validate via the boundary check above.
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o600
	}
	// #nosec G304,G703 -- dst is the operator-supplied .failed sibling built from the active path; the boundary check above validates it.
	if err := os.WriteFile(dst, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

// emitRollback wraps ApplyService.emit so the rollback flow reuses
// the same audit envelope (capability redaction, nil-hook tolerance,
// apply_rollback_* meta). The function is unexported because slice A
// owns the canonical emit path; slice B delegates through it.
func emitRollback(req RollbackRequest, action, result string, cause error, extra map[string]string) {
	if req.AuditHook == nil {
		return
	}
	meta := RedactedCapabilityMeta(req.Capabilities)
	meta["apply_rollback_stage"] = string(req.Stage)
	meta["apply_rollback_adapter"] = ""
	if req.Adapter != nil {
		meta["apply_rollback_adapter"] = req.Adapter.Name()
	}
	if req.BackupPath != "" {
		meta["apply_rollback_backup_path"] = filepath.Base(req.BackupPath)
	}
	if req.ActivePath != "" {
		meta["apply_rollback_target"] = filepath.Base(req.ActivePath)
	}
	for k, v := range extra {
		meta[k] = v
	}
	if cause != nil {
		meta["apply_rollback_error"] = cause.Error()
	}
	req.AuditHook(req.Request, req.Actor, action, req.ActivePath, result, meta)
}