package handler

import (
	"net/http"

	"github.com/composedof2/nrcc/internal/model"
)

// AIHandler handles AI/analysis endpoints (stubs)
type AIHandler struct{}

// NewAIHandler creates a new AI handler
func NewAIHandler() *AIHandler {
	return &AIHandler{}
}

// PostAnalyzeFlow analyzes a flow with AI (stub)
// POST /api/ai/analyze/flow
func (h *AIHandler) PostAnalyzeFlow(w http.ResponseWriter, r *http.Request) {
	model.RespondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Flow analysis is not yet implemented")
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
