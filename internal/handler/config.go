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

	// Issue #877: the credential-encryption secret and the TLS
	// private-key passphrase are write-only for every role, admin
	// included. The response carries the non-secret TLS material so the
	// configuration UI can still render the TLS section, and the stored
	// secrets are untouched — the client reads them as "blank, keep the
	// current value" and the service preserves them on write-back.
	model.RespondJSON(w, http.StatusOK, cfg.RedactedSecrets())
}

// SaveConfig handles POST /api/config - protected, admin only.
// Authorization (admin role) is enforced by middleware.RequireAdmin on the
// route. Claims are read from the context solely for audit logging.
//
// Slice C: when an ApplyCoordinator is wired, the structured save runs
// through it so the settings.js write is single-flighted against
// /api/settings/raw and goes through the validate → backup → atomic write
// → audit transaction. The revision precondition runs inside the
// coordinator, under the single-flight guard and before the structured
// commit, so a rejected save leaves both representations unchanged. A
// revision mismatch surfaces as 409 SETTINGS_REVISION_CONFLICT matching
// the slice C handler envelope.
//
// Issue #878: the body may carry the optional `expectedRevision`
// precondition — the settings.js fingerprint the operator read before
// composing the edit. When it is present the save is rejected with 409
// SETTINGS_REVISION_CONFLICT unless the live revision still matches, so a
// stale browser session cannot overwrite newer state. When it is absent
// the endpoint keeps its historical "no expectation" behaviour for
// clients that never learned the revision (see
// CheckRevisionPrecondition's empty-expectation bypass).
func (h *ConfigHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	capabilities := h.configSvc.ConfigurationCapabilities()
	if !capabilities.Editable {
		model.RespondError(w, http.StatusConflict, "CONFIGURATION_READ_ONLY", capabilities.Reason)
		return
	}

	// The typed NodeRedConfig decode swallows unknown fields, so the
	// shared two-pass decoder is used to keep `expectedRevision` out of
	// the configuration payload while preserving it as the precondition.
	var req StructuredConfigRequest
	if !DecodeStructuredConfigPayload(w, r, &req) {
		return
	}
	cfg := req.Configuration

	if h.applyCoordinator != nil {
		h.saveConfigViaApply(w, r, claims, cfg, capabilities, req.ExpectedRevision)
		return
	}
	h.saveConfigLegacy(w, r, claims, cfg, capabilities, req.ExpectedRevision)
}

// saveConfigViaApply is the slice C delegation path. It commits
// the JSON store, then drives ApplyCoordinator.Apply against the
// rendered settings.js content.
//
// Issue #878: the apply's revision precondition uses the revision the
// operator actually read (expectedRevision from the request body). Only
// when the client omits it does the handler fall back to the live
// revision captured at read time — the historical behaviour that kept
// older dashboards working.
func (h *ConfigHandler) saveConfigViaApply(w http.ResponseWriter, r *http.Request, claims *model.Claims, cfg model.NodeRedConfig, capabilities model.ConfigurationCapabilities, expectedRevision string) {
	live, err := h.configSvc.GetRawSettings()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SETTINGS_ERROR", err.Error())
		return
	}

	expected := live.Revision
	if expectedRevision != "" {
		expected = model.SourceRevision{
			Fingerprint: expectedRevision,
			Algorithm:   service.SourceRevisionAlgorithm,
		}
	}

	req := service.ApplyRequest{
		Path:             live.Path,
		Expected:         expected,
		BackupDir:        backupDirFor(h.configSvc, live.Path),
		StructuredConfig: &cfg,
		Actor:            claims.Username,
		Request:          r,
		Capabilities:     capabilities,
	}
	result, err := h.applyCoordinator.Apply(r.Context(), req)
	if err != nil {
		writeApplyError(w, err, "Failed to apply settings")
		return
	}

	// The apply coordinator already emitted apply.success; we keep
	// the legacy CONFIG_SAVE audit for parity with the pre-slice-C
	// dashboards that consume the action field.
	if h.audit != nil {
		h.audit.Log(r, claims.Username, "CONFIG_SAVE", live.Path, "ok", compatibilityAuditMeta(capabilities))
	}
	// Issue #877: the committed configuration echoes the write-only
	// secrets, so they are redacted before the response leaves the API.
	model.RespondJSON(w, http.StatusOK, result.Configuration.RedactedSecrets())
}

