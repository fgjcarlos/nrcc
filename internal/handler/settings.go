package handler

import (
	"errors"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// SettingsHandler exposes the raw settings.js editor.
type SettingsHandler struct {
	configSvc      *service.ConfigService
	processManager *service.ProcessManager
	audit          *audit.Service
}

// RawSettingsRequest is the payload for raw settings updates.
type RawSettingsRequest struct {
	Content string `json:"content"`
}

// NewSettingsHandler creates a settings handler.
func NewSettingsHandler(configSvc *service.ConfigService) *SettingsHandler {
	return &SettingsHandler{configSvc: configSvc}
}

// SetAuditService injects the audit logger.
func (h *SettingsHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetProcessManager wires the managed Node-RED lifecycle. Raw settings edits
// affect the runtime only after a restart, exactly like structured config.
func (h *SettingsHandler) SetProcessManager(pm *service.ProcessManager) { h.processManager = pm }

// GetRaw handles GET /api/settings/raw.
// Authorization (admin role) is enforced by middleware.RequireAdmin on the
// route; this handler trusts the request context to contain claims and never
// makes the admin/non-admin decision itself.
func (h *SettingsHandler) GetRaw(w http.ResponseWriter, r *http.Request) {
	doc, err := h.configSvc.GetRawSettings()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SETTINGS_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, doc)
}

// SaveRaw handles POST /api/settings/raw.
// Authorization (admin role) is enforced by middleware.RequireAdmin on the
// route. Claims are read from the context solely for audit logging.
func (h *SettingsHandler) SaveRaw(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)

	var req RawSettingsRequest
	if !DecodeJSON(w, r, &req) {
		return
	}
	if req.Content == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Settings content cannot be empty")
		return
	}

	doc, err := h.configSvc.SaveRawSettings(req.Content)
	if err != nil {
		// settings.js sandbox timeouts must surface as a 4xx so the
		// operator gets a clear "settings.js did not terminate" instead
		// of a hung request, and so the bad content is rejected before
		// being persisted to disk (issue #665).
		if errors.Is(err, service.ErrSandboxTimeout) {
			model.RespondError(w, http.StatusUnprocessableEntity, "SETTINGS_TIMEOUT", err.Error())
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "SETTINGS_WRITE_ERROR", err.Error())
		return
	}
	if h.processManager != nil && !h.processManager.IsExternalMode() {
		if status := h.processManager.Status(); status.Status == "running" {
			if err := h.processManager.Restart(); err != nil {
				model.RespondError(w, http.StatusInternalServerError, "SETTINGS_SAVED_RESTART_FAILED", "settings.js was saved, but Node-RED could not restart: "+err.Error())
				return
			}
		}
	}
	h.audit.Log(r, claims.Username, "SETTINGS_UPDATE", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, doc)
}
