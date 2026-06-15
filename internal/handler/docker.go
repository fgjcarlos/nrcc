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
//
// Two code paths share the same HTTP surface:
//
//  1. In-Docker path — nrcc itself runs inside a container. Status
//     answers with a synthetic running container (the parent
//     container); restart/stop signal the parent process so Docker's
//     restart policy can re-create the container.
//
//  2. Native path — nrcc runs natively on a host that has a
//     Node-RED container managed by the `docker` CLI. Status, info,
//     restart, and stop are routed to a DockerService that drives
//     the docker binary directly. The full ContainerInfo shape
//     (id, name, image, status, created, ports, state) is returned
//     so the existing frontend DockerView renders unchanged.
//
// In both cases the UI gets a usable response; the previous behaviour
// — always 503 for native hosts — is what issue #308 fixes.
type DockerHandler struct {
	pm           *service.ProcessManager
	shutdownCh   chan struct{}
	audit        *audit.Service
	dockerSvc    *service.DockerService
	dockerStatus service.DockerStatus
}

// NewDockerHandler creates a new docker handler.
func NewDockerHandler() *DockerHandler {
	return &DockerHandler{}
}

// SetAuditService injects the audit logger.
func (h *DockerHandler) SetAuditService(a *audit.Service) { h.audit = a }

// SetProcessManager injects the process manager so container restarts
// can stop Node-RED gracefully before exiting.
func (h *DockerHandler) SetProcessManager(pm *service.ProcessManager) {
	h.pm = pm
}

// SetShutdownChannel injects the shutdown channel so handlers can
// signal graceful shutdown instead of calling os.Exit.
func (h *DockerHandler) SetShutdownChannel(ch chan struct{}) {
	h.shutdownCh = ch
}

// SetDockerService injects the DockerService that powers the native
// path. The handler caches the latest Status() result so the
// synchronous restart/stop handlers can read the discovered container
// name without re-running discovery.
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

// containerInfo is the lightweight shape returned by GetStatus when in
// Docker. It has the same fields the frontend ContainerInfo type
// reads (id/name/image/status/created/ports/state), plus a synthesised
// id and created timestamp so the existing UI renders without
// special-casing the in-Docker path.
type containerInfo struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Image    string          `json:"image"`
	Status   string          `json:"status"`
	Created  string          `json:"created"`
	Ports    []containerPort `json:"ports,omitempty"`
	State    containerState  `json:"state"`
	InDocker bool            `json:"inDocker"`
}

type containerPort struct {
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort,omitempty"`
	Type        string `json:"type"`
}

type containerState struct {
	Running      bool    `json:"running"`
	Paused       bool    `json:"paused"`
	RestartCount int     `json:"restartCount"`
	Memory       int64   `json:"memory"`
	CPU          float64 `json:"cpu"`
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
			Name:     "nrcc",
			Image:    image,
			Status:   "running",
			InDocker: true,
			State: containerState{
				Running: true,
			},
		})
		return
	}
	// Native path: defer to the DockerService and unwrap to the
	// full containerInfo shape so the frontend doesn't need a
	// separate code path.
	h.refreshDockerStatus()
	if h.dockerStatus.Container != nil {
		c := h.dockerStatus.Container
		model.RespondJSON(w, http.StatusOK, containerInfo{
			ID:      c.ID,
			Name:    c.Name,
			Image:   c.Image,
			Status:  c.Status,
			Created: c.Created,
			Ports:   toHandlerPorts(c.Ports),
			State: containerState{
				Running:      c.State.Running,
				Paused:       c.State.Paused,
				RestartCount: c.State.RestartCount,
				Memory:       c.State.Memory,
				CPU:          c.State.CPU,
			},
		})
		return
	}
	model.RespondJSON(w, http.StatusOK, model.DockerStatus{
		Available: false,
		Message:   dockerNotAvailableMessage(h.dockerStatus),
	})
}

