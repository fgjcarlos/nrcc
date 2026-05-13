package handler

import (
	"net/http"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// FlowHandler handles flow endpoints
type FlowHandler struct {
	svc *service.FlowService
}

// NewFlowHandler creates a new flow handler
func NewFlowHandler(svc *service.FlowService) *FlowHandler {
	return &FlowHandler{svc: svc}
}

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
	result, _ := h.svc.Analyze()
	model.RespondJSON(w, http.StatusOK, result)
}
