package service

import (
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// InstanceStore holds the configured Node-RED instances.
//
// This is the first (read-only, backwards-compatible) slice of the
// multi-instance model (docs/architecture/multi-instance-node-red.md, #144).
// It synthesizes a single "default" instance from the current DATA_DIR and
// persists nothing — existing single-instance installs are byte-for-byte
// unaffected. Persistence and mutation arrive in a later slice alongside
// POST /api/instances.
type InstanceStore struct {
	defaultInstance model.Instance
}

// NewInstanceStore seeds the default instance pointing at the existing data
// directory. The default is the implicit single-instance target every current
// deployment already runs.
func NewInstanceStore(dataDir string) *InstanceStore {
	now := time.Now().UTC()
	return &InstanceStore{
		defaultInstance: model.Instance{
			ID:        model.DefaultInstanceID,
			Name:      "Default",
			Kind:      model.InstanceKindLocal,
			DataDir:   dataDir,
			Health:    model.InstanceHealthUnknown,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// List returns all configured instances. For this slice that is only the
// synthesized default.
func (s *InstanceStore) List() []model.Instance {
	return []model.Instance{s.defaultInstance}
}
