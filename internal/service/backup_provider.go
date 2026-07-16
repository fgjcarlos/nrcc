package service

import (
	"context"
	"fmt"
	"time"
)

// BackupProvider is the optional off-host storage provider that runs after
// the local ZIP is published. Implementations must be safe for concurrent
// use by the BackupService's scheduler and any handler-driven operations.
type BackupProvider interface {
	// Name returns a short stable identifier for the provider (e.g. "restic").
	Name() string

	// Snapshot uploads srcPath to the remote repository and returns a
	// provider-specific identifier for the new snapshot. Implementations
	// should create the underlying repository on first call (lazy init).
	Snapshot(ctx context.Context, srcPath string) (remoteID string, err error)

	// List returns the most recent snapshots in chronological order.
	List(ctx context.Context) ([]RemoteBackup, error)

	// Restore downloads the snapshot identified by remoteID to dstPath. The
	// destination is created if missing.
	Restore(ctx context.Context, remoteID, dstPath string) error
}

// RemoteBackup is the minimal descriptor the UI/API needs to show and select
// a remote snapshot. SizeBytes is the bytes added by this specific snapshot
// (restic `summary.data_added`), populated via a follow-up `stats` call when
// the provider surfaces it; it stays 0 when the provider cannot compute it
// cheaply. ponytail: avoid a `restic stats` round-trip per snapshot in the
// list path; upgrade when a paginated snapshot-detail endpoint is added.
type RemoteBackup struct {
	ID        string    `json:"id"`
	Time      time.Time `json:"time"`
	SizeBytes int64     `json:"size_bytes,omitempty"`
}

// NoopProvider is the default when no remote provider is configured.
type NoopProvider struct{}

func (NoopProvider) Name() string { return "local" }

func (NoopProvider) Snapshot(ctx context.Context, srcPath string) (string, error) {
	return "", fmt.Errorf("remote backup provider is not configured")
}

func (NoopProvider) List(ctx context.Context) ([]RemoteBackup, error) {
	return nil, nil
}

func (NoopProvider) Restore(ctx context.Context, remoteID, dstPath string) error {
	return fmt.Errorf("remote backup provider is not configured")
}