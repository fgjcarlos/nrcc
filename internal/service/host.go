package service

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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

	if !s.IsolatedSettings {
		status.NodeRed = s.inspectNodeRed(status)
	}
	status.Settings = s.resolveSettings(status)
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
		dep.Version = cleanVersionOutput(string(out))
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
	}

	if status.Docker.Installed {
		if dockerEnv, ok := s.inspectDockerNodeRed(); ok {
			return dockerEnv
		}
	}

	return env
}

func (s *HostService) inspectDockerNodeRed() (model.NodeRedEnvironment, bool) {
	cmd := execCommand("docker", "ps", "-a", "--format", "{{.ID}}\t{{.Image}}\t{{.Names}}\t{{.Status}}")
	out, err := cmd.Output()
	if err != nil {
		return model.NodeRedEnvironment{}, false
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		image := strings.ToLower(parts[1])
		name := parts[2]
		if !strings.Contains(image, "node-red") && !strings.Contains(strings.ToLower(name), "node-red") {
			continue
		}

		env := model.NodeRedEnvironment{
			Detected:      true,
			Mode:          model.InstallationModeDocker,
			ManagedByNRCC: strings.HasPrefix(name, "nrcc-"),
			Running:       strings.Contains(strings.ToLower(parts[3]), "up"),
			ContainerID:   parts[0],
			ContainerName: name,
		}

		if version, userDir, settingsPath := s.inspectDockerContainer(parts[0]); version != "" || settingsPath != "" {
			env.Version = version
			env.UserDir = userDir
			env.SettingsPath = settingsPath
		}

		return env, true
	}
	return model.NodeRedEnvironment{}, false
}

func (s *HostService) inspectDockerContainer(containerID string) (version, userDir, settingsPath string) {
	cmd := execCommand(
		"docker", "inspect", containerID,
		"--format",
		"{{range .Mounts}}{{println .Source \"=>\" .Destination}}{{end}}",
	)
	out, err := cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
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
		version = cleanVersionOutput(string(versionOut))
	}
	return version, userDir, settingsPath
}

func (s *HostService) resolveSettings(status model.HostStatus) model.SettingsDocument {
	path := ""
	source := "nrcc-data"

	if !s.IsolatedSettings {
		path = status.NodeRed.SettingsPath
		source = "detected"

		if path == "" {
			if envPath := strings.TrimSpace(os.Getenv("NODE_RED_SETTINGS")); envPath != "" {
				path = envPath
				source = "env"
			}
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
	if !status.NodeJS.Installed {
		items = append(items, "Instala Node.js y npm para soportar una instalacion nativa de Node-RED.")
	}
	if !status.NodeRed.Detected {
		items = append(items, "Instala Node-RED en modo nativo o Docker antes de usar las funciones operativas.")
	}
	if status.NodeRed.Detected && !status.Portless.Installed {
		items = append(items, "Opcional: instala Portless para exponer nrcc o Node-RED con URLs HTTPS .localhost, LAN o Tailscale.")
	}
	if status.NodeRed.Mode == model.InstallationModeDocker && status.NodeRed.SettingsPath == "" {
		items = append(items, "Node-RED en Docker no expone /data como bind mount accesible; nrcc gestionara settings.js en su propio data dir. Usa -v /host/path:/data para editarlo directamente.")
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}

	// Test write permission by creating a temporary file
	testFile := filepath.Join(dir, ".nrcc-write-test-"+strconv.Itoa(os.Getpid()))
	file, err := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return false
	}
	_ = file.Close()
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

// isDevNull reports whether the given file info points at /dev/null.
// /dev/null is a character device, so a naive ModeCharDevice check would
// treat `docker run` (which redirects stdin from /dev/null) as interactive
// and hang the bootstrap wizard on `pterm.DefaultInteractiveConfirm`.
// ponytail: defensive guard against the #445 wizard regression. The wizard
// itself is gone (ADR 0003) but if a future PR ever resurrects a TUI setup
// step, drop this back into isInteractiveTerminal before merging.
func isDevNull(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	if info.Mode()&os.ModeDevice == 0 {
		return false
	}
	// Stat the device so we can compare against the kernel-reported dev number.
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return false
	}
	devNullStat, err := os.Stat(os.DevNull)
	if err != nil {
		return false
	}
	dst, ok := devNullStat.Sys().(*syscall.Stat_t)
	if !ok || dst == nil {
		return false
	}
	return stat.Dev == dst.Dev && stat.Ino == dst.Ino
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
