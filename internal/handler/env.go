package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// EnvHandler handles environment variable endpoints
type EnvHandler struct {
	svc     *service.EnvService
	pm      *service.ProcessManager
	audit   *audit.Service
	dataDir string
	mu      sync.Mutex
}

// NewEnvHandler creates a new environment handler
func NewEnvHandler(svc *service.EnvService, dataDir string) *EnvHandler {
	return &EnvHandler{
		svc:     svc,
		dataDir: dataDir,
	}
}

// SetProcessManager wires a ProcessManager so node-red is restarted automatically
// whenever an env var is saved or deleted.
func (h *EnvHandler) SetProcessManager(pm *service.ProcessManager) { h.pm = pm }

// SetAuditService injects the audit logger.
func (h *EnvHandler) SetAuditService(a *audit.Service) { h.audit = a }

// GetEnv lists all environment variables
// GET /api/env
func (h *EnvHandler) GetEnv(w http.ResponseWriter, r *http.Request) {
	envVars, err := h.svc.List()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "ENV_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, envVars)
}

// PostEnv sets an environment variable and restarts node-red if it is running
// POST /api/env
func (h *EnvHandler) PostEnv(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		Type        string `json:"type,omitempty"`
		Description string `json:"description,omitempty"`
		Encrypted   bool   `json:"encrypted,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := service.ValidateEnvKey(req.Key); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Default type to "string" if not provided
	if req.Type == "" {
		req.Type = "string"
	}

	// Phase 2.2: Validate value for the given type
	if err := service.ValidateValue(req.Value, req.Type); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_VALUE", err.Error())
		return
	}

	// Phase 2.2: Normalize value to canonical form
	normalizedValue, err := service.NormalizeValue(req.Value, req.Type)
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_VALUE", err.Error())
		return
	}

	// Derive Encrypted flag from type (secret → encrypted=true)
	encrypted := (req.Type == "secret")

	restarted, err := h.withManagedNodeRedStopped(func() error {
		return h.svc.Set(req.Key, normalizedValue, req.Type, req.Description, encrypted)
	})
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "ENV_ERROR", err.Error())
		return
	}

	h.audit.Log(r, "", "ENV_SET", req.Key, "ok", map[string]string{"type": req.Type})
	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Environment variable set",
		"restarted": restarted,
	})
}

// DeleteEnv deletes an environment variable and restarts node-red if it is running
// DELETE /api/env/{key}
func (h *EnvHandler) DeleteEnv(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	restarted, err := h.withManagedNodeRedStopped(func() error { return h.svc.Delete(key) })
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "ENV_ERROR", err.Error())
		return
	}

	h.audit.Log(r, "", "ENV_DELETE", key, "ok", nil)
	if restarted {
		model.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "Environment variable deleted",
			"restarted": true,
		})
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// withManagedNodeRedStopped prevents NRCC from racing Node-RED's own writes to
// flows.json. Environment mutations are synchronous so success means both the
// persisted NRCC store and the Node-RED 5 global-config are ready.
func (h *EnvHandler) withManagedNodeRedStopped(change func() error) (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.pm == nil {
		return false, change()
	}
	if h.pm.IsExternalMode() {
		return false, fmt.Errorf("cannot synchronize environment variables for externally managed Node-RED")
	}

	wasRunning := h.pm.Status().Status == "running"
	if wasRunning {
		if err := h.pm.Stop(); err != nil {
			return false, fmt.Errorf("stop Node-RED before environment update: %w", err)
		}
	}

	if err := change(); err != nil {
		if wasRunning {
			if restartErr := h.pm.Start(); restartErr != nil {
				return true, fmt.Errorf("%w; restart Node-RED after failed update: %v", err, restartErr)
			}
		}
		return wasRunning, err
	}

	if wasRunning {
		if err := h.pm.Start(); err != nil {
			return true, fmt.Errorf("restart Node-RED after environment update: %w", err)
		}
	}
	return wasRunning, nil
}

// restartIfRunning restarts the node-red process if it is currently running.
// Returns true if a restart was triggered.
func (h *EnvHandler) restartIfRunning() bool {
	if h.pm == nil {
		return false
	}
	status := h.pm.Status()
	if status.Status != "running" {
		return false
	}
	// Restart in background — don't block the HTTP response
	go func() { _ = h.pm.Restart() }()
	return true
}

// GetDotenv returns the content of data/.env
// GET /api/env/dotenv
func (h *EnvHandler) GetDotenv(w http.ResponseWriter, r *http.Request) {
	content, err := service.ReadDotenv(h.dataDir)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "DOTENV_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]string{
		"content": content,
	})
}

// PutDotenv saves the .env file content
// PUT /api/env/dotenv
func (h *EnvHandler) PutDotenv(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := service.WriteDotenv(h.dataDir, req.Content); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "DOTENV_ERROR", err.Error())
		return
	}

	restarted := h.restartIfRunning()
	h.audit.Log(r, "", "DOTENV_UPDATE", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Archivo .env guardado",
		"restarted": restarted,
	})
}
