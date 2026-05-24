package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/composedof2/nrcc/internal/audit"
	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// backupMetricsRecorder is the narrow interface for recording backup/restore metrics.
// Using an interface instead of *metrics.MetricsCollector keeps BackupHandler
// testable with simple stubs and avoids a direct dependency on the metrics package.
type backupMetricsRecorder interface {
	RecordBackupCreated(backupType string)
	RecordRestoreAttempt(success bool)
}

// BackupHandler handles backup endpoints
type BackupHandler struct {
	svc           *service.BackupService
	audit         *audit.Service
	backupMetrics backupMetricsRecorder
}

// GetBackupStatus returns runtime scheduler status.
// GET /api/backups/status
func (h *BackupHandler) GetBackupStatus(w http.ResponseWriter, r *http.Request) {
	model.RespondJSON(w, http.StatusOK, h.svc.SchedulerStatus())
}

// GetBackupObservability returns recent backup/scheduler observability details.
// GET /api/backups/observability
func (h *BackupHandler) GetBackupObservability(w http.ResponseWriter, r *http.Request) {
	observability, err := h.svc.Observability()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, observability)
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(svc *service.BackupService) *BackupHandler {
	return &BackupHandler{svc: svc}
}

// SetAuditService injects the audit logger.
func (h *BackupHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetBackupMetrics injects the metrics recorder for backup/restore operations.
func (h *BackupHandler) SetBackupMetrics(m backupMetricsRecorder) { h.backupMetrics = m }

// GetBackups lists all backups with optional pagination, sorting, and filtering
// GET /api/backups?page=1&limit=20&sort=date|size|status&order=asc|desc
func (h *BackupHandler) GetBackups(w http.ResponseWriter, r *http.Request) {
	// Parse pagination query parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "date"
	}

	order := r.URL.Query().Get("order")
	if order == "" {
		order = "desc"
	}

	// Call ListPaginated service method (Task 1.5: Refactor to use ListPaginated)
	opts := model.PaginationOpts{
		Page:  page,
		Limit: limit,
		Sort:  sort,
		Order: order,
	}

	result, err := h.svc.ListPaginated(opts)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, result)
}

// PostBackup creates a new backup
// POST /api/backups
func (h *BackupHandler) PostBackup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name,omitempty"`
		Type string `json:"type,omitempty"`
	}

	// Try to parse JSON body if present
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Continue with empty name on parse error
		}
	}

	backup, err := h.svc.CreateTyped(model.BackupType(req.Type), req.Name)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	if h.backupMetrics != nil {
		h.backupMetrics.RecordBackupCreated(string(backup.Type))
	}
	h.audit.Log(r, "", "BACKUP_CREATE", backup.ID, "ok", map[string]string{"type": req.Type})
	model.RespondJSON(w, http.StatusCreated, backup)
}

// GetBackupDetail returns metadata for a specific backup.
// GET /api/backups/{id}
func (h *BackupHandler) GetBackupDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	manifest, err := h.svc.Detail(id)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, manifest)
}

// DeleteBackup deletes a backup
// DELETE /api/backups/{id}
func (h *BackupHandler) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.svc.Delete(id)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	h.audit.Log(r, "", "BACKUP_DELETE", id, "ok", nil)
	w.WriteHeader(http.StatusNoContent)
}

// DownloadBackup downloads a backup
// GET /api/backups/{id}/download
func (h *BackupHandler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Set response headers for file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"backup-"+id+".zip\"")

	err := h.svc.Download(id, w)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}
}

// RestoreBackup restores a backup
// POST /api/backups/{id}/restore
func (h *BackupHandler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	preRestoreID, err := h.svc.RestoreWithSafetyBackup(id)
	if err != nil {
		if h.backupMetrics != nil {
			h.backupMetrics.RecordRestoreAttempt(false)
		}
		h.audit.Log(r, "", "BACKUP_RESTORE", id, "fail", map[string]string{"error": err.Error()})
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	if h.backupMetrics != nil {
		h.backupMetrics.RecordRestoreAttempt(true)
	}
	h.audit.Log(r, "", "BACKUP_RESTORE", id, "ok", map[string]string{"pre_restore_id": preRestoreID})
	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"message":      "Backup restored successfully",
		"preRestoreId": preRestoreID,
	})
}

// GetBackupStorage returns aggregate backup storage stats.
// GET /api/backups/storage
func (h *BackupHandler) GetBackupStorage(w http.ResponseWriter, r *http.Request) {
	storage, err := h.svc.Storage()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, storage)
}

// GetBackupConfig gets persisted backup configuration.
// GET /api/backups/config
func (h *BackupHandler) GetBackupConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.svc.GetConfig()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, config)
}

// PostBackupConfig saves backup configuration.
// POST /api/backups/config
func (h *BackupHandler) PostBackupConfig(w http.ResponseWriter, r *http.Request) {
	var req model.BackupConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	config, err := h.svc.SaveConfig(req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidBackupConfig) {
			model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, config)
}

// PostSchedulerConfig saves scheduler configuration with cron validation.
// POST /api/scheduler/config
func (h *BackupHandler) PostSchedulerConfig(w http.ResponseWriter, r *http.Request) {
	var req model.SchedulerConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate cron expression
	if !service.IsValidCron(req.Cron) {
		model.RespondError(w, http.StatusBadRequest, "INVALID_CRON", "Invalid cron expression: "+req.Cron)
		return
	}

	// Update config with new cron
	config, err := h.svc.GetConfig()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	config.Schedule = "custom"
	config.CustomSchedule = req.Cron

	if _, err := h.svc.SaveConfig(config); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	response := model.SchedulerConfigResponse{
		Cron:  req.Cron,
		Valid: true,
	}

	model.RespondJSON(w, http.StatusOK, response)
}

// GetSchedulerHistory returns paginated scheduler execution history.
// GET /api/scheduler/history?page=1&limit=10
func (h *BackupHandler) GetSchedulerHistory(w http.ResponseWriter, r *http.Request) {
	// Parse pagination query parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	opts := model.PaginationOpts{
		Page:  page,
		Limit: limit,
	}

	result, err := h.svc.GetSchedulerHistory(opts)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, result)
}

// PatchStorageRetention updates retention configuration.
// PATCH /api/storage/retention
func (h *BackupHandler) PatchStorageRetention(w http.ResponseWriter, r *http.Request) {
	var req model.RetentionConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate retention days
	if req.RetentionDays < 1 || req.RetentionDays > 3650 {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Retention days must be between 1 and 3650")
		return
	}

	// Get current config
	config, err := h.svc.GetConfig()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	// Update retention settings
	config.RetentionManual = req.RetentionDays
	config.RetentionAuto = req.RetentionDays
	config.RetentionPreRestore = req.RetentionDays

	if _, err := h.svc.SaveConfig(config); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "BACKUP_ERROR", err.Error())
		return
	}

	response := model.RetentionConfigResponse{
		RetentionDays: req.RetentionDays,
	}

	model.RespondJSON(w, http.StatusOK, response)
}

// Helper functions
