package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// ConfigHandler handles configuration endpoints.
//
// Slice C of #758 — the structured save now routes through
// ApplyCoordinator when one is wired (see SetApplyCoordinator).
// When the coordinator is nil the handler falls back to the legacy
// ConfigService.Save path so existing tests that build a
// ConfigHandler with only a ConfigService keep passing.
type ConfigHandler struct {
	configSvc *service.ConfigService
	audit     *audit.Service

	// Optional ProcessManager wire-up. When non-nil, SaveConfig triggers
	// a Node-RED restart after the settings.js write so Editor Theme,
	// Editor Library, Logging, Projects etc. changes take effect without
	// a separate manual restart click. Stays nil in unit tests where
	// there is no managed Node-RED process — #715.
	processManager *service.ProcessManager

	// Optional ApplyCoordinator (slice C of #758). When non-nil,
	// SaveConfig runs the full validate → backup → atomic write
	// transaction through ApplyCoordinator.Apply so the write is
	// single-flighted and the audit log carries the unified
	// failure_stage envelope. Nil preserves the legacy behaviour
	// so unit tests don't need an ApplyService fixture.
	applyCoordinator *service.ApplyCoordinator
}

// NewConfigHandler creates a new config handler.
func NewConfigHandler(configSvc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{configSvc: configSvc}
}

