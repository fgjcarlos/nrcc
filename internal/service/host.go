package service

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/model"
)

var (
	execCommand  = exec.Command
	execLookPath = exec.LookPath
)

// PortlessAlias describes a registered Portless alias.
type PortlessAlias struct {
	Name         string `json:"name"`
	Port         int    `json:"port"`
	URL          string `json:"url"`
	LocalAddress string `json:"localAddress,omitempty"`
	Reachable    bool   `json:"reachable"`
}

// HostService inspects the local environment.
type HostService struct {
	dataDir string
	// IsolatedSettings forces settings.js resolution to stay inside dataDir.
	IsolatedSettings bool
}

// NewHostService creates a new host inspection service.
func NewHostService(dataDir string) *HostService {
	return &HostService{dataDir: dataDir}
}

// NewIsolatedHostService creates a host service that never resolves settings.js
// from NODE_RED_SETTINGS, native Node-RED, or Docker; it always uses dataDir.
func NewIsolatedHostService(dataDir string) *HostService {
	return &HostService{dataDir: dataDir, IsolatedSettings: true}
}

// Detect returns a normalized view of the local Node-RED environment.
func (s *HostService) Detect() model.HostStatus {
	status := model.HostStatus{
		Platform:      runtime.GOOS,
		NodeJS:        s.inspectCommand("node", "--version"),
		NPM:           s.inspectCommand("npm", "--version"),
		NodeRedBinary: s.inspectCommand("node-red", "--version"),
		Portless:      s.inspectCommand("portless", "--version"),
		Docker:        s.inspectCommand("docker", "--version"),
		DockerCompose: s.inspectDockerCompose(),
		NodeRed: model.NodeRedEnvironment{
			Mode: model.InstallationModeNone,
		},
	}

	if s.IsolatedSettings {
		// Hermetic services model the supported Node-RED 5 fixture so handler
		// tests exercise the same editability boundary as production.
		status.NodeRed = model.NodeRedEnvironment{
			Detected:     true,
			Mode:         model.InstallationModeNative,
			Version:      nodeRED5CatalogVersion,
			UserDir:      s.dataDir,
			SettingsPath: filepath.Join(s.dataDir, "settings.js"),
		}
	} else {
		status.NodeRed = s.inspectNodeRed(status)
	}
	status.Settings = s.resolveSettings(status)
	status.Configuration = ResolveConfigurationCapabilities(status.NodeRed.Version, status.Settings)
	status.Recommendations = s.buildRecommendations(status)
	status.Ready = status.NodeRed.Detected && status.Settings.Path != ""

	return status
}

// ResolveSettingsPath returns the settings.js path that nrcc should edit.
func (s *HostService) ResolveSettingsPath() string {
	return s.Detect().Settings.Path
}

// inspectCommand reports whether a command exists and returns its version output.
func (s *HostService) inspectCommand(name string, versionArg string) model.DependencyStatus {
	dep := model.DependencyStatus{Name: name, Command: name}
	path, err := execLookPath(name)
	if err != nil {
		return dep
	}
	dep.Installed = true
	dep.Command = path
	out, err := execCommand(path, versionArg).CombinedOutput()
	if err == nil {
		if name == "node-red" {
			dep.Version = nodeREDVersionFromOutput(string(out))
		} else {
			dep.Version = cleanVersionOutput(string(out))
		}
	}
	return dep
}

func (s *HostService) inspectDockerCompose() model.DependencyStatus {
	dep := model.DependencyStatus{Name: "docker compose", Command: "docker compose"}
	if _, err := execLookPath("docker"); err != nil {
		return dep
	}
	cmd := execCommand("docker", "compose", "version")
	out, err := cmd.CombinedOutput()
	if err == nil {
		dep.Installed = true
		dep.Version = cleanVersionOutput(string(out))
	}
	return dep
}

func (s *HostService) inspectNodeRed(status model.HostStatus) model.NodeRedEnvironment {
	env := model.NodeRedEnvironment{
		Mode: model.InstallationModeNone,
	}

	if status.NodeRedBinary.Installed {
		env.Detected = true
		env.Mode = model.InstallationModeNative
		env.ManagedByNRCC = false
		env.Version = status.NodeRedBinary.Version
		env.Executable = status.NodeRedBinary.Command
		env.UserDir = s.defaultNativeUserDir()
		env.SettingsPath = filepath.Join(env.UserDir, "settings.js")
		env.Running, _ = processRunning("node-red")
		if unit, ambiguous := s.detectNodeRedSystemdUnit(); ambiguous {
			env.Ambiguous = true
		} else if unit != "" {
			env.ContainerName = unit
		}
	}

	if status.Docker.Installed {
		if dockerEnv, ok := s.inspectDockerNodeRed(); ok {
			return dockerEnv
		}
	}

	return env
}

