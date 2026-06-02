package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/ui"
	"github.com/pterm/pterm"
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

// HostService inspects the local environment and optionally performs guided installs.
type HostService struct {
	dataDir string
	// IsolatedSettings forces settings.js resolution to stay inside dataDir.
	// Tests use this to avoid reading or mutating a developer/server Node-RED install.
	IsolatedSettings bool
	// StartManaged is set to true by BootstrapCLI when the user opts to start
	// Node-RED after a native install. main.go checks this after BootstrapCLI returns.
	StartManaged bool
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
		Interactive:   isInteractiveTerminal(),
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

// PrintDoctorReport writes a readable host report for CLI usage.
func (s *HostService) PrintDoctorReport() model.HostStatus {
	status := s.Detect()

	if status.Portless.Installed {
		if aliases, err := s.ReadPortlessAliases(); err == nil {
			switch len(aliases) {
			case 0:
				status.Portless.Details = "no aliases; run nrcc portless quick-setup"
			case 1:
				status.Portless.Details = "1 alias registered"
			default:
				status.Portless.Details = fmt.Sprintf("%d aliases registered", len(aliases))
			}
		}
	}

	// Build doctor table rows
	rows := []ui.DoctorRow{
		{
			Name:      status.NodeJS.Name,
			Installed: status.NodeJS.Installed,
			Version:   status.NodeJS.Version,
			Command:   status.NodeJS.Command,
		},
		{
			Name:      status.NPM.Name,
			Installed: status.NPM.Installed,
			Version:   status.NPM.Version,
			Command:   status.NPM.Command,
		},
		{
			Name:      status.NodeRedBinary.Name,
			Installed: status.NodeRedBinary.Installed,
			Version:   status.NodeRedBinary.Version,
			Command:   status.NodeRedBinary.Command,
		},
		{
			Name:      status.Portless.Name,
			Installed: status.Portless.Installed,
			Version:   status.Portless.Version,
			Command:   status.Portless.Command,
			Details:   status.Portless.Details,
		},
		{
			Name:      status.Docker.Name,
			Installed: status.Docker.Installed,
			Version:   status.Docker.Version,
			Command:   status.Docker.Command,
		},
		{
			Name:      status.DockerCompose.Name,
			Installed: status.DockerCompose.Installed,
			Version:   status.DockerCompose.Version,
			Command:   status.DockerCompose.Command,
		},
	}

	ui.SectionHeader("Doctor")
	ui.Info(fmt.Sprintf("Platform: %s", status.Platform))
	ui.DoctorTable(rows)

	if status.NodeRed.SettingsPath != "" {
		ui.Info(fmt.Sprintf("settings.js: %s", status.NodeRed.SettingsPath))
	} else if status.Settings.Path != "" {
		ui.Info(fmt.Sprintf("settings.js: %s", status.Settings.Path))
	}

	ui.Info(fmt.Sprintf("Node-RED mode: %s", status.NodeRed.Mode))

	if len(status.Recommendations) > 0 {
		ui.Info("Recommendations:")
		for _, item := range status.Recommendations {
			ui.Info(fmt.Sprintf("  - %s", item))
		}
	}

	return status
}

// BootstrapCLI runs the interactive bootstrap flow before the server starts.
func (s *HostService) BootstrapCLI() error {
	status := s.PrintDoctorReport()
	if !status.Interactive {
		ui.Info("[nrcc] bootstrap: non-interactive terminal detected, skipping interactive setup")
		return nil
	}

	if runtime.GOOS == "linux" && !status.NodeJS.Installed {
		result, _ := pterm.DefaultInteractiveConfirm.Show("Node.js/npm no estan instalados. ¿Quieres instalarlos ahora?")
		if result {
			if err := s.installNodeJS(); err != nil {
				return err
			}
			status = s.PrintDoctorReport()
		}
	}

	if status.NodeRed.Detected {
		if !status.NodeRed.Running {
			ui.Info("Node-RED esta instalado pero no esta en ejecucion.")
			result, _ := pterm.DefaultInteractiveConfirm.Show("¿Quieres arrancarlo ahora?")
			if result {
				s.StartManaged = true
			}
		}
		return nil
	}

	options := availableNodeRedInstallOptions(status)

	ui.Info("Node-RED no esta instalado.")
	ui.Info(buildBootstrapOptionsDisplay(options))

	choice, _ := pterm.DefaultInteractiveSelect.WithOptions(options).Show("Selecciona una opcion")
	switch choice {
	case "native":
		if err := s.InstallNodeRedNative(); err != nil {
			return err
		}
		ui.Info("")
		result, _ := pterm.DefaultInteractiveConfirm.Show("¿Quieres arrancar Node-RED ahora?")
		if result {
			s.StartManaged = true
		}
	case "docker":
		if err := s.installNodeRedDocker(); err != nil {
			return err
		}
		// Docker install already starts the container — just confirm
		ui.Info("✓ Contenedor Node-RED arrancado en http://localhost:1880")
	}

	s.PrintDoctorReport()
	return nil
}

// ResolveSettingsPath returns the settings.js path that nrcc should edit.
func (s *HostService) ResolveSettingsPath() string {
	return s.Detect().Settings.Path
}

// RuntimeStatus returns the best-effort runtime information for an external Node-RED installation.
func (s *HostService) RuntimeStatus() model.RuntimeStatus {
	status := s.Detect()
	pid := 0
	if status.NodeRed.Mode == model.InstallationModeNative && status.NodeRed.Running {
		_, pid = processRunning("node-red")
	}
	return model.RuntimeStatus{
		Status:           runtimeState(status.NodeRed.Running, status.NodeRed.Detected),
		PID:              pid,
		Uptime:           0,
		Version:          status.NodeRed.Version,
		InstallationMode: status.NodeRed.Mode,
		ManagedByNRCC:    status.NodeRed.ManagedByNRCC,
		Detected:         status.NodeRed.Detected,
	}
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
	file.Close()
	os.Remove(testFile)
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

func isInteractiveTerminal() bool {
	if interactiveDisabledByEnv() || runningUnderSystemd() {
		return false
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func interactiveDisabledByEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NRCC_BOOTSTRAP_INTERACTIVE"))) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

func runningUnderSystemd() bool {
	return os.Getenv("INVOCATION_ID") != "" || os.Getenv("JOURNAL_STREAM") != ""
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

func runtimeState(running, detected bool) string {
	switch {
	case running:
		return "running"
	case detected:
		return "detected"
	default:
		return "stopped"
	}
}

func buildBootstrapOptionsDisplay(options []string) string {
	var optionTexts []string
	optionMap := map[string]string{
		"native": "native",
		"docker": "docker",
		"skip":   "skip",
	}
	for _, opt := range options {
		if text, ok := optionMap[opt]; ok {
			optionTexts = append(optionTexts, text)
		}
	}
	return "Opciones: " + strings.Join(optionTexts, ", ")
}

func (s *HostService) installNodeJS() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("automatic Node.js/npm installation is only implemented for linux")
	}
	pm := detectPackageManager()
	if pm == "" {
		return fmt.Errorf("no supported package manager found for automatic Node.js/npm installation (tried: apt-get, dnf, yum, pacman, zypper, apk)")
	}

	var updateCmd, installCmd []string
	switch pm {
	case "apt-get":
		updateCmd = []string{"apt-get", "update"}
		installCmd = []string{"apt-get", "install", "-y", "nodejs", "npm"}
	case "dnf":
		installCmd = []string{"dnf", "install", "-y", "nodejs", "npm"}
	case "yum":
		installCmd = []string{"yum", "install", "-y", "nodejs", "npm"}
	case "pacman":
		installCmd = []string{"pacman", "-Sy", "--noconfirm", "nodejs", "npm"}
	case "zypper":
		installCmd = []string{"zypper", "install", "-y", "nodejs", "npm"}
	case "apk":
		installCmd = []string{"apk", "add", "--no-cache", "nodejs", "npm"}
	default:
		return fmt.Errorf("unsupported package manager for Node.js/npm installation: %s", pm)
	}

	ui.Info("Installing Node.js and npm prerequisites...")
	if updateCmd != nil {
		if err := runElevatedCommands(updateCmd); err != nil {
			return err
		}
	}
	if err := runElevatedCommands(installCmd); err != nil {
		return err
	}

	// Verify installation
	status := s.Detect()
	if !status.NodeJS.Installed || !status.NPM.Installed {
		missing := []string{}
		if !status.NodeJS.Installed {
			missing = append(missing, "node")
		}
		if !status.NPM.Installed {
			missing = append(missing, "npm")
		}
		return fmt.Errorf("Node.js/npm installation appeared to succeed but verification failed; missing: %s", strings.Join(missing, ", "))
	}
	ui.Info(fmt.Sprintf("✓ Node.js %s and npm %s installed successfully", status.NodeJS.Version, status.NPM.Version))
	return nil
}

func (s *HostService) ensureNodeJSAndNPM() error {
	status := s.Detect()
	if status.NodeJS.Installed && status.NPM.Installed {
		return nil
	}
	return s.installNodeJS()
}

func (s *HostService) installDocker() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("automatic Docker installation is only implemented for linux")
	}
	pm := detectPackageManager()
	if pm == "" {
		return fmt.Errorf("no supported package manager found for automatic Docker installation (tried: apt-get, dnf, yum, pacman, zypper, apk)")
	}

	var updateCmd, installCmd []string
	switch pm {
	case "apt-get":
		updateCmd = []string{"apt-get", "update"}
		installCmd = []string{"apt-get", "install", "-y", "docker.io"}
	case "dnf":
		installCmd = []string{"dnf", "install", "-y", "docker"}
	case "yum":
		installCmd = []string{"yum", "install", "-y", "docker"}
	case "pacman":
		installCmd = []string{"pacman", "-Sy", "--noconfirm", "docker"}
	case "zypper":
		installCmd = []string{"zypper", "install", "-y", "docker"}
	case "apk":
		installCmd = []string{"apk", "add", "--no-cache", "docker"}
	default:
		return fmt.Errorf("unsupported package manager for Docker installation: %s", pm)
	}

	ui.Info("Installing Docker prerequisite...")
	if updateCmd != nil {
		if err := runElevatedCommands(updateCmd); err != nil {
			return err
		}
	}
	if err := runElevatedCommands(installCmd); err != nil {
		return err
	}

	status := s.Detect()
	if !status.Docker.Installed {
		return fmt.Errorf("Docker installation appeared to succeed but verification failed")
	}
	if err := s.ensureDockerDaemon(status.Docker.Command); err != nil {
		return err
	}
	ui.Info(fmt.Sprintf("✓ Docker %s installed and daemon reachable", status.Docker.Version))
	return nil
}

func (s *HostService) ensureDockerAvailable() error {
	status := s.Detect()
	if !status.Docker.Installed {
		if err := s.installDocker(); err != nil {
			return err
		}
		status = s.Detect()
	}
	if !status.Docker.Installed {
		return fmt.Errorf("Docker is required for Docker-based installation")
	}
	return s.ensureDockerDaemon(status.Docker.Command)
}

func (s *HostService) ensureDockerDaemon(dockerCmd string) error {
	if dockerCmd == "" {
		dockerCmd = "docker"
	}
	cmd := execCommand(dockerCmd, "info")
	if err := cmd.Run(); err != nil {
		if _, systemctlErr := execLookPath("systemctl"); systemctlErr == nil {
			_ = runElevatedCommands([]string{"systemctl", "enable", "--now", "docker"})
			cmd = execCommand(dockerCmd, "info")
			if retryErr := cmd.Run(); retryErr == nil {
				return nil
			}
		}
		return fmt.Errorf("Docker is installed but the daemon is not reachable; start Docker and retry: %w", err)
	}
	return nil
}

func detectPackageManager() string {
	for _, pm := range []string{"apt-get", "dnf", "yum", "pacman", "zypper", "apk"} {
		if _, err := execLookPath(pm); err == nil {
			return pm
		}
	}
	return ""
}

func (s *HostService) InstallNodeRedNative() error {
	if err := s.ensureNodeJSAndNPM(); err != nil {
		return fmt.Errorf("failed to prepare Node.js/npm for native Node-RED installation: %w", err)
	}

	npmPath, err := execLookPath("npm")
	if err != nil {
		return fmt.Errorf("npm is required to install Node-RED natively after dependency preparation: %w", err)
	}

	nodePath, _ := execLookPath("node")
	installCmd := buildNPMGlobalPackageCommand(npmPath, nodePath, "install", "node-red", true, npmGlobalPrefixWritable(npmPath))

	// Execute the install command
	if len(installCmd) > 0 {
		cmd := execCommand(installCmd[0], installCmd[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	// Verify installation
	status := s.Detect()
	if !status.NodeRedBinary.Installed {
		return fmt.Errorf("Node-RED installation appeared to succeed but verification failed")
	}
	ui.Info(fmt.Sprintf("✓ Node-RED %s instalado correctamente", status.NodeRedBinary.Version))
	return nil
}

// InstallPortless installs the Portless CLI globally via npm.
func (s *HostService) InstallPortless() error {
	if err := s.ensureNodeJSAndNPM(); err != nil {
		return fmt.Errorf("failed to prepare Node.js/npm for Portless installation: %w", err)
	}

	npmPath, err := execLookPath("npm")
	if err != nil {
		return fmt.Errorf("npm is required to install Portless after dependency preparation: %w", err)
	}

	nodePath, _ := execLookPath("node")
	installCmd := buildNPMGlobalPackageCommand(npmPath, nodePath, "install", "portless", false, npmGlobalPrefixWritable(npmPath))
	if len(installCmd) > 0 {
		cmd := execCommand(installCmd[0], installCmd[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	status := s.Detect()
	if !status.Portless.Installed {
		return fmt.Errorf("Portless installation appeared to succeed but verification failed")
	}
	ui.Info(fmt.Sprintf("✓ Portless %s instalado correctamente", status.Portless.Version))
	return nil
}

// ExposePortlessAlias registers a static Portless alias for an already-running local service.
func (s *HostService) ExposePortlessAlias(name string, port int, force bool) error {
	if err := validatePortlessAlias(name, port); err != nil {
		return err
	}
	status := s.Detect()
	if !status.Portless.Installed {
		return fmt.Errorf("Portless is not installed. Run: nrcc portless install")
	}

	args := []string{"alias", name, strconv.Itoa(port)}
	if force {
		args = append(args, "--force")
	}
	cmd := execCommand(status.Portless.Command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ReadPortlessAliases reads registered Portless aliases.
// It prefers the official Portless CLI output and falls back to known state files.
// Returns (nil, nil) if no source exists.
func (s *HostService) ReadPortlessAliases() ([]PortlessAlias, error) {
	if status := s.inspectCommand("portless", "--version"); status.Installed {
		out, err := execCommand(status.Command, "list").CombinedOutput()
		if err == nil {
			if aliases, ok := parsePortlessListOutput(string(out)); ok {
				return aliases, nil
			}
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve home directory: %w", err)
	}

	routesPath := filepath.Join(homeDir, ".portless", "routes.json")
	if aliases, err := readPortlessRoutesFile(routesPath); err != nil {
		return nil, err
	} else if aliases != nil {
		return aliases, nil
	}

	aliasesPath := filepath.Join(homeDir, ".portless", "aliases.json")
	data, err := os.ReadFile(aliasesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Legacy fallback: aliases.json as map[string]int: {"alias_name": port}.
	aliasesMap := make(map[string]int)
	if err := json.Unmarshal(data, &aliasesMap); err != nil {
		return nil, fmt.Errorf("failed to parse aliases.json: %w", err)
	}

	return aliasesFromMap(aliasesMap), nil
}

func readPortlessRoutesFile(path string) ([]PortlessAlias, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var routes []struct {
		Hostname string `json:"hostname"`
		Port     int    `json:"port"`
		PID      int    `json:"pid"`
	}
	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("failed to parse routes.json: %w", err)
	}

	aliasesMap := make(map[string]int)
	for _, route := range routes {
		if route.Hostname == "" || route.Port == 0 || route.PID != 0 {
			continue
		}
		aliasesMap[portlessAliasNameFromHostname(route.Hostname)] = route.Port
	}
	return aliasesFromMap(aliasesMap), nil
}

func parsePortlessListOutput(output string) ([]PortlessAlias, bool) {
	if !strings.Contains(output, "Active routes") && !strings.Contains(output, "No active routes") {
		return nil, false
	}

	aliasesMap := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, "->") || !strings.Contains(line, "(alias)") {
			continue
		}
		parts := strings.Split(line, "->")
		if len(parts) != 2 {
			continue
		}
		urlFields := strings.Fields(strings.TrimSpace(parts[0]))
		if len(urlFields) == 0 {
			continue
		}
		hostname := strings.TrimPrefix(urlFields[0], "https://")
		hostname = strings.TrimPrefix(hostname, "http://")
		targetFields := strings.Fields(strings.TrimSpace(parts[1]))
		if len(targetFields) == 0 {
			continue
		}
		target := targetFields[0]
		_, portText, err := net.SplitHostPort(target)
		if err != nil {
			lastColon := strings.LastIndex(target, ":")
			if lastColon == -1 || lastColon == len(target)-1 {
				continue
			}
			portText = target[lastColon+1:]
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port == 0 {
			continue
		}
		aliasesMap[portlessAliasNameFromHostname(hostname)] = port
	}

	return aliasesFromMap(aliasesMap), true
}

func aliasesFromMap(aliasesMap map[string]int) []PortlessAlias {
	var aliases []PortlessAlias
	var names []string
	for name := range aliasesMap {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		aliases = append(aliases, PortlessAlias{
			Name: name,
			Port: aliasesMap[name],
			URL:  fmt.Sprintf("https://%s.localhost", name),
		})
	}

	return aliases
}

func portlessAliasNameFromHostname(hostname string) string {
	hostname = strings.TrimSpace(hostname)
	hostname = strings.TrimSuffix(hostname, "/")
	return strings.TrimSuffix(hostname, ".localhost")
}

// CheckPortlessAliasReachability annotates aliases with the local upstream
// address Portless needs to reach. A closed port is the usual cause of 502.
func (s *HostService) CheckPortlessAliasReachability(aliases []PortlessAlias) []PortlessAlias {
	checked := make([]PortlessAlias, len(aliases))
	for i, alias := range aliases {
		alias.LocalAddress = fmt.Sprintf("127.0.0.1:%d", alias.Port)
		alias.Reachable = canReachTCP(alias.LocalAddress)
		checked[i] = alias
	}
	return checked
}

func canReachTCP(address string) bool {
	conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// QuickSetupPortless exposes nrcc (port 3001) and node-red (port 1880) aliases in one call.
// force=true overwrites any existing aliases.
func (s *HostService) QuickSetupPortless(force bool) error {
	status := s.Detect()
	if !status.Portless.Installed {
		return fmt.Errorf("Portless is not installed. Run: nrcc portless install")
	}

	aliases, err := s.ReadPortlessAliases()
	if err != nil {
		return fmt.Errorf("read Portless aliases: %w", err)
	}
	existing := make(map[string]bool, len(aliases))
	for _, alias := range aliases {
		existing[alias.Name] = true
	}

	targets := []PortlessAlias{
		{Name: "nrcc", Port: 3001},
		{Name: "node-red", Port: 1880},
	}
	var errs []string
	for _, target := range targets {
		if !force && existing[target.Name] {
			continue
		}
		if err := s.ExposePortlessAlias(target.Name, target.Port, force); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", target.Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("quick setup failed: %s. Use --force to overwrite existing aliases", strings.Join(errs, "; "))
	}

	if err := s.StartPortlessProxy(); err != nil {
		return fmt.Errorf("failed to start Portless proxy: %w", err)
	}

	return nil
}

// StartPortlessProxy starts the Portless reverse proxy daemon required for .localhost URLs.
func (s *HostService) StartPortlessProxy() error {
	status := s.Detect()
	if !status.Portless.Installed {
		return fmt.Errorf("Portless is not installed. Run: nrcc portless install")
	}

	cmd := execCommand(status.Portless.Command, "proxy", "start")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// SetupPortlessTrust runs the `portless trust` command with stdin/stdout passthrough.
func (s *HostService) SetupPortlessTrust() error {
	status := s.Detect()
	if !status.Portless.Installed {
		return fmt.Errorf("Portless is not installed. Run: nrcc portless install")
	}

	cmd := execCommand(status.Portless.Command, "trust")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// UninstallPortless removes the Portless npm package.
// If cleanAliases=true, also removes ~/.portless/ directory.
func (s *HostService) UninstallPortless(cleanAliases bool) error {
	npmPath, err := execLookPath("npm")
	if err != nil {
		return fmt.Errorf("npm is required to uninstall Portless")
	}

	// Check if Portless is installed
	status := s.Detect()
	if !status.Portless.Installed {
		return nil // Already uninstalled, return success
	}

	nodePath, _ := execLookPath("node")
	uninstallCmd := buildNPMGlobalPackageCommand(npmPath, nodePath, "uninstall", "portless", false, npmGlobalPrefixWritable(npmPath))

	// Execute the uninstall command
	if len(uninstallCmd) > 0 {
		cmd := execCommand(uninstallCmd[0], uninstallCmd[1:]...)
		cmd.Stdin = os.Stdin
		var output bytes.Buffer
		cmd.Stdout = &output
		cmd.Stderr = &output
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm uninstall failed: %w: %s. Try: sudo npm uninstall -g portless", err, strings.TrimSpace(output.String()))
		}
	}

	// Verify uninstallation
	newStatus := s.Detect()
	if newStatus.Portless.Installed {
		return fmt.Errorf("Portless uninstallation appeared to succeed but verification failed")
	}

	// Clean up aliases directory if requested
	if cleanAliases {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot resolve home directory: %w", err)
		}
		portlessDir := filepath.Join(homeDir, ".portless")
		if err := os.RemoveAll(portlessDir); err != nil {
			return fmt.Errorf("failed to remove %s: %w", portlessDir, err)
		}
	}

	ui.Info("✓ Portless uninstalado correctamente")
	return nil
}

// UninstallNodeRedNative uninstalls Node-RED from the system using npm.
// Follows the same pattern as installNodeRedNative with sudo detection.
func (s *HostService) UninstallNodeRedNative() error {
	npmPath, err := execLookPath("npm")
	if err != nil {
		return fmt.Errorf("npm is required to uninstall Node-RED natively")
	}

	nodePath, _ := execLookPath("node")

	var uninstallCmd []string

	// Check if npm global prefix is user-writable
	if npmGlobalPrefixWritable(npmPath) {
		// npm prefix is writable by current user, no sudo needed
		uninstallCmd = []string{npmPath, "uninstall", "-g", "node-red"}
	} else {
		// npm prefix is not writable, need sudo
		if nodePath != "" {
			// Use explicit node interpreter to avoid /usr/bin/env node shebang issue
			uninstallCmd = []string{"sudo", nodePath, npmPath, "uninstall", "-g", "node-red"}
		} else {
			// Last resort: pass PATH through sudo env
			uninstallCmd = []string{"sudo", "env", "PATH=" + os.Getenv("PATH"), npmPath, "uninstall", "-g", "node-red"}
		}
	}

	// Execute the uninstall command
	if len(uninstallCmd) > 0 {
		cmd := execCommand(uninstallCmd[0], uninstallCmd[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	// Verify uninstallation by checking if Node-RED is no longer detected
	status := s.Detect()
	if status.NodeRedBinary.Installed {
		return fmt.Errorf("Node-RED uninstallation appeared to succeed but verification failed — binary still detected")
	}
	ui.Info("✓ Node-RED uninstalado correctamente")
	return nil
}

// UpdateNodeRedNative updates Node-RED to the latest version.
// Returns the old version (before update) and new version (after update).
func (s *HostService) UpdateNodeRedNative() (oldVersion, newVersion string, err error) {
	npmPath, lookErr := execLookPath("npm")
	if lookErr != nil {
		return "", "", fmt.Errorf("npm is required to update Node-RED natively")
	}

	// Get current version BEFORE update
	oldStatus := s.Detect()
	oldVersion = oldStatus.NodeRedBinary.Version
	if !oldStatus.NodeRedBinary.Installed {
		return "", "", fmt.Errorf("Node-RED is not installed")
	}

	nodePath, _ := execLookPath("node")

	var updateCmd []string

	// Check if npm global prefix is user-writable
	if npmGlobalPrefixWritable(npmPath) {
		// npm prefix is writable by current user, no sudo needed
		updateCmd = []string{npmPath, "update", "-g", "node-red"}
	} else {
		// npm prefix is not writable, need sudo
		if nodePath != "" {
			// Use explicit node interpreter to avoid /usr/bin/env node shebang issue
			updateCmd = []string{"sudo", nodePath, npmPath, "update", "-g", "node-red"}
		} else {
			// Last resort: pass PATH through sudo env
			updateCmd = []string{"sudo", "env", "PATH=" + os.Getenv("PATH"), npmPath, "update", "-g", "node-red"}
		}
	}

	// Execute the update command
	if len(updateCmd) > 0 {
		cmd := execCommand(updateCmd[0], updateCmd[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return oldVersion, "", err
		}
	}

	// Get new version AFTER update
	newStatus := s.Detect()
	newVersion = newStatus.NodeRedBinary.Version

	if newVersion == oldVersion {
		ui.Info("ℹ Node-RED is already on the latest version")
	} else {
		ui.Info(fmt.Sprintf("✓ Node-RED actualizado: %s → %s", oldVersion, newVersion))
	}

	return oldVersion, newVersion, nil
}

// npmGlobalPrefixWritable checks if the npm global prefix is writable by the current user.
// It runs 'npm prefix -g', captures the prefix path, and tests if it's writable.
func npmGlobalPrefixWritable(npmPath string) bool {
	// Get the npm global prefix
	cmd := execCommand(npmPath, "prefix", "-g")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	prefixPath := strings.TrimSpace(string(out))
	if prefixPath == "" {
		return false
	}

	// Test write permission by attempting to create a temporary file
	testFile := filepath.Join(prefixPath, ".nrcc-write-test-"+strconv.Itoa(os.Getpid()))
	file, err := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}

func buildNPMGlobalPackageCommand(npmPath, nodePath, action, packageName string, unsafePerm, prefixWritable bool) []string {
	args := []string{action, "-g"}
	if unsafePerm {
		args = append(args, "--unsafe-perm")
	}
	args = append(args, packageName)

	if prefixWritable {
		return append([]string{npmPath}, args...)
	}
	if nodePath != "" {
		return append([]string{"sudo", nodePath, npmPath}, args...)
	}
	return append([]string{"sudo", "env", "PATH=" + os.Getenv("PATH"), npmPath}, args...)
}

func validatePortlessAlias(name string, port int) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("alias name is required")
	}
	if strings.ContainsAny(name, " \t\n/") {
		return fmt.Errorf("alias name must be a hostname label without spaces or slashes")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func (s *HostService) installNodeRedDocker() (err error) {
	if err := s.ensureDockerAvailable(); err != nil {
		return fmt.Errorf("failed to prepare Docker for Docker-based Node-RED installation: %w", err)
	}
	dockerPath, dockerErr := execLookPath("docker")
	if dockerErr != nil {
		return fmt.Errorf("docker is required for Docker-based installation after dependency preparation: %w", dockerErr)
	}
	_ = dockerPath
	targetDir := filepath.Join(s.dataDir, "nodered")

	// Track if we created the directory so we can clean up if needed
	dirCreated := false
	dirInfo, statErr := os.Stat(targetDir)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			dirCreated = true
		}
	} else if !dirInfo.IsDir() {
		return fmt.Errorf("target path exists but is not a directory: %s", targetDir)
	}

	if mkErr := os.MkdirAll(targetDir, 0755); mkErr != nil {
		return fmt.Errorf("create docker data dir: %w", mkErr)
	}

	// Defer cleanup of directory if it was newly created and later steps fail
	defer func() {
		if dirCreated && err != nil {
			ui.Info(fmt.Sprintf("[nrcc] cleaning up newly created directory: %s", targetDir))
			os.RemoveAll(targetDir)
		}
	}()

	// Check for existing container with the same name
	if containerExists("nrcc-node-red") {
		ui.Info("Container nrcc-node-red already exists. Removing...")
		if rmErr := runElevatedCommands([]string{"docker", "rm", "-f", "nrcc-node-red"}); rmErr != nil {
			err = fmt.Errorf("failed to remove existing container: %w", rmErr)
			return
		}
	}

	// Pull image first
	if pullErr := runElevatedCommands([]string{"docker", "pull", "nodered/node-red:latest"}); pullErr != nil {
		err = fmt.Errorf("failed to pull Docker image: %w", pullErr)
		return
	}

	// Run container
	if runErr := runElevatedCommands(
		[]string{
			"docker", "run", "-d",
			"--name", "nrcc-node-red",
			"-p", "1880:1880",
			"-v", targetDir + ":/data",
			"nodered/node-red:latest",
		},
	); runErr != nil {
		// Attempt cleanup of partially created container
		ui.Info("[nrcc] docker run failed, attempting cleanup of partial container")
		execCommand("docker", "rm", "-f", "nrcc-node-red").Run()
		err = fmt.Errorf("failed to run Docker container: %w", runErr)
		return
	}

	// Verify installation
	status := s.Detect()
	if !status.Docker.Installed || !status.NodeRed.Detected {
		err = fmt.Errorf("Docker Node-RED installation appeared to succeed but verification failed")
		return
	}
	ui.Info("✓ Node-RED Docker container instalado y corriendo correctamente")
	return nil
}

func containerExists(name string) bool {
	cmd := execCommand("docker", "ps", "-a", "--filter", "name="+name, "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == name
}

func runElevatedCommands(commands ...[]string) error {
	for _, args := range commands {
		if len(args) == 0 {
			continue
		}

		cmdName := args[0]
		cmdArgs := args[1:]

		// Use sudo if not already root
		if os.Geteuid() != 0 {
			if _, err := execLookPath("sudo"); err == nil {
				// Resolve the absolute path of the command before passing to sudo.
				// sudo uses a restricted PATH that may not include user-local installs
				// (e.g. nvm, ~/.local/bin). Using the full path avoids "command not found".
				if fullPath, err := execLookPath(args[0]); err == nil {
					cmdName = "sudo"
					cmdArgs = append([]string{fullPath}, args[1:]...)
				} else {
					cmdName = "sudo"
					cmdArgs = args // fallback: pass as-is
				}
			}
		}

		cmd := execCommand(cmdName, cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
