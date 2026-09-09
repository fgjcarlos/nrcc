package handler

import (
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// DashboardHandler exposes read-only dashboard discovery.
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler creates a dashboard discovery handler.
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// GetDiscovery returns installed dashboard capabilities and flow-owned paths.
// GET /api/dashboards/discovery
func (h *DashboardHandler) GetDiscovery(w http.ResponseWriter, r *http.Request) {
	model.RespondJSON(w, http.StatusOK, h.svc.Discover())
}
