package handler

import (
	"net/http"
	"os"
	"time"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// DockerHandler handles Docker-related endpoints.
// When nrcc runs inside a Docker container it exposes real restart/stop actions;
// otherwise the endpoints return 503 so the UI disables them.
type DockerHandler struct {
	pm         *service.ProcessManager
	shutdownCh chan struct{}
	audit      *audit.Service
}

// NewDockerHandler creates a new docker handler.
func NewDockerHandler() *DockerHandler {
	return &DockerHandler{}
}

// SetAuditService injects the audit logger.
func (h *DockerHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetProcessManager injects the process manager so container restarts can
// stop Node-RED gracefully before exiting.
func (h *DockerHandler) SetProcessManager(pm *service.ProcessManager) {
	h.pm = pm
}

// SetShutdownChannel injects the shutdown channel so handlers can signal
// graceful shutdown instead of calling os.Exit.
func (h *DockerHandler) SetShutdownChannel(ch chan struct{}) {
	h.shutdownCh = ch
}

// isInDocker returns true when running inside a Docker container.
func isInDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

// containerInfo is the lightweight shape returned by GetStatus when in Docker.
// It has the same fields the frontend ContainerInfo type reads (status, image).
type containerInfo struct {
	Status   string `json:"status"`
	Image    string `json:"image,omitempty"`
	InDocker bool   `json:"inDocker"`
}

// GetStatus returns container status.
// GET /api/docker/status
func (h *DockerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if isInDocker() {
		image := os.Getenv("NRCC_IMAGE")
		if image == "" {
			image = "nrcc"
		}
		model.RespondJSON(w, http.StatusOK, containerInfo{
			Status:   "running",
			Image:    image,
			InDocker: true,
		})
		return
	}
	model.RespondJSON(w, http.StatusOK, model.DockerStatus{
		Available: false,
		Message:   "Docker mode not available — binary manages Node-RED directly",
	})
}

// GetInfo returns Docker engine info (stub).
// GET /api/docker/info
func (h *DockerHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	if isInDocker() {
		model.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"inDocker": true,
		})
		return
	}
	model.RespondJSON(w, http.StatusOK, model.DockerStatus{
		Available: false,
		Message:   "Docker mode not available — binary manages Node-RED directly",
	})
}

// PostRestart restarts the container by stopping Node-RED and signaling shutdown.
// Docker's restart policy (always / unless-stopped) brings everything back up.
// POST /api/docker/restart
func (h *DockerHandler) PostRestart(w http.ResponseWriter, r *http.Request) {
	if !isInDocker() {
		model.RespondError(w, http.StatusServiceUnavailable, "DOCKER_NOT_AVAILABLE",
			"Not running inside a Docker container")
		return
	}
	h.audit.Log(r, "", "CONTAINER_RESTART", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, map[string]string{"message": "Container restarting…"})
	go func() {
		if h.pm != nil {
			h.pm.Stop()
		}
		time.Sleep(300 * time.Millisecond)
		if h.shutdownCh != nil {
			h.shutdownCh <- struct{}{}
		}
	}()
}

// PostStop stops the container (same mechanism — Docker policy controls restart).
// POST /api/docker/stop
func (h *DockerHandler) PostStop(w http.ResponseWriter, r *http.Request) {
	if !isInDocker() {
		model.RespondError(w, http.StatusServiceUnavailable, "DOCKER_NOT_AVAILABLE",
			"Not running inside a Docker container")
		return
	}
	h.audit.Log(r, "", "CONTAINER_STOP", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, map[string]string{"message": "Container stopping…"})
	go func() {
		if h.pm != nil {
			h.pm.Stop()
		}
		time.Sleep(300 * time.Millisecond)
		if h.shutdownCh != nil {
			h.shutdownCh <- struct{}{}
		}
	}()
}
