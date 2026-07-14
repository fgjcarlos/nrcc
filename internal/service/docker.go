package service

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DockerService manages a Node-RED container that is owned by the
// Docker CLI (not by nrcc itself). It is the native-host companion to
// the in-container path implemented in internal/handler/docker.go.
//
// The service intentionally talks to the `docker` binary via the same
// exec shim used by HostService; no Docker socket client is added.
// Discovery mirrors the heuristic in
// HostService.inspectDockerNodeRed: a container qualifies when its
// image or name contains "node-red" (case-insensitive). The first
// match is authoritative.
type DockerService struct {
	// execCommand is the exec shim used to invoke `docker`. It
	// defaults to dockerExecCommand (a package variable that wraps
	// exec.Command) so tests can override it per-instance.
	execCommand func(name string, arg ...string) *exec.Cmd
	// lookPath is the os/exec.LookPath shim used to check that the
	// `docker` binary is available on the host. Optional: nil falls
	// back to exec.LookPath.
	lookPath func(name string) (string, error)
}

// NewDockerService builds a DockerService that uses the package-level
// exec shims so tests can stub them out.
func NewDockerService() *DockerService {
	return &DockerService{
		execCommand: dockerExecCommand,
		lookPath:    exec.LookPath,
	}
}

// WithLookPath overrides the binary-lookup function used to check that
// the `docker` binary is available on the host.
//
// This helper is kept ONLY for the docker binary because its install
// path is configurable in practice (system docker on /usr/bin/docker,
// docker-desktop on /usr/local/bin/docker, custom installs under
// /opt/docker/bin/docker). For systemd binaries (systemctl,
// journalctl) and pgrep the well-known paths are used directly so
// tests do not have to mock a lookup helper.
//
// nil is a no-op so callers can chain it without nil-checking.
func (d *DockerService) WithLookPath(fn func(name string) (string, error)) *DockerService {
	if d == nil || fn == nil {
		return d
	}
	d.lookPath = fn
	return d
}

// WithExecCommand overrides the exec shim used to invoke the docker
// binary. nil restores the package default. Useful for tests; not
// required in production because the default wraps exec.Command.
func (d *DockerService) WithExecCommand(fn func(name string, arg ...string) *exec.Cmd) *DockerService {
	if d == nil {
		return d
	}
	if fn == nil {
		d.execCommand = dockerExecCommand
		return d
	}
	d.execCommand = fn
	return d
}

// DockerStatus mirrors the lightweight shape returned by
// internal/handler/docker.go for the in-Docker path. It is reused by
// the native path so the frontend keeps a single mental model.
type DockerStatus struct {
	Available   bool            `json:"available"`
	InDocker    bool            `json:"inDocker,omitempty"`
	Source      string          `json:"source,omitempty"`
	Message     string          `json:"message,omitempty"`
	Container   *ContainerInfo  `json:"container,omitempty"`
}

// ContainerInfo is the shape the frontend's DockerView consumes.
// Fields mirror the TypeScript ContainerInfo in
// frontend/src/shared/types/index.ts.
type ContainerInfo struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Image   string          `json:"image"`
	Status  string          `json:"status"`
	Created string          `json:"created"`
	Ports   []ContainerPort `json:"ports"`
	State   ContainerState  `json:"state"`
}

// ContainerPort mirrors frontend PortMapping.
type ContainerPort struct {
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort,omitempty"`
	Type        string `json:"type"`
}

// ContainerState mirrors frontend ContainerState.
type ContainerState struct {
	Running      bool    `json:"running"`
	Paused       bool    `json:"paused"`
	RestartCount int     `json:"restartCount"`
	Memory       int64   `json:"memory"`
	CPU          float64 `json:"cpu"`
}

// errDockerNotFound is returned by Discover/Status when the `docker`
// binary is missing, when no Node-RED container is present, or when
// `docker ps` itself fails. The HTTP layer turns it into a 503.
var errDockerNotFound = errors.New("docker node-red container not available")

// dockerExecCommand is package-local so we can stub it in tests.
var dockerExecCommand = exec.Command

// Available reports whether the `docker` binary is on $PATH.
func (d *DockerService) Available() bool {
	if d == nil || d.lookPath == nil {
		return false
	}
	_, err := d.lookPath("docker")
	return err == nil
}

