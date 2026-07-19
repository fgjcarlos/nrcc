package handler

import (
	"net/http"
	"os"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// DockerHandler serves the lightweight container-status read the dashboard
// uses. Mutating endpoints (restart/stop) and the engine-info endpoint were
// removed in #477 — restart/stop on nrcc's own container is structurally
// meaningless under the docker-first model, and the engine-info response
// was never consumed outside the (also-removed) /docker page.
type DockerHandler struct {
	dockerSvc    *service.DockerService
	dockerStatus service.DockerStatus
}

// NewDockerHandler creates a new docker handler.
func NewDockerHandler() *DockerHandler {
	return &DockerHandler{}
}

// SetDockerService injects the DockerService that powers the native
// path. The handler caches the latest Status() result so subsequent
// calls don't pay the docker-cli discovery cost.
func (h *DockerHandler) SetDockerService(svc *service.DockerService) {
	h.dockerSvc = svc
}

// refreshDockerStatus updates the cached status payload. The HTTP
// handlers call this lazily so they always work from the freshest
// state.
func (h *DockerHandler) refreshDockerStatus() {
	if h.dockerSvc == nil {
		return
	}
	h.dockerStatus = h.dockerSvc.Status()
}

// isInDocker returns true when running inside a Docker container.
func isInDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

// GetStatus returns container status.
// GET /api/docker/status
func (h *DockerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if isInDocker() {
		image := os.Getenv("NRCC_IMAGE")
		if image == "" {
			image = "nrcc"
		}
		model.RespondJSON(w, http.StatusOK, &service.ContainerInfo{
			Name:     "nrcc",
			Image:    image,
			Status:   "running",
			InDocker: true,
			State: service.ContainerState{
				Running: true,
			},
		})
		return
	}
	h.refreshDockerStatus()
	if h.dockerStatus.Container != nil {
		c := h.dockerStatus.Container
		c.InDocker = false
		model.RespondJSON(w, http.StatusOK, c)
		return
	}
	model.RespondJSON(w, http.StatusOK, model.DockerStatus{
		Available: false,
		Message:   dockerNotAvailableMessage(h.dockerStatus),
	})
}

// dockerNotAvailableMessage returns a human-readable message for the
// not-available payload. The service may set a custom message
// (e.g. "No Node-RED container found"); when it does not, fall back
// to a generic explanation that covers both missing-CLI and
// missing-container cases.
func dockerNotAvailableMessage(s service.DockerStatus) string {
	if s.Message != "" {
		return s.Message
	}
	return "Docker mode not available — no Node-RED container found"
}
