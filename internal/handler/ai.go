package handler

import (
	"errors"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// AIHandler handles AI/analysis endpoints.
type AIHandler struct {
	svc *service.AIService
}

// NewAIHandler creates a new AI handler.
func NewAIHandler(svc ...*service.AIService) *AIHandler {
	if len(svc) > 0 && svc[0] != nil {
		return &AIHandler{svc: svc[0]}
	}
	return &AIHandler{svc: service.NewAIService(service.LoadAIConfigFromEnv())}
}

// PostAnalyzeFlow analyzes a flow with explicit, review-first AI assistance.
// POST /api/ai/analyze/flow
func (h *AIHandler) PostAnalyzeFlow(w http.ResponseWriter, r *http.Request) {
	var req service.AIFlowRequest
	if !DecodeJSON(w, r, &req) {
		return
	}
	if req.Action == "" {
		req.Action = service.AIActionExplain
	}

	resp, err := h.svc.AssistFlow(r.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		code := "AI_REQUEST_ERROR"
		if errors.Is(err, http.ErrHandlerTimeout) {
			status = http.StatusGatewayTimeout
		}
		if errors.Is(err, service.ErrAIDisabled) {
			status = http.StatusServiceUnavailable
			code = "AI_DISABLED"
		} else if errors.Is(err, service.ErrAIKeyRequired) || errors.Is(err, service.ErrAIEndpointReqd) {
			status = http.StatusServiceUnavailable
			code = "AI_NOT_CONFIGURED"
		}
		model.RespondError(w, status, code, err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, resp)
}
