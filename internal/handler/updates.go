package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// updateMetricsRecorder is the narrow interface for recording update attempt metrics.
// Using an interface instead of *metrics.MetricsCollector keeps UpdateHandler
// testable with simple stubs and avoids a direct dependency on the metrics package.
type updateMetricsRecorder interface {
	RecordUpdateAttempt(success bool)
}

// UpdateHandler handles Node-RED update endpoints.
//
// API Overview:
// - GET /api/updates/status — Returns cached update status (fast, no npm call)
// - GET /api/updates/check — Performs fresh update check via npm (slower, up to 10s)
// - POST /api/updates/apply — Applies the latest update via npm install
// - GET /api/updates/history — Returns update history log (currently empty)
//
// Cache Strategy:
// The backend maintains a persistent cache (./data/update_cache.json) that is updated:
// 1. Once per configured poll interval (default 4 hours) by a background goroutine
// 2. Immediately when a manual check is triggered via GET /api/updates/check
//
// Response Format:
// All endpoints return JSON with fields:
// - currentVersion: string (e.g., "4.0.1")
// - latestVersion: string (e.g., "4.0.2")
// - updateAvailable: boolean (true if latestVersion > currentVersion)
// - checkedAt: RFC3339 timestamp (when the check was last performed)
// - error: string (optional, present if npm call failed)
type UpdateHandler struct {
	svc           *service.UpdateService
	audit         *audit.Service
	updateMetrics updateMetricsRecorder
}

// NewUpdateHandler creates a new update handler
func NewUpdateHandler(svc *service.UpdateService) *UpdateHandler {
	return &UpdateHandler{svc: svc}
}

// SetAuditService injects the audit logger.
func (h *UpdateHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetUpdateMetrics injects the metrics recorder for update attempts.
func (h *UpdateHandler) SetUpdateMetrics(m updateMetricsRecorder) { h.updateMetrics = m }

// GetStatus returns the cached update status.
// GET /api/updates/status
//
// Response: UpdateCacheEntry (cached from last polling interval or manual check)
// Timing: <1ms (in-memory read, no I/O)
// Never spawns npm process.
func (h *UpdateHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.svc.GetCachedStatus()

	// Always return 200 OK with the cached status
	model.RespondJSON(w, http.StatusOK, status)
}

// GetCheck forces an immediate update check.
// GET /api/updates/check
//
// Response: UpdateCacheEntry (fresh result from npm)
// Timing: up to 10 seconds (npm timeout) + network latency
// Concurrent checks are deduplicated (only one npm call if multiple requests arrive simultaneously).
// Frontend should call this endpoint for manual "Check Now" button clicks.
func (h *UpdateHandler) GetCheck(w http.ResponseWriter, r *http.Request) {
	// Use request context with a timeout
	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()

	entry, err := h.svc.ForceCheck(ctx)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, entry)
}

// PostApply applies the latest update with automatic backup.
// POST /api/updates/apply
//
// Flow:
// 1. Check if state is Idle; if not, return 409 Conflict
// 2. Launch asynchronous update flow: BackingUp → Applying → Completed/Failed
// 3. Return 200 OK immediately with current state + backupId (if available)
//
// Frontend polls GET /api/updates/state every 500ms to track progress.
// Once state reaches Completed/Failed, polling stops.
//
// Response on success: {success: true, state: "BackingUp", backupId: "..."}
// Response on conflict: HTTP 409 {success: false, error: "update already in progress"}
func (h *UpdateHandler) PostApply(w http.ResponseWriter, r *http.Request) {
	// Check if an update is already in progress
	flowState := h.svc.GetFlowState()
	if flowState.State != model.StateIdle {
		// Another update is already in progress
		model.RespondError(w, http.StatusConflict, "UPDATE_IN_PROGRESS", 
			"An update is already in progress. Please wait for it to complete.")
		return
	}

	// Capture pre-apply status to report version transition (for logging/auditing)
	preApplyStatus := h.svc.GetCachedStatus()
	fromVersion := preApplyStatus.CurrentVersion

	// Capture fields needed in the goroutine to avoid capturing the handler receiver.
	svc := h.svc
	updateMetrics := h.updateMetrics

	// Launch the update flow asynchronously.
	// This does NOT block the HTTP response; frontend polls for progress.
	go func() {
		ctx := context.Background() // Background context; not tied to HTTP request lifetime
		err := svc.ApplyUpdateWithBackup(ctx)
		if updateMetrics != nil {
			// Determine success from the final flow state set by ApplyUpdateWithBackup.
			finalState := svc.GetFlowState()
			updateMetrics.RecordUpdateAttempt(err == nil && finalState.State == model.StateCompleted)
		}
		if err != nil {
			// Update failed; state is already set to Failed by ApplyUpdateWithBackup
			// Log for debugging
			_ = err // Error already captured in flowState; frontend can see it via GET /state
		}
	}()

	// Get current state (should be BackingUp or Applying since we just launched the flow)
	flowState = h.svc.GetFlowState()
	toVersion := preApplyStatus.LatestVersion
	if toVersion == "" {
		toVersion = "latest"
	}

	h.audit.Log(r, "", "UPDATE_APPLY", "", "ok", map[string]string{"from": fromVersion, "to": toVersion})

	// Return 200 OK with current state
	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"message":      "Update flow started",
		"state":        flowState.State,
		"backupId":     flowState.BackupID,
		"fromVersion":  fromVersion,
		"toVersion":    toVersion,
	})
}

// GetHistory returns update history (stub).
// GET /api/updates/history
//
// Returns the applied-update backup catalog (most recent updates).
func (h *UpdateHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	model.RespondJSON(w, http.StatusOK, h.svc.History())
}

// GetState returns the current update flow state.
// GET /api/updates/state
//
// Returns: UpdateFlowState with current state, phase, backupId, error, availableVersion.
// This endpoint is polled by frontend every 500ms while an update is in progress.
// Response format: {state: "Idle|BackingUp|Applying|Completed|Failed", phase: "...", ...}
func (h *UpdateHandler) GetState(w http.ResponseWriter, r *http.Request) {
	flowState := h.svc.GetFlowState()
	model.RespondJSON(w, http.StatusOK, flowState)
}
