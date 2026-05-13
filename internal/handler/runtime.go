package handler

import (
	"net/http"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// RuntimeHandler handles Node-RED runtime endpoints
type RuntimeHandler struct {
	pm      *service.ProcessManager
	hostSvc *service.HostService
}

// NewRuntimeHandler creates a new RuntimeHandler
func NewRuntimeHandler(pm *service.ProcessManager, hostSvc *service.HostService) *RuntimeHandler {
	return &RuntimeHandler{
		pm:      pm,
		hostSvc: hostSvc,
	}
}

// SetProcessManager updates the runtime handler after server bootstrap.
func (h *RuntimeHandler) SetProcessManager(pm *service.ProcessManager) {
	h.pm = pm
}

// GetStatus returns the current runtime status (GET /api/runtime/status)
func (h *RuntimeHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if h.pm == nil {
		model.RespondJSON(w, http.StatusOK, h.hostSvc.RuntimeStatus())
		return
	}
	status := h.pm.Status()
	model.RespondJSON(w, http.StatusOK, status)
}

// GetUptime returns the current uptime in seconds (GET /api/runtime/uptime)
func (h *RuntimeHandler) GetUptime(w http.ResponseWriter, r *http.Request) {
	if h.pm == nil {
		model.RespondJSON(w, http.StatusOK, map[string]interface{}{"uptime": int64(0)})
		return
	}
	status := h.pm.Status()
	uptime := map[string]interface{}{
		"uptime": status.Uptime,
	}
	model.RespondJSON(w, http.StatusOK, uptime)
}

// PostRestart restarts the Node-RED process (POST /api/runtime/restart)
func (h *RuntimeHandler) PostRestart(w http.ResponseWriter, r *http.Request) {
	if h.pm == nil {
		model.RespondError(w, http.StatusConflict, "RUNTIME_UNMANAGED", "Node-RED is detected but not managed directly by nrcc")
		return
	}
	if err := h.pm.Restart(); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "RUNTIME_ERROR", err.Error())
		return
	}

	response := map[string]interface{}{
		"message": "Node-RED restarted successfully",
	}
	model.RespondJSON(w, http.StatusOK, response)
}

// StartNodeRed starts the Node-RED process (POST /api/runtime/start)
func (h *RuntimeHandler) StartNodeRed(w http.ResponseWriter, r *http.Request) {
	if h.pm == nil {
		model.RespondError(w, http.StatusConflict, "RUNTIME_UNMANAGED", "Node-RED no está gestionado por nrcc")
		return
	}
	status := h.pm.Status()
	if status.Status == "running" {
		model.RespondError(w, http.StatusConflict, "RUNTIME_ALREADY_RUNNING", "Node-RED ya está en ejecución")
		return
	}
	if err := h.pm.Start(); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "RUNTIME_ERROR", err.Error())
		return
	}

	response := map[string]interface{}{
		"message": "Node-RED iniciado",
	}
	model.RespondJSON(w, http.StatusOK, response)
}

// StopNodeRed stops the Node-RED process (POST /api/runtime/stop)
func (h *RuntimeHandler) StopNodeRed(w http.ResponseWriter, r *http.Request) {
	if h.pm == nil {
		model.RespondError(w, http.StatusConflict, "RUNTIME_UNMANAGED", "Node-RED no está gestionado por nrcc")
		return
	}
	status := h.pm.Status()
	if status.Status != "running" {
		model.RespondError(w, http.StatusConflict, "RUNTIME_NOT_RUNNING", "Node-RED no está en ejecución")
		return
	}
	if err := h.pm.Stop(); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "RUNTIME_ERROR", err.Error())
		return
	}

	response := map[string]interface{}{
		"message": "Node-RED detenido",
	}
	model.RespondJSON(w, http.StatusOK, response)
}
