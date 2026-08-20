package handler

import (
	"errors"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// ConfigHandler handles configuration endpoints
type ConfigHandler struct {
	configSvc *service.ConfigService
	audit     *audit.Service
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(configSvc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{configSvc: configSvc}
}

// SetAuditService injects the audit logger.
func (h *ConfigHandler) SetAuditService(a *audit.Service) { h.audit = a }

// GetConfig handles GET /api/config - protected
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	cfg, err := h.configSvc.Get()
	if err != nil {
		// settings.js sandbox timeouts must surface as a 4xx so the
		// operator gets a clear "settings.js did not terminate" instead
		// of a hung request (issue #665).
		if errors.Is(err, service.ErrSandboxTimeout) {
			model.RespondError(w, http.StatusUnprocessableEntity, "SETTINGS_TIMEOUT", err.Error())
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "CONFIG_ERROR", "Failed to read config")
		return
	}

	// MEDIUM-015: redact secrets for non-admin viewers. Password hashes and
	// cleartext env values are not safe to expose to a logged-in viewer.
	// Encrypted env blobs are cipher bytes, not cleartext, so they pass
	// through unchanged.
	if claims.Role != model.RoleAdmin {
		if cfg.AdminAuth != nil {
			for i := range cfg.AdminAuth.Users {
				cfg.AdminAuth.Users[i].Password = ""
			}
		}
		for i := range cfg.EnvVars {
			if !cfg.EnvVars[i].Encrypted {
				cfg.EnvVars[i].Value = "********"
			}
		}
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
	if !DecodeJSON(w, r, &cfg) {
		return
	}

	if err := h.configSvc.Save(cfg); err != nil {
		model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	h.audit.Log(r, claims.Username, "CONFIG_SAVE", "", "ok", nil)
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
	if !DecodeJSON(w, r, &cfg) {
		return
	}

	if err := h.configSvc.Validate(cfg); err != nil {
		model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]bool{"valid": true})
}
