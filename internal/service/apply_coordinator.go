// Package service — slice C of issue #758.
//
// Slice A delivered ApplyService.Apply (validate → backup →
// atomic write → audit). Slice B added the adapter restart +
// readiness + rollback primitives that slot in between the
// write and success stages. Slice C is the wiring layer that
// the HTTP handlers go through: it provides single-flight
// concurrency control per settings.js path, an explicit
// IfMatch-style revision precondition (see apply_revision.go)
// and the typed error envelope the new endpoints return.
//
// Why a coordinator?
// ==================
//
// ApplyService.Apply is a single transaction that takes the
// settings.js write lock indirectly (via the atomic-rename
// helper). Two overlapping apply calls on the same settings.js
// path would race the backup stage: the second call would
// back up the just-written first-call content, leaving the
// backup directory with no copy of the pre-first-call state and
// silently rolling the rollback target forward. Slice C's
// coordinator refuses overlapping calls with a typed error so
// the second caller can surface "another apply is in progress"
// instead of corrupting the audit trail.
//
// Keyed concurrency
// -----------------
//
// ApplyCoordinator holds a sync.Mutex around a map keyed on
// ApplyRequest.Path. A second call that arrives while the first
// is still running returns ErrApplyInFlight without ever
// invoking ApplyService.Apply. Different settings.js paths
// (rare — Node-RED only writes one — but possible in tests)
// proceed in parallel. The keyed approach keeps the coordinator
// stateless from the caller's point of view: there is no global
// "apply lock" that would unnecessarily serialise unrelated
// transactions.
//
// Cancellation propagation
// -------------------------
//
// The coordinator honours ctx cancellation the same way
// ApplyService does. A cancelled Apply call releases its
// in-flight slot synchronously via the deferred release so the
// next caller does not get stuck behind a cancelled transaction.
//
// Audit discipline
// ----------------
//
// apply.start is emitted with failure_stage=start before any
// I/O. apply.success is emitted on a clean return.
// apply.failure is emitted on every typed ApplyError AND on a
// *RevisionConflictError with failure_stage=validate so the
// failure envelope mirrors the slice A contract while carrying
// the slice C typed error. apply.failure on ErrApplyInFlight is
// emitted with failure_stage=start (no stage work happened) so
// the audit reader can distinguish "rejected before work" from
// "rejected after validate".

package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
)

// ErrApplyInFlight is returned by ApplyCoordinator.Apply when
// another transaction is already running against the same
// settings.js path. Use errors.Is to detect.
var ErrApplyInFlight = errors.New("apply: another transaction is in progress for this settings.js path")

// applyFlight is the per-path in-flight slot the coordinator
// maintains. The channel is closed on completion so concurrent
// callers that choose to wait (none today) could observe
// completion, but the typed-error fast path means most callers
// never block.
type applyFlight struct {
	done chan struct{}
}

// ApplyCoordinator serialises overlapping Apply transactions on
// the same settings.js path. Construct via NewApplyCoordinator
// and reuse for the lifetime of the server; the coordinator is
// stateless beyond its in-flight map.
type ApplyCoordinator struct {
	apply   *ApplyService
	mu      sync.Mutex
	flights map[string]*applyFlight
}

// NewApplyCoordinator wraps apply in a single-flight guard. The
// apply service must already be wired with the configSvc + audit
// hook the coordinator delegates to.
func NewApplyCoordinator(apply *ApplyService) *ApplyCoordinator {
	return &ApplyCoordinator{
		apply:   apply,
		flights: make(map[string]*applyFlight),
	}
}

