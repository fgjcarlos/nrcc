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

	// Optional ProcessManager wire-up. When non-nil, SaveConfig triggers
	// a Node-RED restart after the settings.js write so Editor Theme,
	// Editor Library, Logging, Projects etc. changes take effect without
	// a separate manual restart click. Stays nil in unit tests where
	// there is no managed Node-RED process — #715.
	processManager *service.ProcessManager
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(configSvc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{configSvc: configSvc}
}

// SetAuditService injects the audit logger.
func (h *ConfigHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetProcessManager wires the ProcessManager so a successful Save
// triggers a Node-RED restart. Pass nil to disable auto-restart
// (used in edge mode and in tests).
func (h *ConfigHandler) SetProcessManager(pm *service.ProcessManager) { h.processManager = pm }

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

// SaveConfig handles POST /api/config - protected, admin only.
// Authorization (admin role) is enforced by middleware.RequireAdmin on the
// route. Claims are read from the context solely for audit logging.
func (h *ConfigHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)

	var cfg model.NodeRedConfig
	if !DecodeJSON(w, r, &cfg) {
		return
	}

	if err := h.configSvc.Save(cfg); err != nil {
		model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Node-RED reads settings.js only at start. Do not acknowledge the save
	// until a managed running process has successfully reloaded it; otherwise
	// the UI would claim success while the active runtime still used the old
	// configuration.
	if h.processManager != nil && !h.processManager.IsExternalMode() {
		if status := h.processManager.Status(); status.Status == "running" {
			if err := h.processManager.Restart(); err != nil {
				model.RespondError(w, http.StatusInternalServerError, "CONFIG_SAVED_RESTART_FAILED", "Configuration was saved, but Node-RED could not restart: "+err.Error())
				return
			}
		}
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
