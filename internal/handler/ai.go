package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid AI flow request")
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
		if err.Error() == "AI flow assistance is disabled; set NRCC_AI_ENABLED=true to enable" {
			status = http.StatusServiceUnavailable
			code = "AI_DISABLED"
		} else if err.Error() == "AI provider API key is required for non-offline providers" || err.Error() == "AI provider endpoint is required" {
			status = http.StatusServiceUnavailable
			code = "AI_NOT_CONFIGURED"
		}
		model.RespondError(w, status, code, err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, resp)
}

// PostAnalyzePatterns analyzes patterns with AI (stub)
// POST /api/ai/analyze/patterns
func (h *AIHandler) PostAnalyzePatterns(w http.ResponseWriter, r *http.Request) {
	model.RespondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Pattern analysis is not yet implemented")
}

// GetPatternReadme gets pattern documentation (stub)
// GET /api/ai/patterns/{id}/readme
func (h *AIHandler) GetPatternReadme(w http.ResponseWriter, r *http.Request) {
	model.RespondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Pattern documentation is not yet available")
}

// DownloadPattern downloads a pattern (stub)
// GET /api/ai/patterns/{id}/download
func (h *AIHandler) DownloadPattern(w http.ResponseWriter, r *http.Request) {
	model.RespondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Pattern download is not yet available")
}
