// Package service — slice B of issue #757 introduces the revision
// fingerprint that lets the backend refuse a save when the active
// settings.js changed externally between the operator's read and their
// write. The fingerprint is intentionally separate from the source-
// preserving patch contract delivered in slice A: revision detection only
// needs the bytes of the file, not its parseable shape, so the two
// concerns can land independently and merge in any order.
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// SourceRevisionAlgorithm is the fingerprint algorithm this build uses to
// identify revisions of settings.js source. It is surfaced on
// model.SourceRevision so a future algorithm change is self-describing on
// the wire contract (callers that pin a different algorithm receive a
// mismatch instead of a silent collision).
const SourceRevisionAlgorithm = "sha256"

// ErrSourceRevisionMismatch is returned by SaveRawSettingsWithRevision when
// the caller's expected revision does not match the current on-disk
// fingerprint. The handler surfaces this as HTTP 409 Conflict so the
// operator can re-read the file, re-render their edits against the new
// revision and retry instead of overwriting an external change.
//
// Slice deferral: the frontend revision-echo UI (returning the new
// revision in the response payload and re-binding it on the Save button)
// is out of scope for this slice.
var ErrSourceRevisionMismatch = errors.New("settings.js source revision changed since last read")

// FingerprintSource computes a SourceRevision for content. The fingerprint
// is a SHA-256 hex digest of the raw bytes — every byte, including
// whitespace outside the module.exports body, comments and trailing
// newlines, participates. Two sources that differ by even one byte produce
// different revisions; two sources that are byte-for-byte identical produce
// the same revision regardless of when the fingerprint was taken.
//
// CapturedAt is recorded in UTC RFC 3339 for operator-visible audit logs
// only. Equality MUST be determined by Algorithm+Fingerprint, never by
// CapturedAt, so that two revisions captured moments apart still compare
// equal when their bytes are identical.
func FingerprintSource(content string) model.SourceRevision {
	sum := sha256.Sum256([]byte(content))
	return model.SourceRevision{
		Fingerprint: hex.EncodeToString(sum[:]),
		Algorithm:   SourceRevisionAlgorithm,
		CapturedAt:  time.Now().UTC().Format(time.RFC3339),
	}
}

// RevisionMatches reports whether expected equals current.
//
// An empty expected.Fingerprint is treated as "no expectation" and always
// matches. This keeps the legacy SaveRawSettings path — and any caller
// that has not yet been wired to capture the revision — backward
// compatible: a missing revision never blocks a save.
//
// A non-empty expected.Algorithm that disagrees with current.Algorithm is
// treated as a mismatch even when the Fingerprint happens to collide, so
// rolling the algorithm forward is safe instead of becoming a silent
// collision surface.
func RevisionMatches(expected, current model.SourceRevision) bool {
	if expected.Fingerprint == "" {
		return true
	}
	if expected.Algorithm != "" && current.Algorithm != "" && expected.Algorithm != current.Algorithm {
		return false
	}
	return expected.Fingerprint == current.Fingerprint
}