// SetAuditService injects the audit logger.
func (h *ConfigHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetProcessManager wires the ProcessManager so a successful Save
// triggers a Node-RED restart. Pass nil to disable auto-restart
// (used in edge mode and in tests).
func (h *ConfigHandler) SetProcessManager(pm *service.ProcessManager) { h.processManager = pm }

// SetApplyCoordinator wires the slice C apply coordinator so the
// structured Save goes through validate → backup → atomic write →
// audit. Pass nil to disable (tests that do not exercise the apply
// pipeline). Slice C of #758.
func (h *ConfigHandler) SetApplyCoordinator(c *service.ApplyCoordinator) { h.applyCoordinator = c }

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
		if cfg.HTTPNodeAuth != nil {
			cfg.HTTPNodeAuth.Pass = ""
		}
		if cfg.HTTPStaticAuth != nil {
			cfg.HTTPStaticAuth.Pass = ""
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
//
// Slice C: when an ApplyCoordinator is wired, the structured save runs
// through it so the settings.js write is single-flighted against
// /api/settings/raw and goes through the validate → backup → atomic write
// → audit transaction. The JSON store commit still happens FIRST (the
// same order as the legacy Save) so password preservation / bcrypt
// upgrade / validation keep their existing semantics. A revision
// mismatch at the apply stage surfaces as 409 SETTINGS_REVISION_CONFLICT
// matching the slice C handler envelope.
func (h *ConfigHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	capabilities := h.configSvc.ConfigurationCapabilities()
	if !capabilities.Editable {
		model.RespondError(w, http.StatusConflict, "CONFIGURATION_READ_ONLY", capabilities.Reason)
		return
	}

	var cfg model.NodeRedConfig
	if !DecodeJSON(w, r, &cfg) {
		return
	}

	if h.applyCoordinator != nil {
		h.saveConfigViaApply(w, r, claims, cfg, capabilities)
		return
	}
	h.saveConfigLegacy(w, r, claims, cfg, capabilities)
}

// saveConfigViaApply is the slice C delegation path. It commits
// the JSON store, then drives ApplyCoordinator.Apply against the
// rendered settings.js content. The apply's revision check uses
// the live revision captured during CommitStructured (no wire
// expectedRevision is supplied by /api/config for backward
// compatibility — the frontend learns the revision via GET).
func (h *ConfigHandler) saveConfigViaApply(w http.ResponseWriter, r *http.Request, claims *model.Claims, cfg model.NodeRedConfig, capabilities model.ConfigurationCapabilities) {
	staged, err := h.configSvc.CommitStructured(cfg)
	if err != nil {
		// Mirror the legacy SaveConfig error envelope: validation
		// failures are 400 VALIDATION_ERROR so the operator sees the
		// same UX they had before slice C. The apply coordinator is
		// never invoked for invalid configs.
		model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	req := service.ApplyRequest{
		Path:         staged.Live.Path,
		Content:      staged.Content,
		Expected:     staged.Live.Revision,
		BackupDir:    staged.BackupDir,
		Actor:        claims.Username,
		Request:      r,
		Capabilities: capabilities,
	}
	result, err := h.applyCoordinator.Apply(r.Context(), req)
	if err != nil {
		writeApplyError(w, err, "Failed to apply settings")
		return
	}
	_ = result

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

	// The apply coordinator already emitted apply.success; we keep
	// the legacy CONFIG_SAVE audit for parity with the pre-slice-C
	// dashboards that consume the action field.
	if h.audit != nil {
		h.audit.Log(r, claims.Username, "CONFIG_SAVE", staged.Live.Path, "ok", compatibilityAuditMeta(capabilities))
	}
	model.RespondJSON(w, http.StatusOK, staged.Committed)
}

// saveConfigLegacy is the pre-slice-C path: it commits the JSON
// store + writes settings.js directly via ConfigService.Save and
// triggers the ProcessManager restart. Kept so existing unit
// tests that do not wire an ApplyCoordinator keep passing.
func (h *ConfigHandler) saveConfigLegacy(w http.ResponseWriter, r *http.Request, claims *model.Claims, cfg model.NodeRedConfig, capabilities model.ConfigurationCapabilities) {
	if err := h.configSvc.Save(cfg); err != nil {
		model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if h.processManager != nil && !h.processManager.IsExternalMode() {
		if status := h.processManager.Status(); status.Status == "running" {
			if err := h.processManager.Restart(); err != nil {
				model.RespondError(w, http.StatusInternalServerError, "CONFIG_SAVED_RESTART_FAILED", "Configuration was saved, but Node-RED could not restart: "+err.Error())
				return
			}
		}
	}

	h.audit.Log(r, claims.Username, "CONFIG_SAVE", "", "ok", compatibilityAuditMeta(capabilities))
	model.RespondJSON(w, http.StatusOK, cfg)
}

// writeApplyError maps the slice C apply errors onto HTTP
// envelopes. The function is shared by the structured and raw
// handlers in config_apply.go so the revision-conflict envelope
// is identical across both endpoints.
func writeApplyError(w http.ResponseWriter, err error, fallback string) {
	var conflict *service.RevisionConflictError
	if errors.As(err, &conflict) {
		writeRevisionConflict(w, conflict)
		return
	}
	var ae *service.ApplyError
	if errors.As(err, &ae) {
		switch ae.Stage {
		case service.ApplyStageValidate:
			model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", ae.Error())
		case service.ApplyStageBackup:
			model.RespondError(w, http.StatusInternalServerError, "SETTINGS_BACKUP_FAILED", ae.Error())
		case service.ApplyStageWrite:
			model.RespondError(w, http.StatusInternalServerError, "SETTINGS_WRITE_ERROR", ae.Error())
		default:
			model.RespondError(w, http.StatusInternalServerError, "APPLY_ERROR", ae.Error())
		}
		return
	}
	if errors.Is(err, service.ErrApplyInFlight) {
		model.RespondError(w, http.StatusConflict, "APPLY_IN_FLIGHT", err.Error())
		return
	}
	if errors.Is(err, service.ErrSandboxTimeout) {
		model.RespondError(w, http.StatusUnprocessableEntity, "SETTINGS_TIMEOUT", err.Error())
		return
	}
	if errors.Is(err, service.ErrSourceRevisionMismatch) {
		// Defensive fallback for callers that still see the
		// sentinel (e.g. a future path that bypasses the
		// typed error envelope). The new typed
		// *RevisionConflictError satisfies errors.Is above
		// so this branch is rarely hit.
		model.RespondError(w, http.StatusConflict, "SETTINGS_REVISION_CONFLICT", err.Error())
		return
	}
	model.RespondError(w, http.StatusInternalServerError, "APPLY_ERROR", fallback+": "+err.Error())
}

// _ ensures context import stays in the package surface even
// when saveConfigViaApply stops touching r.Context() directly.
var _ = context.TODO

func compatibilityAuditMeta(capabilities model.ConfigurationCapabilities) map[string]string {
	return map[string]string{
		"runtime_version": capabilities.RuntimeVersion,
		"adapter":         capabilities.Adapter,
		"settings_source": capabilities.Source,
	}
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
