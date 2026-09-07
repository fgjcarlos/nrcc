// Package handler — slice C of issue #758.
//
// Slice A wired the apply pipeline (validate → backup → atomic
// write → audit). Slice B added the adapter restart + readiness
// + rollback primitives. Slice C exposes the HTTP surface that
// drives ApplyCoordinator.Apply from a single endpoint, with
// an explicit revision precondition that returns a structured
// 409 envelope on mismatch.
//
// Two endpoints
// =============
//
//   - POST /api/config/apply       — structured (typed settings.js
//                                    payload) save with an optional
//                                    expectedRevision header or
//                                    body field. Delegates to
//                                    ApplyCoordinator.Apply after
//                                    committing the JSON store via
//                                    ConfigService.CommitStructured.
//
//   - POST /api/config/apply/raw   — raw settings.js payload
//                                    with an explicit
//                                    expectedRevision field.
//                                    Skips the JSON store commit
//                                    because the raw editor lives
//                                    outside NRCC's typed config
//                                    model.
//
// Both endpoints require admin privileges (enforced by
// middleware.RequireAdmin on the routes) and return the same
// 409 SETTINGS_REVISION_CONFLICT envelope on revision mismatch
// so the operator UI can render a single conflict dialog for
// both surfaces.
//
// Audit discipline
// ----------------
//
// The apply coordinator owns apply.{start,success,failure}
// events. The handler emits a per-endpoint
// CONFIG_APPLY / SETTINGS_APPLY event with the endpoint name
// in the meta so the audit reader can split the two surfaces
// without parsing the action verb. The failure_stage key is
// set by the coordinator and the handler does NOT duplicate it
// in its own event.

package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// ConfigApplyHandler owns the slice C apply endpoints. The
// handler is intentionally thin: it validates the request,
// parses the revision precondition, and delegates to
// ApplyCoordinator.Apply. All transaction discipline
// (single-flight, validate, backup, atomic write, audit)
// lives in the service package; the handler only adapts
// HTTP-shaped inputs to ApplyRequest and HTTP-shaped
// outputs from ApplyResult.
type ConfigApplyHandler struct {
	configSvc       *service.ConfigService
	applyCoordinator *service.ApplyCoordinator
	audit           *audit.Service

	// processManager is optional and follows the same wire-up
	// rule as ConfigHandler: a successful apply triggers a
	// Node-RED restart when a managed process is configured.
	// nil is the test default.
	processManager *service.ProcessManager
}

// NewConfigApplyHandler creates a new handler wired to the
// configSvc + applyCoordinator pair. The audit service and
// process manager are optional and may be injected later via
// the Set* setters.
func NewConfigApplyHandler(configSvc *service.ConfigService, applyCoordinator *service.ApplyCoordinator) *ConfigApplyHandler {
	return &ConfigApplyHandler{
		configSvc:       configSvc,
		applyCoordinator: applyCoordinator,
	}
}