// Apply runs the IfMatch precondition, then delegates to
// ApplyService.Apply with single-flight guard keyed on
// req.Path. See the file header for the contract.
//
// Failure semantics:
//
//   - Empty Path → *ApplyError{Stage: validate} (the coordinator
//     does not synthesise a new error type — the existing
//     ApplyError contract is the source of truth).
//   - Revision mismatch → *RevisionConflictError with
//     failure_stage=validate in the audit meta.
//   - Overlapping transaction on the same path → ErrApplyInFlight
//     with failure_stage=start in the audit meta; no Apply work
//     happened so the typed error is NOT wrapped in *ApplyError.
//   - ApplyService.Apply error → returned verbatim (typed
//     *ApplyError or wrapped ErrSourceRevisionMismatch); audit
//     event has failure_stage matching the *ApplyError.Stage.
//
// ctx cancellation propagates through to ApplyService.Apply and
// also releases the per-path slot synchronously via the
// deferred release so a cancelled apply does not lock out the
// next caller.
func (c *ApplyCoordinator) Apply(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	if c.apply == nil {
		return ApplyResult{}, errors.New("apply coordinator: underlying apply service is nil")
	}
	if req.Path == "" {
		err := &ApplyError{
			Stage: ApplyStageValidate,
			Cause: errors.New("apply path is required"),
		}
		c.emitFailure(req, "validate", err)
		return ApplyResult{}, err
	}

	// Capture the slot BEFORE running the revision check so two
	// simultaneous calls with different Expected revisions still
	// serialise — the second's GetRawSettings would otherwise see
	// the first's write half-completed.
	flight, err := c.acquire(req.Path)
	if err != nil {
		c.emitFailure(req, "start", err)
		return ApplyResult{}, err
	}
	defer c.release(req.Path, flight)

	// Revision precondition: run BEFORE ApplyService.Apply so
	// we never copy a stale bytes-set into a backup.
	if err := CheckRevisionPrecondition(ctx, c.apply.configSvc, req); err != nil {
		var conflict *RevisionConflictError
		if errors.As(err, &conflict) {
			c.emitFailure(req, "validate", conflict)
			return ApplyResult{}, conflict
		}
		// ctx.Err() or read errors — surface as a validate-stage
		// typed error so the audit envelope is consistent.
		ae := &ApplyError{Stage: ApplyStageValidate, Cause: err, Path: req.Path, Request: req}
		c.emitFailure(req, "validate", ae)
		return ApplyResult{}, ae
	}

	result, err := c.apply.Apply(ctx, req)
	if err != nil {
		// ApplyService.Apply already emitted the apply.failure
		// audit event with its own meta (apply_stage). We do NOT
		// double-emit; the slice A discipline remains the source
		// of truth. The slice C failure_stage meta is added by
		// re-emitting only for the *RevisionConflictError branch
		// above (where ApplyService never ran).
		return ApplyResult{}, err
	}
	c.emitSuccess(req)
	return result, nil
}

// acquire grabs the per-path slot. Returns ErrApplyInFlight if
// another transaction is already running on path.
func (c *ApplyCoordinator) acquire(path string) (*applyFlight, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.flights[path]; ok {
		// Another transaction is in flight. We deliberately do
		// not wait: the typed ErrApplyInFlight surfaces the
		// conflict at the HTTP layer instead of holding the
		// second caller in a queue it cannot inspect.
		select {
		case <-existing.done:
			// Race: the previous transaction finished between
			// our map lookup and the wait decision. Fall
			// through and acquire the freshly-released slot.
		default:
			return nil, ErrApplyInFlight
		}
	}
	flight := &applyFlight{done: make(chan struct{})}
	c.flights[path] = flight
	return flight, nil
}

// release closes the flight channel and removes the slot from
// the map. Always paired with acquire via defer.
func (c *ApplyCoordinator) release(path string, flight *applyFlight) {
	c.mu.Lock()
	if c.flights[path] == flight {
		delete(c.flights, path)
	}
	done := flight.done
	c.mu.Unlock()
	close(done)
}

// emitFailure writes the apply.failure audit event with the
// failure_stage meta key the slice C acceptance criteria
// require. We deliberately use the slice A AuditHookFunc
// signature (see apply.go) so a single audit consumer (the
// production audit.Service OR the recordingAuditHook in tests)
// renders the slice C envelope identically to slice A's.
func (c *ApplyCoordinator) emitFailure(req ApplyRequest, failureStage string, cause error) {
	if c.apply == nil || c.apply.auditHook == nil {
		return
	}
	meta := RedactedCapabilityMeta(req.Capabilities)
	meta["failure_stage"] = failureStage
	if req.Path != "" {
		meta["apply_path"] = filepath.Base(req.Path)
	}
	if req.BackupDir != "" {
		meta["apply_backup_dir"] = filepath.Base(req.BackupDir)
	}
	if cause != nil {
		meta["apply_error"] = cause.Error()
	}
	c.apply.auditHook(req.Request, req.Actor, "apply.failure", req.Path, "error", meta)
}

// emitSuccess writes apply.success with failure_stage=ok so the
// audit reader can tell a clean transaction from a start-only
// record.
func (c *ApplyCoordinator) emitSuccess(req ApplyRequest) {
	if c.apply == nil || c.apply.auditHook == nil {
		return
	}
	meta := RedactedCapabilityMeta(req.Capabilities)
	meta["failure_stage"] = "ok"
	if req.Path != "" {
		meta["apply_path"] = filepath.Base(req.Path)
	}
	if req.BackupDir != "" {
		meta["apply_backup_dir"] = filepath.Base(req.BackupDir)
	}
	c.apply.auditHook(req.Request, req.Actor, "apply.success", req.Path, "ok", meta)
}