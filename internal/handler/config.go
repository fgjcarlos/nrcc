package handler

import (
	"encoding/json"
	"net/http"

	"github.com/composedof2/nrcc/internal/middleware"
	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// ConfigHandler handles configuration endpoints
type ConfigHandler struct {
	configSvc *service.ConfigService
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(configSvc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{configSvc: configSvc}
}

// GetConfig handles GET /api/config - protected
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	cfg, err := h.configSvc.Get()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "CONFIG_ERROR", "Failed to read config")
		return
	}

	model.RespondJSON(w, http.StatusOK, cfg)
}

// SaveConfig handles POST /api/config - protected, admin only
func (h *ConfigHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil || claims.Role != model.RoleAdmin {
		model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	var cfg model.NodeRedConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.configSvc.Save(cfg); err != nil {
		model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, cfg)
}

// GetDefaultConfig handles GET /api/config/default - protected
func (h *ConfigHandler) GetDefaultConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	cfg := h.configSvc.GetDefault()
	model.RespondJSON(w, http.StatusOK, cfg)
}

// ValidateConfig handles POST /api/config/validate - protected
func (h *ConfigHandler) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var cfg model.NodeRedConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.configSvc.Validate(cfg); err != nil {
		model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]bool{"valid": true})
}