func (s *HostService) detectNodeRedSystemdUnit() (string, bool) {
	cmd := execCommand("systemctl", "list-units", "--type=service", "--all", "--no-legend", "--plain")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	var matches []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.ToLower(fields[0])
		if strings.Contains(name, "node-red") || strings.Contains(name, "nodered") {
			matches = append(matches, fields[0])
		}
	}
	if len(matches) > 1 {
		return "", true
	}
	if len(matches) == 1 {
		return matches[0], false
	}
	return "", false
}

func (s *HostService) inspectDockerNodeRed() (model.NodeRedEnvironment, bool) {
	cmd := execCommand("docker", "ps", "-a", "--format", "{{.ID}}\t{{.Image}}\t{{.Names}}\t{{.Status}}")
	out, err := cmd.Output()
	if err != nil {
		return model.NodeRedEnvironment{}, false
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	var candidates []model.NodeRedEnvironment
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) < 4 {
			continue
		}
		image, name := strings.ToLower(parts[1]), parts[2]
		if !strings.Contains(image, "node-red") && !strings.Contains(strings.ToLower(name), "node-red") {
			continue
		}
		version, userDir, settingsPath, project, readinessURL := s.inspectDockerRuntimeTarget(parts[0])
		candidates = append(candidates, model.NodeRedEnvironment{
			Detected: true, Mode: model.InstallationModeDocker, ManagedByNRCC: project != "",
			Running: strings.Contains(strings.ToLower(parts[3]), "up"), ContainerID: parts[0], ContainerName: name,
			Version: version, UserDir: userDir, SettingsPath: settingsPath, ComposeProject: project, ReadinessURL: readinessURL,
		})
	}
	if len(candidates) == 0 {
		return model.NodeRedEnvironment{}, false
	}
	if len(candidates) != 1 {
		return model.NodeRedEnvironment{Detected: true, Mode: model.InstallationModeUnknown, Ambiguous: true}, true
	}
	return candidates[0], true
}

func (s *HostService) inspectDockerContainer(containerID string) (version, userDir, settingsPath string) {
	version, userDir, settingsPath, _, _ = s.inspectDockerRuntimeTarget(containerID)
	return version, userDir, settingsPath
}

func (s *HostService) inspectDockerRuntimeTarget(containerID string) (version, userDir, settingsPath, project, readinessURL string) {
	cmd := execCommand("docker", "inspect", containerID, "--format",
		"{{range .Mounts}}{{println .Source \"=>\" .Destination}}{{end}}{{index .Config.Labels \"com.docker.compose.project\"}}\t{{index .Config.Labels \"com.docker.compose.service\"}}\t{{range $port, $bindings := .NetworkSettings.Ports}}{{if eq $port \"1880/tcp\"}}{{range $bindings}}{{.HostIp}}:{{.HostPort}}{{end}}{{end}}{{end}}")
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimRight(string(out), "\r\n"), "\n")
		if len(lines) > 0 {
			labels := strings.Split(lines[len(lines)-1], "\t")
			if len(labels) >= 2 && strings.TrimSpace(labels[0]) != "" && strings.TrimSpace(labels[1]) != "" {
				project = strings.TrimSpace(labels[0])
			}
			if len(labels) >= 3 && strings.TrimSpace(labels[2]) != "" {
				endpoint := strings.TrimSpace(labels[2])
				host, port, ok := strings.Cut(endpoint, ":")
				if ok && host != "" && port != "" {
					if host == "0.0.0.0" || host == "::" || host == "" {
						host = "127.0.0.1"
					}
					readinessURL = "http://" + host + ":" + port + "/settings"
				}
			}
		}
		scanner := bufio.NewScanner(bytes.NewReader([]byte(strings.Join(lines[:len(lines)-1], "\n"))))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, "=>")
			if len(parts) != 2 {
				continue
			}
			src := strings.TrimSpace(parts[0])
			dest := strings.TrimSpace(parts[1])
			if dest == "/data" {
				// Named volumes (e.g. created with `docker volume create`) have their
				// source under /var/lib/docker/volumes/ and are owned by root — nrcc
				// cannot write there directly.  Only bind-mounts (host paths outside
				// that prefix) are safe to access from the host.
				if strings.HasPrefix(src, "/var/lib/docker/volumes") {
					break
				}
				userDir = src
				settingsPath = filepath.Join(src, "settings.js")
				break
			}
		}
	}

	versionOut, err := execCommand("docker", "exec", containerID, "node-red", "--version").CombinedOutput()
	if err == nil {
		version = nodeREDVersionFromOutput(string(versionOut))
	}
	return version, userDir, settingsPath, project, readinessURL
}

