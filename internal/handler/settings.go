package handler

import (
	"encoding/json"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// SettingsHandler exposes the raw settings.js editor.
type SettingsHandler struct {
	configSvc *service.ConfigService
	audit     *audit.Service
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

// GetRaw handles GET /api/settings/raw.
func (h *SettingsHandler) GetRaw(w http.ResponseWriter, r *http.Request) {
	if middleware.ClaimsFromContext(r) == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	doc, err := h.configSvc.GetRawSettings()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SETTINGS_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, doc)
}

// SaveRaw handles POST /api/settings/raw.
func (h *SettingsHandler) SaveRaw(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil || claims.Role != model.RoleAdmin {
		model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	var req RawSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.Content == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Settings content cannot be empty")
		return
	}

	doc, err := h.configSvc.SaveRawSettings(req.Content)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SETTINGS_WRITE_ERROR", err.Error())
		return
	}
	h.audit.Log(r, claims.Username, "SETTINGS_UPDATE", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, doc)
}
