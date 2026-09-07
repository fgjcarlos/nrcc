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
//
// Slice C of #758 — SaveRaw delegates to ApplyCoordinator when
// one is wired (see SetApplyCoordinator). When the coordinator
// is nil the handler falls back to the legacy
// ConfigService.SaveRawSettingsWithRevision path so existing
// unit tests that build a SettingsHandler with only a
// ConfigService keep passing.
type SettingsHandler struct {
	configSvc      *service.ConfigService
	processManager *service.ProcessManager
	audit          *audit.Service

	// applyCoordinator is the optional slice C wiring. When
	// non-nil, SaveRaw routes the write through the
	// single-flight apply coordinator so the raw surface
	// shares concurrency control with the new
	// POST /api/config/apply/raw endpoint.
	applyCoordinator *service.ApplyCoordinator
}

// RawSettingsRequest is the payload for raw settings updates.
type RawSettingsRequest struct {
	Content          string `json:"content"`
	ExpectedRevision string `json:"expectedRevision,omitempty"`
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

// SetApplyCoordinator wires the slice C apply coordinator so the
// raw Save goes through validate → backup → atomic write → audit.
// Pass nil to disable (the default in tests that do not exercise
// the apply pipeline). Slice C of #758.
func (h *SettingsHandler) SetApplyCoordinator(c *service.ApplyCoordinator) { h.applyCoordinator = c }

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
//
// Slice C of #758 — when an ApplyCoordinator is wired, the raw
// save routes through it so the write participates in the
// validate → backup → atomic write → audit transaction and
// shares the single-flight concurrency guard with the new
// /api/config/apply/raw endpoint. When the coordinator is nil
// the legacy ConfigService.SaveRawSettingsWithRevision path is
// used so existing tests keep passing.
func (h *SettingsHandler) SaveRaw(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	capabilities := h.configSvc.ConfigurationCapabilities()
	if !capabilities.Editable {
		model.RespondError(w, http.StatusConflict, "CONFIGURATION_READ_ONLY", capabilities.Reason)
		return
	}

	var req RawSettingsRequest
	if !DecodeJSON(w, r, &req) {
		return
	}
	if req.Content == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Settings content cannot be empty")
		return
	}

	if h.applyCoordinator != nil {
		h.saveRawViaApply(w, r, claims, req, capabilities)
		return
	}
	h.saveRawLegacy(w, r, claims, req, capabilities)
}

// saveRawViaApply routes the raw save through
// ApplyCoordinator.Apply. The handler reads the live document
// to capture the path + revision, then drives the apply
// transaction. A non-empty req.ExpectedRevision upgrades the
// precondition to an IfMatch check; an empty field falls
// through the empty-bypass documented on CheckRevisionPrecondition
// so legacy callers that have not yet captured the revision
// keep working.
func (h *SettingsHandler) saveRawViaApply(w http.ResponseWriter, r *http.Request, claims *model.Claims, req RawSettingsRequest, capabilities model.ConfigurationCapabilities) {
	live, err := h.configSvc.GetRawSettings()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SETTINGS_ERROR", err.Error())
		return
	}

	expected := live.Revision
	if req.ExpectedRevision != "" {
		expected = model.SourceRevision{
			Fingerprint: req.ExpectedRevision,
			Algorithm:   service.SourceRevisionAlgorithm,
		}
	}

	applyReq := service.ApplyRequest{
		Path:         live.Path,
		Content:      req.Content,
		Expected:     expected,
		BackupDir:    backupDirFor(h.configSvc, live.Path),
		Actor:        claims.Username,
		Request:      r,
		Capabilities: capabilities,
	}
	if _, err := h.applyCoordinator.Apply(r.Context(), applyReq); err != nil {
		writeApplyError(w, err, "Failed to apply raw settings")
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
	if h.audit != nil {
		h.audit.Log(r, claims.Username, "SETTINGS_UPDATE", live.Path, "ok", compatibilityAuditMeta(capabilities))
	}
	doc, err := h.configSvc.GetRawSettings()
	if err != nil {
		// Path is authoritative; the next GET re-stamps the
		// revision. Don't fail the response — the apply
		// already succeeded.
		doc = model.SettingsDocument{Path: live.Path}
	}
	model.RespondJSON(w, http.StatusOK, doc)
}

// saveRawLegacy is the pre-slice-C path. It calls
// ConfigService.SaveRawSettingsWithRevision directly and
// triggers the ProcessManager restart on success. Kept so
// existing tests that do not wire an ApplyCoordinator keep
// passing.
func (h *SettingsHandler) saveRawLegacy(w http.ResponseWriter, r *http.Request, claims *model.Claims, req RawSettingsRequest, capabilities model.ConfigurationCapabilities) {
	expected := model.SourceRevision{Algorithm: service.SourceRevisionAlgorithm}
	if req.ExpectedRevision != "" {
		expected.Fingerprint = req.ExpectedRevision
	}

	doc, err := h.configSvc.SaveRawSettingsWithRevision(req.Content, expected)
	if err != nil {
		// Source revision mismatch — the operator read a different copy of
		// settings.js than the one currently on disk. Refuse the write so
		// the external change is not overwritten silently (slice B of #757).
		if errors.Is(err, service.ErrSourceRevisionMismatch) {
			model.RespondError(w, http.StatusConflict, "SOURCE_REVISION_MISMATCH", err.Error())
			return
		}
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
	h.audit.Log(r, claims.Username, "SETTINGS_UPDATE", "", "ok", compatibilityAuditMeta(capabilities))
	model.RespondJSON(w, http.StatusOK, doc)
}