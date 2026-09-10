package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// DashboardHandler exposes read-only dashboard discovery.
type DashboardHandler struct {
	svc    *service.DashboardService
	access *service.DashboardAccessService
}

// NewDashboardHandler creates a dashboard discovery handler.
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// SetAccessService wires the optional transactional dashboard policy service.
func (h *DashboardHandler) SetAccessService(access *service.DashboardAccessService) {
	h.access = access
}

// GetDiscovery returns installed dashboard capabilities and flow-owned paths.
// GET /api/dashboards/discovery
func (h *DashboardHandler) GetDiscovery(w http.ResponseWriter, r *http.Request) {
	model.RespondJSON(w, http.StatusOK, h.svc.Discover())
}

// ApplyAccess applies a reviewed dashboard policy through the shared coordinator.
func (h *DashboardHandler) ApplyAccess(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil || h.access == nil {
		model.RespondError(w, http.StatusConflict, "DASHBOARD_POLICY_UNAVAILABLE", "Dashboard policy is unavailable")
		return
	}
	var policy model.DashboardAccessPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid dashboard policy")
		return
	}
	doc, err := h.access.Apply(r.Context(), policy, claims.Username, r)
	if err != nil {
		if errors.Is(err, service.ErrDashboardPolicyUnavailable) || errors.Is(err, service.ErrUnsafeDashboardPolicy) {
			model.RespondError(w, http.StatusBadRequest, "DASHBOARD_POLICY_REJECTED", "Dashboard policy was rejected")
			return
		}
		var conflict *service.RevisionConflictError
		if errors.As(err, &conflict) {
			writeRevisionConflict(w, conflict)
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "DASHBOARD_POLICY_APPLY_FAILED", "Failed to apply dashboard policy")
		return
	}
	doc.Content = service.RedactSettingsContent(doc.Content)
	model.RespondJSON(w, http.StatusOK, struct {
		Document model.SettingsDocument `json:"document"`
	}{Document: doc})
}