// SetAuditService injects the audit logger.
func (h *ConfigApplyHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetProcessManager wires the ProcessManager so a successful
// apply triggers a Node-RED restart. Pass nil to disable
// (the default in tests and in edge mode).
func (h *ConfigApplyHandler) SetProcessManager(pm *service.ProcessManager) { h.processManager = pm }

// ApplyStructuredRequest is the wire payload for
// POST /api/config/apply. The structured payload mirrors
// the existing POST /api/config body; the new field is
// ExpectedRevision, an optional sha256 fingerprint of the
// settings.js bytes the operator read before composing this
// edit. When non-empty the apply coordinator enforces the
// IfMatch precondition and returns 409 SETTINGS_REVISION_CONFLICT
// on mismatch.
//
// The handler decodes the payload in two passes (see
// ApplyStructured) so the NodeRedConfig.UnmarshalJSON hook
// does not consume the entire body and drop the
// ExpectedRevision field. The type is kept as a documented
// shape reference for the operator UI; it is not used as
// the decode target directly.
type ApplyStructuredRequest struct {
	Configuration    model.NodeRedConfig
	ExpectedRevision string
}

// ApplyStructuredResponse is the success envelope. The
// applied slice C endpoint returns the live SettingsDocument
// after the apply so the operator UI can re-bind its
// ExpectedRevision field without a follow-up GET.
type ApplyStructuredResponse struct {
	Configuration model.NodeRedConfig       `json:"configuration"`
	Document      model.SettingsDocument    `json:"document"`
}

// ApplyRawRequest is the wire payload for
// POST /api/config/apply/raw. Content is the operator-edited
// settings.js source; ExpectedRevision is the optional
// sha256 fingerprint of the bytes the operator read before
// composing the edit.
type ApplyRawRequest struct {
	Content          string `json:"content"`
	ExpectedRevision string `json:"expectedRevision,omitempty"`
}

// ApplyRawResponse mirrors ApplyStructuredResponse for the
// raw endpoint. Only the SettingsDocument is meaningful on
// this surface — the structured Configuration is the empty
// value because the raw editor does not update the JSON
// store.
type ApplyRawResponse struct {
	Document model.SettingsDocument `json:"document"`
}

// ApplyStructured handles POST /api/config/apply — admin only.
//
// Validation order:
//
//   1. Admin claims required (401 otherwise).
//   2. Configuration is editable (409 CONFIGURATION_READ_ONLY otherwise).
//   3. JSON payload decodes (400 INVALID_REQUEST otherwise).
//   4. JSON store commit succeeds (400 VALIDATION_ERROR otherwise).
//   5. ApplyCoordinator.Apply succeeds (5xx / 409 / 422 mapped by
//      writeApplyError).
//
// On success the response carries both the committed
// configuration (with bcrypt password hashes) and the live
// settings.js document so the operator UI can re-render
// without a follow-up round trip.
func (h *ConfigApplyHandler) ApplyStructured(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	capabilities := h.configSvc.ConfigurationCapabilities()
	if !capabilities.Editable {
		model.RespondError(w, http.StatusConflict, "CONFIGURATION_READ_ONLY", capabilities.Reason)
		return
	}

	var req ApplyStructuredRequest
	if !DecodeStructuredApplyPayload(w, r, &req) {
		return
	}

	staged, err := h.configSvc.CommitStructured(req.Configuration)
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	expected := staged.Live.Revision
	if req.ExpectedRevision != "" {
		expected = model.SourceRevision{
			Fingerprint: req.ExpectedRevision,
			Algorithm:   service.SourceRevisionAlgorithm,
		}
	}

	applyReq := service.ApplyRequest{
		Path:         staged.Live.Path,
		Content:      staged.Content,
		Expected:     expected,
		BackupDir:    staged.BackupDir,
		Actor:        claims.Username,
		Request:      r,
		Capabilities: capabilities,
	}
	if _, err := h.applyCoordinator.Apply(r.Context(), applyReq); err != nil {
		writeApplyError(w, err, "Failed to apply settings")
		return
	}

	if err := h.maybeRestart(); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "CONFIG_SAVED_RESTART_FAILED", "Configuration was applied, but Node-RED could not restart: "+err.Error())
		return
	}

	if h.audit != nil {
		h.audit.Log(r, claims.Username, "CONFIG_APPLY", staged.Live.Path, "ok", compatibilityAuditMeta(capabilities))
	}
	model.RespondJSON(w, http.StatusOK, ApplyStructuredResponse{
		Configuration: staged.Committed,
		Document:      h.refreshDocument(staged.Live.Path),
	})
}

// ApplyRaw handles POST /api/config/apply/raw — admin only.
// Unlike ApplyStructured the raw endpoint does NOT commit the
// JSON store; the raw editor lives outside NRCC's typed
// configuration model. The only state mutation is the
// settings.js bytes themselves, via the apply coordinator.
func (h *ConfigApplyHandler) ApplyRaw(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	capabilities := h.configSvc.ConfigurationCapabilities()
	if !capabilities.Editable {
		model.RespondError(w, http.StatusConflict, "CONFIGURATION_READ_ONLY", capabilities.Reason)
		return
	}

	var req ApplyRawRequest
	if !DecodeJSON(w, r, &req) {
		return
	}
	if req.Content == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Settings content cannot be empty")
		return
	}

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

	if err := h.maybeRestart(); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SETTINGS_SAVED_RESTART_FAILED", "settings.js was applied, but Node-RED could not restart: "+err.Error())
		return
	}

	if h.audit != nil {
		h.audit.Log(r, claims.Username, "SETTINGS_APPLY", live.Path, "ok", compatibilityAuditMeta(capabilities))
	}
	model.RespondJSON(w, http.StatusOK, ApplyRawResponse{
		Document: h.refreshDocument(live.Path),
	})
}

// maybeRestart triggers the Node-RED restart on a successful
// apply when a managed process is configured. The function
// returns the restart error so the caller can map it onto the
// existing CONFIG_SAVED_RESTART_FAILED / SETTINGS_SAVED_RESTART_FAILED
// envelopes. nil when no restart is needed (no process manager,
// external mode, or process not running).
func (h *ConfigApplyHandler) maybeRestart() error {
	if h.processManager == nil || h.processManager.IsExternalMode() {
		return nil
	}
	status := h.processManager.Status()
	if status.Status != "running" {
		return nil
	}
	return h.processManager.Restart()
}

