package handler

import (
	"net/http"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// BootstrapHandler exposes host/bootstrap status to the UI.
type BootstrapHandler struct {
	hostSvc *service.HostService
}

// NewBootstrapHandler creates a bootstrap handler.
func NewBootstrapHandler(hostSvc *service.HostService) *BootstrapHandler {
	return &BootstrapHandler{hostSvc: hostSvc}
}

// GetStatus handles GET /api/bootstrap/status.
func (h *BootstrapHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	model.RespondJSON(w, http.StatusOK, h.hostSvc.Detect())
}
