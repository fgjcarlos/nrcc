package handler

import (
	"errors"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// AIHandler handles AI/analysis endpoints.
type AIHandler struct {
	svc       *service.AIService
	configSvc *service.AIConfigService
	audit     *audit.Service
}

// NewAIHandler creates a new AI handler.
func NewAIHandler(svc *service.AIService, configSvc *service.AIConfigService) *AIHandler {
	if svc == nil {
		svc = service.NewAIService(service.LoadAIConfigFromEnv())
	}
	return &AIHandler{svc: svc, configSvc: configSvc}
}

// SetAuditService injects the audit logger.
func (h *AIHandler) SetAuditService(a *audit.Service) { h.audit = a }

// GetConfig handles GET /api/ai/config.
func (h *AIHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if middleware.ClaimsFromContext(r) == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	view, err := h.configSvc.View()
	if err != nil {
		h.respondConfigError(w, err)
		return
	}
	model.RespondJSON(w, http.StatusOK, view)
}

// GetStatus handles GET /api/ai/status without probing a provider.
func (h *AIHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if middleware.ClaimsFromContext(r) == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	status, err := h.configSvc.Status()
	if err != nil {
		h.respondConfigError(w, err)
		return
	}
	model.RespondJSON(w, http.StatusOK, status)
}

// PutConfig handles PUT /api/ai/config.
func (h *AIHandler) PutConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	var cfg service.AIConfig
	if !DecodeJSON(w, r, &cfg) {
		return
	}
	if err := h.configSvc.Validate(cfg); err != nil {
		h.respondConfigError(w, err)
		return
	}
	if err := h.configSvc.Save(cfg); err != nil {
		h.respondConfigError(w, err)
		return
	}
	view, err := h.configSvc.View()
	if err != nil {
		h.respondConfigError(w, err)
		return
	}
	h.audit.Log(r, claims.Username, "AI_CONFIG_SAVE", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, view)
}

// TestConfig handles POST /api/ai/config/test.
func (h *AIHandler) TestConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	status, err := h.configSvc.Test(r.Context())
	if err != nil {
		h.respondConfigError(w, err)
		return
	}
	h.audit.Log(r, claims.Username, "AI_CONFIG_TEST", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, status)
}

func (h *AIHandler) respondConfigError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "AI_CONFIG_ERROR"
	switch {
	case errors.Is(err, service.ErrAIInvalidProvider):
		status, code = http.StatusBadRequest, "AI_INVALID_PROVIDER"
	case errors.Is(err, service.ErrAIInvalidEndpoint):
		status, code = http.StatusBadRequest, "AI_INVALID_ENDPOINT"
	case errors.Is(err, service.ErrAIEncryptionKeyRequired):
		status, code = http.StatusServiceUnavailable, "AI_ENCRYPTION_KEY_REQUIRED"
	case errors.Is(err, service.ErrAIConfigIncomplete), errors.Is(err, service.ErrAIKeyRequired):
		status, code = http.StatusServiceUnavailable, "AI_INCOMPLETE"
	case errors.Is(err, service.ErrAIDisabled):
		status, code = http.StatusServiceUnavailable, "AI_DISABLED"
	case errors.Is(err, service.ErrAIProbeTimeout):
		status, code = http.StatusGatewayTimeout, "AI_PROBE_TIMEOUT"
	case errors.Is(err, service.ErrAIProbeUnreachable):
		status, code = http.StatusBadGateway, "AI_PROBE_UNREACHABLE"
	}
	model.RespondError(w, status, code, err.Error())
}

// PostAnalyzeFlow analyzes a flow with explicit, review-first AI assistance.
// POST /api/ai/analyze/flow
func (h *AIHandler) PostAnalyzeFlow(w http.ResponseWriter, r *http.Request) {
	if h.configSvc != nil {
		providerStatus, err := h.configSvc.Status()
		if err != nil {
			h.respondConfigError(w, err)
			return
		}
		if providerStatus.Status != "ready" {
			model.RespondError(w, http.StatusServiceUnavailable, "AI_NOT_READY", providerStatus.Reason)
			return
		}
	}

	var req service.AIFlowRequest
	if !DecodeJSON(w, r, &req) {
		return
	}
	if req.Action == "" {
		req.Action = service.AIActionExplain
	}

	aiSvc := h.svc
	if h.configSvc != nil {
		cfg, err := h.configSvc.Load()
		if err != nil {
			h.respondConfigError(w, err)
			return
		}
		aiSvc = service.NewAIService(cfg)
	}
	resp, err := aiSvc.AssistFlow(r.Context(), req)
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