// refreshDocument re-reads the settings.js document after a
// successful apply so the response carries the freshly
// fingerprinted Revision. The apply coordinator already
// stamped the new fingerprint inside its ApplyResult, but
// returning the document via GetRawSettings keeps the
// envelope identical to the legacy SETTINGS_UPDATE response
// shape.
func (h *ConfigApplyHandler) refreshDocument(path string) model.SettingsDocument {
	doc, err := h.configSvc.GetRawSettings()
	if err != nil {
		// Fall back to a minimal document carrying just the
		// path the operator targeted. The next read will
		// re-populate content + revision; the operator UI
		// does not need them in the immediate response.
		return model.SettingsDocument{Path: path}
	}
	return doc
}

// writeRevisionConflict writes the 409 SETTINGS_REVISION_CONFLICT
// envelope the slice C acceptance criteria pin. The envelope
// shape is:
//
//	{
//	  "success": false,
//	  "error":   { "code": "SETTINGS_REVISION_CONFLICT", "message": "..." },
//	  "data":    { "liveRevision": {...}, "providedRevision": "..." },
//	  "timestamp": "..."
//	}
//
// The function is shared by the structured and raw handlers so
// the operator UI can render a single conflict dialog for both
// surfaces. It is also used by ConfigHandler.saveConfigViaApply
// for the same reason.
// DecodeStructuredApplyPayload decodes a POST /api/config/apply
// body in two passes so the NodeRedConfig.UnmarshalJSON hook does
// not consume the entire payload and drop ExpectedRevision.
//
// The first pass decodes into a generic map; the second pass
// re-encodes that map minus the expectedRevision field and
// decodes the result as a NodeRedConfig so the typed
// UnmarshalJSON runs against the standard payload shape. The
// expectedRevision field is preserved as a string in dst.
//
// On any decode failure the function writes a 400 INVALID_REQUEST
// envelope and returns false so the handler can early-return.
func DecodeStructuredApplyPayload(w http.ResponseWriter, r *http.Request, dst *ApplyStructuredRequest) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return false
	}
	// First pass: generic map so we can pull out the slice C
	// specific field without losing it to a typed UnmarshalJSON.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return false
	}
	if msg, ok := raw["expectedRevision"]; ok {
		var fp string
		if err := json.Unmarshal(msg, &fp); err == nil {
			dst.ExpectedRevision = fp
		}
		delete(raw, "expectedRevision")
	}
	// Second pass: re-encode and decode as NodeRedConfig so
	// the typed UnmarshalJSON runs against the standard
	// payload shape.
	rest, err := json.Marshal(raw)
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return false
	}
	if err := json.Unmarshal(rest, &dst.Configuration); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return false
	}
	return true
}

func writeRevisionConflict(w http.ResponseWriter, conflict *service.RevisionConflictError) {
	resp := revisionConflictEnvelope{
		Success:   false,
		Error:     &model.ApiError{Code: "SETTINGS_REVISION_CONFLICT", Message: conflict.Error()},
		Data: revisionConflictData{
			LiveRevision:     conflict.Live,
			ProvidedRevision: conflict.Provided.Fingerprint,
		},
		Timestamp: model.NowISO8601(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(resp)
}

// revisionConflictEnvelope mirrors model.ApiResponse but lets
// us populate both Error AND Data on the same payload. The
// Data field uses omitempty in the standard envelope so
// handlers do not have to invent a typed zero value when they
// only have an error; here we explicitly want the structured
// data to ride along with the error.
type revisionConflictEnvelope struct {
	Success   bool                 `json:"success"`
	Error     *model.ApiError      `json:"error"`
	Data      revisionConflictData `json:"data"`
	Timestamp string               `json:"timestamp"`
}

type revisionConflictData struct {
	LiveRevision     model.SourceRevision `json:"liveRevision"`
	ProvidedRevision string               `json:"providedRevision"`
}

// backupDirFor computes the per-deployment backup directory
// the apply coordinator will write the pre-write snapshot
// into. The convention matches the one ConfigService.Save
// used before slice C: a sibling "backups/settings" directory
// next to the settings.js location.
//
// On managed Node-RED installs settings.js lives in the
// operator's home directory (e.g. /home/node-red/.node-red/
// settings.js) so a sibling-of-settings anchor keeps every
// deployment's backups colocated with its source. Existing
// backup-discovery tooling that scans `<dataDir>/backups/
// settings` keeps working for the canonical NRCC-managed
// install because ConfigService writes settings.js under
// dataDir by default — the two anchors coincide.
func backupDirFor(configSvc *service.ConfigService, settingsPath string) string {
	dir := filepath.Dir(settingsPath)
	return filepath.Join(dir, "backups", "settings")
}