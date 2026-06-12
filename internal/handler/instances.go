package handler

import (
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// InstanceHandler serves the read-only multi-instance API. The first slice
// exposes only GET /api/instances; lifecycle and mutation routes come later
// (docs/architecture/multi-instance-node-red.md).
type InstanceHandler struct {
	store *service.InstanceStore
}

// NewInstanceHandler creates an instance handler backed by the given store.
func NewInstanceHandler(store *service.InstanceStore) *InstanceHandler {
	return &InstanceHandler{store: store}
}

// GetInstances handles GET /api/instances. Read-only and backwards-compatible:
// it returns the configured instances (currently just the default) and does not
// change the single-instance runtime.
func (h *InstanceHandler) GetInstances(w http.ResponseWriter, r *http.Request) {
	model.RespondJSON(w, http.StatusOK, h.store.List())
}
