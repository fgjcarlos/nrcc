// Package service — slice C of issue #758.
//
// Slice A's ApplyService.Apply already does a revision check inside
// its validate stage (it calls RevisionMatches against the live
// settings.js revision). Slice C exposes two new entry points that
// share the same precondition but also classify the conflict as a
// typed, recoverable error so the HTTP handlers can render a
// structured 409 envelope with both the live and provided revisions.
//
// Integration point with slice B
// ==============================
//
// RevisionConflictError is defined here — NOT in slice B's
// source_revision.go — because slice B only exposes the sentinel
// ErrSourceRevisionMismatch + the helpers FingerprintSource /
// RevisionMatches. The typed error carries the live vs. provided
// fingerprints so the handler envelope can echo them back. Slice
// C is the slice that produces the envelope, so the typed error
// lives next to the helper that emits it.
//
// errors.Is(err, ErrSourceRevisionMismatch) still matches every
// RevisionConflictError (see the Unwrap/Is methods below) so the
// existing handler in internal/handler/settings.go keeps working
// without modification — slice C's only obligation is to upgrade
// the sentinel to a richer envelope at the new endpoint.

package service

import (
	"context"
	"fmt"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// RevisionConflictError is returned by ApplyCoordinator.Apply when
// the live settings.js revision differs from the caller's
// ExpectedRevision. The typed error exposes both sides of the
// comparison so the HTTP handler can return a 409 envelope shaped
// as `{liveRevision, provided}`. Use errors.As to recover the
// fields; errors.Is(err, ErrSourceRevisionMismatch) also returns
// true so existing handlers and tests keep matching.
type RevisionConflictError struct {
	// Live is the SourceRevision that ApplyCoordinator read from
	// the on-disk settings.js at the moment of the apply attempt.
	Live model.SourceRevision
	// Provided is the SourceRevision the caller passed in
	// ApplyRequest.ExpectedRevision. Fingerprint is the only field
	// the operator normally populates; Algorithm defaults to
	// SourceRevisionAlgorithm when omitted.
	Provided model.SourceRevision
	// Path is the on-disk settings.js path the conflict was
	// detected on. Surfaced for audit meta.
	Path string
}

// Error renders the conflict as "settings.js revision conflict:
// live=<liveFingerprint> provided=<providedFingerprint>". The
// message intentionally omits algorithm metadata because the
// conflict is keyed on the digest, not the algorithm.
func (e *RevisionConflictError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("settings.js revision conflict: live=%s provided=%s",
		e.Live.Fingerprint, e.Provided.Fingerprint)
}

// Is lets errors.Is(err, ErrSourceRevisionMismatch) match a
// *RevisionConflictError so the typed error can replace the
// sentinel at the new slice C endpoints without breaking callers
// that still compare against the sentinel. We deliberately do NOT
// delegate to e.Unwrap() here — the chain would reach
// ErrSourceRevisionMismatch via the wrapped Cause below, but
// matching against *RevisionConflictError must be the fastest path.
func (e *RevisionConflictError) Is(target error) bool {
	return target == ErrSourceRevisionMismatch
}

// Unwrap exposes ErrSourceRevisionMismatch so errors.Is unwraps
// naturally and any future wrapping in *ApplyError carries the
// typed conflict along the chain.
func (e *RevisionConflictError) Unwrap() error {
	return ErrSourceRevisionMismatch
}

// CheckRevisionPrecondition is the IfMatch-style guard the
// ApplyCoordinator runs before delegating to ApplyService.Apply.
// It reads the live settings.js document via configSvc and
// returns a *RevisionConflictError when the caller's expected
// revision does not match the live one.
//
// Empty-expectation bypass
// ------------------------
// When req.Expected.Fingerprint is empty the function returns nil
// without reading the live revision. This preserves the legacy
// "no expectation" semantics documented on RevisionMatches: a
// caller that has not yet been wired to capture the revision
// (e.g. an older operator dashboard) is never blocked by a
// mismatch. Slice C's HTTP handlers ALWAYS capture the revision
// from the GET response, so the empty path is for forward
// compatibility and for the coordinator's own synthetic callers.
//
// Cancellation
// ------------
// ctx is honoured BEFORE the live read so a cancelled apply does
// not perform any I/O. ctx is NOT threaded into
// ConfigService.GetRawSettings because that helper has its own
// boundary contract (a snapshot read of a small file); a cancel
// that lands mid-read just returns the partial error the
// underlying os.ReadFile surfaces, which the handler maps to
// SETTINGS_ERROR.
//
// The function deliberately does NOT emit an audit event. The
// caller (ApplyCoordinator) is responsible for the
// apply.failure audit discipline so a single emit covers both
// the structured path and the raw path without duplication.
func CheckRevisionPrecondition(ctx context.Context, configSvc *ConfigService, req ApplyRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if req.Expected.Fingerprint == "" {
		// Empty expectation: backward-compat bypass. The caller is
		// opting out of the precondition for this apply; do not
		// penalise them with a live read.
		return nil
	}
	live, err := configSvc.GetRawSettings()
	if err != nil {
		return fmt.Errorf("read live settings for revision check: %w", err)
	}
	if RevisionMatches(req.Expected, live.Revision) {
		return nil
	}
	return &RevisionConflictError{
		Live:     live.Revision,
		Provided: req.Expected,
		Path:     req.Path,
	}
}