// DiscoverNodeRed returns the first Node-RED container on the host, or
// errDockerNotFound if none is reachable. The container fields are
// populated from `docker ps -a`; the State block is filled in by a
// follow-up `docker inspect` so memory/CPU/restartCount can be
// reported (not available from `docker ps` alone).
func (d *DockerService) DiscoverNodeRed() (*ContainerInfo, error) {
	if d == nil {
		return nil, errDockerNotFound
	}
	if d.execCommand == nil {
		return nil, errDockerNotFound
	}
	if !d.Available() {
		return nil, errDockerNotFound
	}

	// `docker ps -a --format` with a custom Go-template gives us a
	// stable, parseable line per container. We use tabs as separators
	// to match the heuristic in HostService.inspectDockerNodeRed.
	psCmd := d.execCommand("docker", "ps", "-a",
		"--format", "{{.ID}}\t{{.Image}}\t{{.Names}}\t{{.Status}}\t{{.CreatedAt}}")
	psOut, err := psCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: docker ps: %v", errDockerNotFound, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(psOut))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 5 {
			continue
		}
		image := strings.ToLower(parts[1])
		name := parts[2]
		if !strings.Contains(image, "node-red") && !strings.Contains(strings.ToLower(name), "node-red") {
			continue
		}
		info := &ContainerInfo{
			ID:      parts[0],
			Name:    name,
			Image:   parts[1],
			Status:  mapContainerStatus(parts[3]),
			Created: parts[4],
		}
		// Best-effort: enrich with state. Don't fail the whole
		// discovery if `docker inspect` errors out.
		d.fillContainerState(info, parts[0])
		return info, nil
	}
	return nil, errDockerNotFound
}

// Status returns the DockerStatus payload for a native host. It is the
// caller-friendly entry point for the HTTP layer: on success it
// includes the full ContainerInfo, on a missing CLI / container it
// returns a structured "not available" payload with a human message.
func (d *DockerService) Status() DockerStatus {
	container, err := d.DiscoverNodeRed()
	if err != nil {
		return DockerStatus{
			Available: false,
			Source:    "docker-cli",
			Message:   "No Node-RED container found",
		}
	}
	return DockerStatus{
		Available: true,
		InDocker:  false,
		Source:    "docker-cli",
		Container: container,
	}
}

// Restart runs `docker restart <name>` against the discovered
// Node-RED container. The container name is the operation target; if
// the container is missing the function returns a typed error so the
// HTTP layer can answer 503 with DOCKER_NOT_AVAILABLE.
func (d *DockerService) Restart() error {
	info, err := d.DiscoverNodeRed()
	if err != nil {
		return err
	}
	return d.runDocker("restart", info.Name)
}

// Stop runs `docker stop <name>` against the discovered Node-RED
// container.
func (d *DockerService) Stop() error {
	info, err := d.DiscoverNodeRed()
	if err != nil {
		return err
	}
	return d.runDocker("stop", info.Name)
}

// runDocker executes `docker <action> <name>` and returns nil on
// success. Any non-zero exit is wrapped so callers can audit-log the
// outcome.
func (d *DockerService) runDocker(action, name string) error {
	if d == nil || d.execCommand == nil {
		return errDockerNotFound
	}
	cmd := d.execCommand("docker", action, name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s %s failed: %v: %s", action, name, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// fillContainerState augments a ContainerInfo with state fields
// (Running, Paused, RestartCount) from `docker inspect`. The function
// is best-effort: a failed inspect is silently ignored because the
// `docker ps` line is already enough to render the UI.
func (d *DockerService) fillContainerState(info *ContainerInfo, id string) {
	inspectCmd := d.execCommand("docker", "inspect", id,
		"--format", "{{.State.Running}}\t{{.State.Paused}}\t{{.RestartCount}}")
	out, err := inspectCmd.Output()
	if err != nil {
		return
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return
	}
	parts := strings.Split(line, "\t")
	if len(parts) < 3 {
		return
	}
	info.State.Running = parts[0] == "true"
	info.State.Paused = parts[1] == "true"
	if n, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
		info.State.RestartCount = n
	}
}

// mapContainerStatus reduces a `docker ps` Status line (e.g.
// "Up 5 minutes", "Exited (0) 3 minutes ago") to the values the
// frontend's ContainerStatus enum accepts. The fallback is "exited"
// so the UI can render a "stopped" badge.
func mapContainerStatus(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(lower, "up"):
		return "running"
	case strings.HasPrefix(lower, "exited"):
		return "exited"
	case strings.HasPrefix(lower, "paused"):
		return "paused"
	case strings.HasPrefix(lower, "restarting"):
		return "restarting"
	case strings.HasPrefix(lower, "created"):
		return "created"
	case strings.HasPrefix(lower, "removing"):
		return "removing"
	case strings.HasPrefix(lower, "dead"):
		return "dead"
	default:
		return "exited"
	}
}
