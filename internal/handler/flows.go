package handler

import (
	"net/http"

	"github.com/composedof2/nrcc/internal/audit"
	"github.com/composedof2/nrcc/internal/middleware"
	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// FlowHandler handles flow endpoints
type FlowHandler struct {
	svc        *service.FlowService
	versionSvc *service.FlowVersionService
	audit      *audit.Service
}

// NewFlowHandler creates a new flow handler
func NewFlowHandler(svc *service.FlowService) *FlowHandler {
	return &FlowHandler{svc: svc}
}

// SetVersionService injects the flow version service.
func (h *FlowHandler) SetVersionService(vs *service.FlowVersionService) { h.versionSvc = vs }

// SetAuditService injects the audit logger.
func (h *FlowHandler) SetAuditService(a *audit.Service) { h.audit = a }

// GetFlows lists all flows
// GET /api/flows
func (h *FlowHandler) GetFlows(w http.ResponseWriter, r *http.Request) {
	flows, err := h.svc.GetFlows()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "FLOW_ERROR", err.Error())
		return
	}

	if flows == nil {
		flows = []interface{}{}
	}

	model.RespondJSON(w, http.StatusOK, flows)
}

// GetFlow gets a single flow by ID
// GET /api/flows/{id}
func (h *FlowHandler) GetFlow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	flow, err := h.svc.GetFlow(id)
	if err != nil {
		model.RespondError(w, http.StatusNotFound, "FLOW_NOT_FOUND", "Flow not found")
		return
	}

	model.RespondJSON(w, http.StatusOK, flow)
}

// ExportFlows exports all flows
// GET /api/flows/export
func (h *FlowHandler) ExportFlows(w http.ResponseWriter, r *http.Request) {
	flowsData, err := h.svc.ExportFlows()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "FLOW_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"flows.json\"")
	w.WriteHeader(http.StatusOK)
	w.Write(flowsData)
}

// AnalyzeFlows analyzes flows (stub)
// POST /api/flows/analyze
func (h *FlowHandler) AnalyzeFlows(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Analyze()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "FLOW_ANALYZE_ERROR", err.Error())
		return
	}
	model.RespondJSON(w, http.StatusOK, result)
}

// GetVersions lists flow versions
// GET /api/flows/versions
func (h *FlowHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
	if h.versionSvc == nil {
		model.RespondJSON(w, http.StatusOK, []interface{}{})
		return
	}

	versions, err := h.versionSvc.ListVersions()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "VERSION_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, versions)
}

// GetVersionDiff computes diff between two flow versions
// GET /api/flows/versions/{from}/diff/{to}
func (h *FlowHandler) GetVersionDiff(w http.ResponseWriter, r *http.Request) {
	if h.versionSvc == nil {
		model.RespondError(w, http.StatusServiceUnavailable, "VERSION_ERROR", "Versioning not available")
		return
	}

	fromID := chi.URLParam(r, "from")
	toID := chi.URLParam(r, "to")

	diff, err := h.versionSvc.DiffVersions(fromID, toID)
	if err != nil {
		model.RespondError(w, http.StatusNotFound, "VERSION_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, diff)
}

// PostRevert reverts flows to a specific version
// POST /api/flows/versions/{id}/revert
func (h *FlowHandler) PostRevert(w http.ResponseWriter, r *http.Request) {
	if h.versionSvc == nil {
		model.RespondError(w, http.StatusServiceUnavailable, "VERSION_ERROR", "Versioning not available")
		return
	}

	versionID := chi.URLParam(r, "id")

	if err := h.versionSvc.CaptureNow(); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "VERSION_ERROR", "Failed to snapshot current state")
		return
	}

	if err := h.versionSvc.Revert(versionID); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "VERSION_ERROR", err.Error())
		return
	}

	actor := ""
	if claims := middleware.ClaimsFromContext(r); claims != nil {
		actor = claims.Username
	}
	h.audit.Log(r, actor, "FLOW_REVERT", versionID, "ok", nil)

	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Flows reverted",
		"versionId": versionID,
	})
}

// PostSnapshot captures a manual flow snapshot
// POST /api/flows/versions
func (h *FlowHandler) PostSnapshot(w http.ResponseWriter, r *http.Request) {
	if h.versionSvc == nil {
		model.RespondError(w, http.StatusServiceUnavailable, "VERSION_ERROR", "Versioning not available")
		return
	}

	if err := h.versionSvc.CaptureNow(); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "VERSION_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusCreated, map[string]string{"message": "Snapshot captured"})
}