// saveConfigLegacy is the pre-slice-C path: it commits the JSON
// store + writes settings.js directly via ConfigService.Save and
// triggers the ProcessManager restart. Kept so existing unit
// tests that do not wire an ApplyCoordinator keep passing.
//
// Issue #878: the legacy path has no apply coordinator to run the
// revision precondition, so the handler enforces the client-supplied
// expectation itself before the first write.
func (h *ConfigHandler) saveConfigLegacy(w http.ResponseWriter, r *http.Request, claims *model.Claims, cfg model.NodeRedConfig, capabilities model.ConfigurationCapabilities, expectedRevision string) {
	if !h.checkExpectedRevision(w, expectedRevision) {
		return
	}
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
	// Issue #877: the echo carries the write-only secrets, so they are
	// redacted before the response leaves the API.
	model.RespondJSON(w, http.StatusOK, cfg.RedactedSecrets())
}

// checkExpectedRevision enforces the client-supplied revision
// precondition (issue #878) on the paths that do not run an apply
// coordinator. It returns true when the request may proceed.
//
// An empty expectation keeps the historical "no expectation" semantics:
// the caller never read a revision, so there is nothing to compare
// against. A non-empty expectation that does not match the live
// settings.js revision writes the same 409 SETTINGS_REVISION_CONFLICT
// envelope the apply surfaces use, so the operator UI renders one
// conflict dialog for every save surface.
func (h *ConfigHandler) checkExpectedRevision(w http.ResponseWriter, expectedRevision string) bool {
	if expectedRevision == "" {
		return true
	}
	live, err := h.configSvc.GetRawSettings()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SETTINGS_ERROR", err.Error())
		return false
	}
	provided := model.SourceRevision{
		Fingerprint: expectedRevision,
		Algorithm:   service.SourceRevisionAlgorithm,
	}
	if service.RevisionMatches(provided, live.Revision) {
		return true
	}
	writeRevisionConflict(w, &service.RevisionConflictError{
		Live:     live.Revision,
		Provided: provided,
		Path:     live.Path,
	})
	return false
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
		case service.ApplyStageReconcile:
			// A source that declares a configuration the structured mirror
			// cannot hold is a validation failure (issue #880); a failing
			// mirror commit is an infrastructure failure.
			if errors.Is(ae, service.ErrMirrorRejected) {
				model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", ae.Error())
				return
			}
			model.RespondError(w, http.StatusInternalServerError, "SETTINGS_RECONCILE_ERROR", ae.Error())
		case service.ApplyStageRestart:
			var rollbackErr *service.RollbackError
			if errors.As(ae, &rollbackErr) {
				model.RespondError(w, http.StatusInternalServerError, "RUNTIME_RECOVERY_FAILED", "Configuration apply failed and runtime recovery was unsuccessful")
				return
			}
			model.RespondError(w, http.StatusInternalServerError, "RUNTIME_RESTART_FAILED", "Configuration apply failed while restarting Node-RED")
		case service.ApplyStageReady:
			var rollbackErr *service.RollbackError
			if errors.As(ae, &rollbackErr) {
				model.RespondError(w, http.StatusInternalServerError, "RUNTIME_RECOVERY_FAILED", "Configuration apply failed and runtime recovery was unsuccessful")
				return
			}
			model.RespondError(w, http.StatusInternalServerError, "RUNTIME_READINESS_FAILED", "Configuration apply failed because Node-RED did not become ready")
		case service.ApplyStageRollback:
			model.RespondError(w, http.StatusInternalServerError, "RUNTIME_RECOVERY_FAILED", "Configuration apply failed and authoritative state recovery was unsuccessful")
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
	// Issue #877: the defaults are write-only secrets too — keep the
	// contract uniform so no response shape ever carries them.
	model.RespondJSON(w, http.StatusOK, cfg.RedactedSecrets())
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