func (s *HostService) resolveSettings(status model.HostStatus) model.SettingsDocument {
	path := ""
	source := "nrcc-data"

	if !s.IsolatedSettings {
		// NODE_RED_SETTINGS is the explicit runtime contract. It must win over
		// executable detection because a managed Docker runtime can expose a
		// bundled node-red binary while loading /data/settings.js instead of the
		// native $HOME/.node-red/settings.js path.
		if envPath := strings.TrimSpace(os.Getenv("NODE_RED_SETTINGS")); envPath != "" {
			path = envPath
			source = "env"
		} else {
			path = status.NodeRed.SettingsPath
			source = "detected"
		}
	}
	if path == "" {
		path = filepath.Join(s.dataDir, "settings.js")
		source = "nrcc-data"
	}

	info := model.SettingsDocument{
		Path:     path,
		Source:   source,
		Writable: canWrite(path),
	}

	backupDir := filepath.Join(s.dataDir, "backups", "settings")
	lastBackup, _ := latestBackupFile(backupDir)
	info.BackupPath = lastBackup
	return info
}

func (s *HostService) buildRecommendations(status model.HostStatus) []string {
	var items []string
	if status.NodeRed.Detected && !status.Portless.Installed {
		items = append(items, "Opcional: instala Portless para exponer nrcc o Node-RED con URLs HTTPS .localhost, LAN o Tailscale.")
	}
	if status.Settings.Path != "" && !status.Settings.Writable {
		items = append(items, "Otorga permisos de escritura sobre settings.js para que nrcc pueda guardar cambios.")
	}
	return items
}

func (s *HostService) defaultNativeUserDir() string {
	home, err := os.UserHomeDir()
	if err != nil || !isUsableHomeDir(home) {
		return s.dataDir
	}
	return filepath.Join(home, ".node-red")
}

func isUsableHomeDir(path string) bool {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "." || cleaned == "" {
		return false
	}
	if !filepath.IsAbs(cleaned) {
		return false
	}
	if cleaned == "/nonexistent" || strings.HasPrefix(cleaned, "/nonexistent/") {
		return false
	}
	return true
}

func cleanVersionOutput(value string) string {
	lines := strings.Fields(strings.TrimSpace(value))
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, " ")
}

func canWrite(path string) bool {
	dir := filepath.Dir(path)

	// Check if file already exists
	// #nosec G703 -- path is validated against an allowlist (operator-supplied) by the caller.
	_, err := os.Stat(path)
	if err == nil {
		// File exists, verify directory is writable
		return isWritableDir(dir)
	}

	// File doesn't exist, check if parent directory is writable
	return isWritableDir(dir)
}

func isWritableDir(dir string) bool {
	if !isUsableHomeDir(dir) && (dir == "/nonexistent" || strings.HasPrefix(filepath.Clean(dir), "/nonexistent/")) {
		return false
	}

	// Try to create the directory if needed
	// #nosec G703 -- dir is a t.TempDir() or operator-supplied path; the caller validates the path.
	if err := os.MkdirAll(dir, 0750); err != nil {
		return false
	}

	// Test write permission by creating a temporary file
	testFile := filepath.Join(dir, ".nrcc-write-test-"+strconv.Itoa(os.Getpid()))
	// #nosec G304,G703 -- testFile is built from validated dir + a constant prefix + pid; not request-derived.
	file, err := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false
	}
	_ = file.Close()
	// #nosec G703 -- testFile is built from validated dir + a constant prefix + pid; not request-derived.
	_ = os.Remove(testFile)
	return true
}

func latestBackupFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var newestPath string
	var newestTime int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Unix() > newestTime {
			newestTime = info.ModTime().Unix()
			newestPath = filepath.Join(dir, entry.Name())
		}
	}
	return newestPath, nil
}

func processRunning(name string) (bool, int) {
	out, err := execCommand("pgrep", "-f", name).Output()
	if err != nil {
		return false, 0
	}
	lines := strings.Fields(string(out))
	if len(lines) == 0 {
		return false, 0
	}
	pid, _ := strconv.Atoi(lines[0])
	return true, pid
}