// GetInfo returns Docker engine info.
// GET /api/docker/info
func (h *DockerHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	if isInDocker() {
		model.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"inDocker": true,
		})
		return
	}
	h.refreshDockerStatus()
	if h.dockerStatus.Container != nil {
		model.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"available":     true,
			"inDocker":      false,
			"source":        h.dockerStatus.Source,
			"containerName": h.dockerStatus.Container.Name,
		})
		return
	}
	model.RespondJSON(w, http.StatusOK, model.DockerStatus{
		Available: false,
		Message:   dockerNotAvailableMessage(h.dockerStatus),
	})
}

// PostRestart restarts the container.
//   - In-Docker: stops Node-RED and signals shutdown so Docker's
//     restart policy brings everything back up.
//   - Native: runs `docker restart <name>` against the discovered
//     container and audit-logs the result.
//
// POST /api/docker/restart
func (h *DockerHandler) PostRestart(w http.ResponseWriter, r *http.Request) {
	if isInDocker() {
		h.audit.Log(r, "", "CONTAINER_RESTART", "", "ok", nil)
		model.RespondJSON(w, http.StatusOK, map[string]string{"message": "Container restarting…"})
		go func() {
			if h.pm != nil {
				_ = h.pm.Stop()
			}
			time.Sleep(300 * time.Millisecond)
			if h.shutdownCh != nil {
				h.shutdownCh <- struct{}{}
			}
		}()
		return
	}
	if h.dockerSvc == nil {
		model.RespondError(w, http.StatusServiceUnavailable, "DOCKER_NOT_AVAILABLE",
			"Docker management is not configured on this host")
		return
	}
	if err := h.dockerSvc.Restart(); err != nil {
		h.audit.Log(r, "", "CONTAINER_RESTART", "", "error",
			map[string]string{"error": err.Error()})
		model.RespondError(w, http.StatusServiceUnavailable, "DOCKER_NOT_AVAILABLE",
			err.Error())
		return
	}
	h.audit.Log(r, "", "CONTAINER_RESTART", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, map[string]string{"message": "Container restarting…"})
}

// PostStop stops the container.
//   - In-Docker: stops Node-RED and signals shutdown.
//   - Native: runs `docker stop <name>` against the discovered
//     container and audit-logs the result.
//
// POST /api/docker/stop
func (h *DockerHandler) PostStop(w http.ResponseWriter, r *http.Request) {
	if isInDocker() {
		h.audit.Log(r, "", "CONTAINER_STOP", "", "ok", nil)
		model.RespondJSON(w, http.StatusOK, map[string]string{"message": "Container stopping…"})
		go func() {
			if h.pm != nil {
				_ = h.pm.Stop()
			}
			time.Sleep(300 * time.Millisecond)
			if h.shutdownCh != nil {
				h.shutdownCh <- struct{}{}
			}
		}()
		return
	}
	if h.dockerSvc == nil {
		model.RespondError(w, http.StatusServiceUnavailable, "DOCKER_NOT_AVAILABLE",
			"Docker management is not configured on this host")
		return
	}
	if err := h.dockerSvc.Stop(); err != nil {
		h.audit.Log(r, "", "CONTAINER_STOP", "", "error",
			map[string]string{"error": err.Error()})
		model.RespondError(w, http.StatusServiceUnavailable, "DOCKER_NOT_AVAILABLE",
			err.Error())
		return
	}
	h.audit.Log(r, "", "CONTAINER_STOP", "", "ok", nil)
	model.RespondJSON(w, http.StatusOK, map[string]string{"message": "Container stopping…"})
}

// toHandlerPorts converts service-level ContainerPort values to the
// handler-level containerPort struct. Kept local so the JSON output
// stays stable when the service grows new fields.
func toHandlerPorts(in []service.ContainerPort) []containerPort {
	if len(in) == 0 {
		return nil
	}
	out := make([]containerPort, 0, len(in))
	for _, p := range in {
		out = append(out, containerPort{
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		})
	}
	return out
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
