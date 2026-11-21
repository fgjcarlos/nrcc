package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// applyPendingSuffix names the marker that records an apply in flight
// (issue #1012). It sits next to settings.js and holds the path of the
// pre-apply snapshot. It deliberately does not share the ".apply-" prefix
// of the atomic writer's tmp siblings.
const applyPendingSuffix = ".pending-apply"

func applyPendingMarkerPath(settingsPath string) string {
	return settingsPath + applyPendingSuffix
}

// writeApplyPendingMarker durably records snapshotPath before the candidate
// settings.js is written, so a process killed before readiness can be
// rolled back at the next boot.
func writeApplyPendingMarker(ctx context.Context, settingsPath, snapshotPath string) error {
	return AtomicWriteSettings(ctx, applyPendingMarkerPath(settingsPath), snapshotPath)
}

// removeApplyPendingMarker clears the marker once the apply outcome is
// settled. A missing marker is not an error.
func removeApplyPendingMarker(settingsPath string) error {
	if err := os.Remove(applyPendingMarkerPath(settingsPath)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// RecoverInterruptedApply rolls back an apply that never confirmed
// readiness (issue #1012). When the pending-apply marker next to
// settingsPath exists, the snapshot it names is restored over settings.js,
// the marker is removed, and the structured mirror is re-derived from the
// restored source. Call it at boot, before Node-RED starts.
//
// It returns the restored snapshot path, or "" when no apply was pending.
// A non-nil error with a non-empty path means the source was restored but
// a follow-up step (mirror or marker cleanup) failed.
func (s *ConfigService) RecoverInterruptedApply(ctx context.Context, settingsPath string) (string, error) {
	if settingsPath == "" {
		return "", nil
	}
	marker := applyPendingMarkerPath(settingsPath)
	// #nosec G304 -- marker is settingsPath (operator-supplied dataDir) plus a constant suffix.
	data, err := os.ReadFile(marker)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read apply marker %s: %w", marker, err)
	}
	snapshot := strings.TrimSpace(string(data))
	if snapshot == "" {
		return "", fmt.Errorf("apply marker %s names no snapshot", marker)
	}
	if err := restoreSnapshotContext(ctx, snapshot, settingsPath); err != nil {
		// The marker stays so the next boot retries the restore.
		return "", fmt.Errorf("restore snapshot %s: %w", snapshot, err)
	}
	// The source is restored: drop the marker first so a later save is
	// never reverted by a stale marker, even if the mirror step fails.
	if err := removeApplyPendingMarker(settingsPath); err != nil {
		return snapshot, fmt.Errorf("remove apply marker %s: %w", marker, err)
	}
	// The source is authoritative (issue #880): the structured mirror goes
	// back with it.
	content, err := os.ReadFile(settingsPath) // #nosec G304 -- operator-supplied settings path.
	if err != nil {
		return snapshot, fmt.Errorf("read restored settings: %w", err)
	}
	if _, err := s.ReconcileRawSettings(string(content)); err != nil {
		return snapshot, fmt.Errorf("reconcile structured mirror: %w", err)
	}
	return snapshot, nil
}
