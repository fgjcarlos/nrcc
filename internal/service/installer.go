package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/pterm/pterm"
)

// systemdUnitTemplate is the content for /etc/systemd/system/nrcc.service
const systemdUnitTemplate = `[Unit]
Description=Node-RED Control Center
Documentation=https://github.com/composedof2/nrcc
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=nrcc
Group=nrcc
WorkingDirectory=/var/lib/nrcc
EnvironmentFile=/etc/nrcc/nrcc.env
ExecStart=/usr/local/bin/nrcc
Restart=on-failure
RestartSec=10s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nrcc

[Install]
WantedBy=multi-user.target
`

// envFileTemplate is the content for /etc/nrcc/nrcc.env
const envFileTemplate = `# nrcc configuration — managed by: sudo nrcc install
# Edit this file, then: sudo systemctl restart nrcc

PORT=3001
DATA_DIR=/var/lib/nrcc
JWT_SECRET=%s
NRCC_MANAGE_NODE_RED=true
NRCC_BOOTSTRAP_INTERACTIVE=false
NODE_RED_CMD=node-red
NODE_RED_USER_DIR=
NODE_RED_SETTINGS=
`

// InstallerService orchestrates system installation of nrcc
type InstallerService struct {
	systemd SystemdManager
	layout  model.InstallLayout
}

type nodeRedInstallDecision struct {
	Mode         model.NodeRedInstallMode
	Detected     bool
	Command      string
	UserDir      string
	SettingsPath string
}

type installRollback struct {
	committed bool
	steps     []func()
}

func (r *installRollback) Defer(fn func()) {
	if fn != nil {
		r.steps = append(r.steps, fn)
	}
}

func (r *installRollback) Commit() {
	r.committed = true
}

func (r *installRollback) Run() {
	if r.committed {
		return
	}
	for i := len(r.steps) - 1; i >= 0; i-- {
		r.steps[i]()
	}
}

var (
	lookupUser       = user.Lookup
	lookupGroup      = user.LookupGroup
	chown            = os.Chown
	executablePath   = os.Executable
	detectHostStatus = func(hostSvc *HostService) model.HostStatus {
		return hostSvc.Detect()
	}
	installNodeRedNative = func(hostSvc *HostService) error {
		return hostSvc.InstallNodeRedNative()
	}
	installNodeRedDocker = func(hostSvc *HostService) error {
		return hostSvc.installNodeRedDocker()
	}
	selectNodeRedInstallMode = func(options []string) (model.NodeRedInstallMode, error) {
		choice, err := pterm.DefaultInteractiveSelect.WithOptions(options).Show("Select how to prepare Node-RED before starting nrcc")
		return model.NodeRedInstallMode(choice), err
	}
)

// NewInstallerService creates a new installer service
func NewInstallerService(layout model.InstallLayout) *InstallerService {
	return &InstallerService{
		systemd: NewSystemdManager(),
		layout:  layout,
	}
}

// GenerateJWTSecret creates a cryptographically secure random secret
func GenerateJWTSecret() (string, error) {
	// Generate 32 bytes of random data
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	// Encode as hex (64 characters)
	return fmt.Sprintf("%x", secret), nil
}

// IsInstalled checks if nrcc is already installed
func (s *InstallerService) IsInstalled() bool {
	_, errUnit := os.Stat(s.layout.SystemdUnit)
	_, errBin := os.Stat(s.layout.BinaryPath)
	_, errConf := os.Stat(s.layout.ConfigDir)
	return errUnit == nil && errBin == nil && errConf == nil
}

// Install performs the full system installation from CLI/API options.
func (s *InstallerService) Install(ctx context.Context, opts model.InstallOpts) error {
	hostSvc := NewHostService(s.layout.DataDir)
	pterm.Info.Println("Checking host dependencies before install decisions...")
	status := detectHostStatus(hostSvc)
	plan := model.NewInstallPlanFromOpts(opts)

	resolvedPlan, err := s.resolveInstallPlan(hostSvc, plan, status)
	if err != nil {
		return err
	}
	return s.InstallPlan(ctx, resolvedPlan)
}

// InstallPlan consumes a normalized install plan. This is the shared execution
// path for flags and the guided wizard, so all install front-ends get the same
// rollback and commit semantics.
func (s *InstallerService) InstallPlan(ctx context.Context, plan model.InstallPlan) error {
	// Check if already installed (non-fatal, just print info)
	if s.IsInstalled() {
		pterm.Info.Println("nrcc is already installed. Re-running installation to ensure service is active...")
	}

	// Ensure systemd is available
	if !s.systemd.IsAvailable() {
		return fmt.Errorf("systemd not available on this system. Only systemd is supported.")
	}

	rollback := &installRollback{}
	defer rollback.Run()

	// Step 1: Create system user and group
	if err := s.ensureSystemUser(); err != nil {
		return fmt.Errorf("failed to create system user: %w", err)
	}

	// Step 2: Create directories
	if err := s.createDirectoriesWithRollback(rollback); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Step 3: Install the currently running binary to the system service path
	if err := s.ensureInstalledBinaryWithRollback(rollback); err != nil {
		return fmt.Errorf("failed to install nrcc binary: %w", err)
	}

	hostSvc := NewHostService(s.layout.DataDir)
	decision := nodeRedInstallDecision{
		Mode:         plan.NodeRedMode,
		Detected:     plan.NodeRedDetected,
		Command:      plan.NodeRedCommand,
		UserDir:      plan.NodeRedUserDir,
		SettingsPath: plan.NodeRedSettings,
	}
	decision, err := s.applyNodeRedInstallDecision(hostSvc, decision)
	if err != nil {
		return err
	}

	// Step 4: Generate and write env file
	if err := s.generateEnvFileWithRollback(decision, rollback); err != nil {
		return fmt.Errorf("failed to generate env file: %w", err)
	}

	// Commit point: the systemd unit makes the install visible to service managers.
	if err := s.writeSystemdUnit(); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	rollback.Commit()

	// Optional: install Portless before starting nrcc so post-install aliases can be created immediately.
	if plan.WithPortless {
		if err := s.installPortlessAddons(plan); err != nil {
			return err
		}
	}

	// Step 5: Daemon reload and enable/start service
	if err := s.systemd.DaemonReload(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	if err := s.systemd.EnableAndStart(s.layout.ServiceName); err != nil {
		return fmt.Errorf("failed to enable/start service: %w", err)
	}

	// Step 6: Poll for service to become active
	if err := s.pollServiceStatus(ctx); err != nil {
		pterm.Warning.Println("Service enabled but not yet active. Check: journalctl -u nrcc -n 50")
		// Non-fatal — installation is successful, runtime issue is advisory
	}

	return nil
}

func (s *InstallerService) installPortlessAddons(plan model.InstallPlan) error {
	hostSvc := NewHostService(s.layout.DataDir)
	if err := hostSvc.InstallPortless(); err != nil {
		return fmt.Errorf("failed to install Portless: %w", err)
	}
	if plan.PortlessQuickSetup {
		if err := hostSvc.QuickSetupPortless(false); err != nil {
			return fmt.Errorf("failed to configure Portless aliases: %w", err)
		}
		pterm.Info.Println("Portless aliases configured and proxy started: https://nrcc.localhost and https://node-red.localhost")
		if !plan.PortlessTrust {
			pterm.Info.Println("If you see HTTPS certificate warnings, run: sudo nrcc portless setup-trust")
		}
	}
	if plan.PortlessTrust {
		if err := hostSvc.SetupPortlessTrust(); err != nil {
			return fmt.Errorf("failed to configure Portless trust: %w", err)
		}
	}

	return nil
}

func (s *InstallerService) resolveInstallPlan(hostSvc *HostService, plan model.InstallPlan, status model.HostStatus) (model.InstallPlan, error) {
	decision, err := s.resolveNodeRedInstallDecision(hostSvc, plan, status)
	if err != nil {
		return model.InstallPlan{}, err
	}
	plan.NodeRedMode = decision.Mode
	plan.NodeRedDetected = decision.Detected
	plan.NodeRedCommand = decision.Command
	plan.NodeRedUserDir = decision.UserDir
	plan.NodeRedSettings = decision.SettingsPath
	return plan, nil
}

func (s *InstallerService) resolveNodeRedInstallDecision(hostSvc *HostService, plan model.InstallPlan, status model.HostStatus) (nodeRedInstallDecision, error) {
	if plan.NodeRedMode != "" {
		decision := nodeRedInstallDecision{Mode: plan.NodeRedMode}
		if plan.NodeRedMode == model.NodeRedInstallModeNative {
			decision.Command = nativeNodeRedCommand(status)
			decision.Detected = decision.Command != ""
			decision.UserDir = status.NodeRed.UserDir
			decision.SettingsPath = status.NodeRed.SettingsPath
		}
		return decision, nil
	}

	if status.NodeRed.Detected {
		command := ""
		if status.NodeRed.Mode == model.InstallationModeNative {
			command = nativeNodeRedCommand(status)
		}
		return nodeRedInstallDecision{
			Mode:         model.NodeRedInstallModeSkip,
			Detected:     true,
			Command:      command,
			UserDir:      status.NodeRed.UserDir,
			SettingsPath: status.NodeRed.SettingsPath,
		}, nil
	}

	if plan.SkipPrompt {
		return nodeRedInstallDecision{Mode: model.NodeRedInstallModeSkip}, nil
	}

	if !status.Interactive {
		pterm.Warning.Println("Node-RED is not installed and no interactive terminal is available; continuing with --node-red skip. nrcc service will start without managing Node-RED.")
		return nodeRedInstallDecision{Mode: model.NodeRedInstallModeSkip}, nil
	}

	options := allNodeRedInstallOptions()
	pterm.Info.Println("Node-RED is not installed.")
	pterm.Info.Println(buildBootstrapOptionsDisplay(options))
	choice, err := selectNodeRedInstallMode(options)
	if err != nil {
		return nodeRedInstallDecision{}, err
	}
	return nodeRedInstallDecision{Mode: choice}, nil
}

func (s *InstallerService) applyNodeRedInstallDecision(hostSvc *HostService, decision nodeRedInstallDecision) (nodeRedInstallDecision, error) {
	if decision.Detected {
		return decision, nil
	}

	switch decision.Mode {
	case "", model.NodeRedInstallModeSkip:
		return decision, nil
	case model.NodeRedInstallModeNative:
		if err := installNodeRedNative(hostSvc); err != nil {
			return decision, fmt.Errorf("failed to install Node-RED natively: %w", err)
		}
		decision.Command = nativeNodeRedCommand(detectHostStatus(hostSvc))
		return decision, nil
	case model.NodeRedInstallModeDocker:
		if err := installNodeRedDocker(hostSvc); err != nil {
			return decision, fmt.Errorf("failed to install Node-RED with Docker: %w", err)
		}
		return decision, nil
	default:
		return decision, fmt.Errorf("unsupported Node-RED install mode %q", decision.Mode)
	}
}

func nativeNodeRedCommand(status model.HostStatus) string {
	if status.NodeRed.Mode == model.InstallationModeNative && status.NodeRed.Executable != "" {
		return status.NodeRed.Executable
	}
	if status.NodeRedBinary.Installed && status.NodeRedBinary.Command != "" {
		return status.NodeRedBinary.Command
	}
	return ""
}

func allNodeRedInstallOptions() []string {
	return []string{
		string(model.NodeRedInstallModeSkip),
		string(model.NodeRedInstallModeNative),
		string(model.NodeRedInstallModeDocker),
	}
}

func availableNodeRedInstallOptions(status model.HostStatus) []string {
	options := []string{string(model.NodeRedInstallModeSkip)}
	if status.NPM.Installed {
		options = append(options, string(model.NodeRedInstallModeNative))
	}
	if status.Docker.Installed {
		options = append(options, string(model.NodeRedInstallModeDocker))
	}
	return options
}

// Uninstall removes the installation (optionally keeping data)
func (s *InstallerService) Uninstall(ctx context.Context, opts model.UninstallOpts) error {
	// Check if installed
	if !s.IsInstalled() {
		pterm.Info.Println("No nrcc installation found. Nothing to remove.")
		return nil
	}

	// Collect step errors instead of silently discarding them. Best-effort steps
	// (stopping an already-stopped service) only warn; removal steps that fail
	// are surfaced both as warnings and in the returned aggregate error so the
	// command never reports success when it actually removed nothing.
	var errs []error
	warn := func(step string, err error) {
		if err != nil {
			pterm.Warning.Printfln("uninstall: %s: %v", step, err)
		}
	}
	fail := func(step string, err error) {
		if err != nil {
			pterm.Warning.Printfln("uninstall: %s: %v", step, err)
			errs = append(errs, fmt.Errorf("%s: %w", step, err))
		}
	}

	// Step 1: Stop and disable service (best-effort).
	warn("stop service", s.systemd.Stop(s.layout.ServiceName))
	warn("disable service", s.systemd.Disable(s.layout.ServiceName))

	// Step 2: Remove unit file, then reload.
	fail("remove unit file", removeIfExists(s.layout.SystemdUnit))
	warn("daemon reload", s.systemd.DaemonReload())

	// Step 3: Remove binary.
	fail("remove binary", removeIfExists(s.layout.BinaryPath))

	// Step 4: Decide whether to keep the data directory.
	keepData := opts.KeepData
	if opts.Purge {
		keepData = false
	} else if !opts.KeepData && !opts.SkipPrompt {
		// Interactive prompt; on error default to keeping data.
		result, err := pterm.DefaultInteractiveConfirm.Show("Keep /var/lib/nrcc data?")
		keepData = err != nil || result
	}

	// Step 5: A full uninstall (no data kept) also removes the data directory and
	// the dedicated system user/group created at install time.
	if !keepData {
		fail("remove data directory", os.RemoveAll(s.layout.DataDir))
		fail("remove system user", s.removeSystemUser())
	}

	// Step 6: Always remove the config directory.
	fail("remove config directory", os.RemoveAll(s.layout.ConfigDir))

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// removeIfExists removes a path, treating "not found" as success.
func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// removeSystemUser deletes the dedicated nrcc system user and group if present.
func (s *InstallerService) removeSystemUser() error {
	var errs []error
	if _, err := lookupUser(s.layout.ServiceUser); err == nil {
		if err := execCommand("userdel", s.layout.ServiceUser).Run(); err != nil {
			errs = append(errs, fmt.Errorf("userdel: %w", err))
		}
	}
	if _, err := lookupGroup(s.layout.ServiceUser); err == nil {
		if err := execCommand("groupdel", s.layout.ServiceUser).Run(); err != nil {
			errs = append(errs, fmt.Errorf("groupdel: %w", err))
		}
	}
	return errors.Join(errs...)
}

// Status returns the current installation state
func (s *InstallerService) Status() model.InstallStatus {
	status := model.InstallStatus{}

	// Check service state
	if s.systemd.IsAvailable() {
		svcState, err := s.systemd.GetServiceStatus(s.layout.ServiceName)
		if err != nil {
			status.ServiceState = "unknown"
		} else {
			status.ServiceState = svcState
		}
	} else {
		status.ServiceState = "unknown"
	}

	// Check directories and files
	_, err := os.Stat(s.layout.DataDir)
	status.DataDirExists = err == nil

	_, err = os.Stat(s.layout.EnvFile)
	status.EnvFileExists = err == nil

	_, err = os.Stat(s.layout.SystemdUnit)
	status.UnitFileExists = err == nil

	return status
}

// --- Private helper methods ---

// ensureSystemUser creates the nrcc system user and group if they don't exist
func (s *InstallerService) ensureSystemUser() error {
	if _, err := lookupGroup(s.layout.ServiceUser); err != nil {
		groupCmd := execCommand("groupadd", "-f", s.layout.ServiceUser)
		if err := groupCmd.Run(); err != nil {
			return err
		}
		if _, err := lookupGroup(s.layout.ServiceUser); err != nil {
			return fmt.Errorf("group %s not found after creation: %w", s.layout.ServiceUser, err)
		}
	}

	if _, err := lookupUser(s.layout.ServiceUser); err == nil {
		return nil
	}

	// Create system user (no home, no shell, system account)
	userCmd := execCommand("useradd",
		"-r",                      // system account
		"-s", "/usr/sbin/nologin", // no login shell
		"-d", "/nonexistent", // no home directory
		"-g", s.layout.ServiceUser,
		s.layout.ServiceUser,
	)
	var useraddErr error
	if err := userCmd.Run(); err != nil {
		// Check if already exists (exit code 9 on some systems, exit code 2 on others)
		if exitErr, ok := err.(*exec.ExitError); ok && (exitErr.ExitCode() == 9 || exitErr.ExitCode() == 2) {
			useraddErr = err
		} else {
			return err
		}
	}

	if _, err := lookupUser(s.layout.ServiceUser); err != nil {
		if useraddErr != nil {
			return fmt.Errorf("user %s not found after useradd: %w", s.layout.ServiceUser, useraddErr)
		}
		return fmt.Errorf("user %s not found after creation: %w", s.layout.ServiceUser, err)
	}

	return nil
}

func (s *InstallerService) ensureInstalledBinary() error {
	source, err := executablePath()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	if source == "" {
		return fmt.Errorf("resolve current executable: empty path")
	}

	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolve current executable path: %w", err)
	}
	destinationAbs, err := filepath.Abs(s.layout.BinaryPath)
	if err != nil {
		return fmt.Errorf("resolve install destination path: %w", err)
	}

	if sameFile(sourceAbs, destinationAbs) {
		pterm.Info.Printf("nrcc binary already installed at %s\n", destinationAbs)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(destinationAbs), 0755); err != nil {
		return fmt.Errorf("create binary install directory: %w", err)
	}
	if err := copyExecutable(sourceAbs, destinationAbs); err != nil {
		return err
	}
	pterm.Success.Printf("nrcc binary installed to %s\n", destinationAbs)
	return nil
}

func (s *InstallerService) ensureInstalledBinaryWithRollback(rollback *installRollback) error {
	destination := s.layout.BinaryPath
	before, statErr := os.Stat(destination)
	existed := statErr == nil
	if err := s.ensureInstalledBinary(); err != nil {
		return err
	}
	if !existed {
		rollback.Defer(func() { _ = os.Remove(destination) })
		return nil
	}
	// If the file existed and we overwrote it, do not try to reconstruct it from
	// stale metadata. Existing installs are past the pre-commit cleanup boundary.
	_ = before
	return nil
}

func sameFile(source string, destination string) bool {
	sourceInfo, sourceErr := os.Stat(source)
	destinationInfo, destinationErr := os.Stat(destination)
	if sourceErr == nil && destinationErr == nil {
		return os.SameFile(sourceInfo, destinationInfo)
	}
	return filepath.Clean(source) == filepath.Clean(destination)
}

func copyExecutable(source string, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open current executable: %w", err)
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(destination), ".nrcc-install-*")
	if err != nil {
		return fmt.Errorf("create temporary binary: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("copy current executable: %w", err)
	}
	if err := tmp.Chmod(0755); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set executable permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("flush installed binary: %w", err)
	}
	if err := os.Rename(tmpPath, destination); err != nil {
		return fmt.Errorf("move installed binary into place: %w", err)
	}
	return nil
}

// createDirectories creates the required installation directories
func (s *InstallerService) createDirectories() error {
	return s.createDirectoriesWithRollback(nil)
}

func (s *InstallerService) createDirectoriesWithRollback(rollback *installRollback) error {
	configExisted := pathExists(s.layout.ConfigDir)
	dataExisted := pathExists(s.layout.DataDir)

	// Create /etc/nrcc
	if err := os.MkdirAll(s.layout.ConfigDir, 0755); err != nil {
		return err
	}
	if rollback != nil && !configExisted {
		rollback.Defer(func() { _ = os.RemoveAll(s.layout.ConfigDir) })
	}

	// Create /var/lib/nrcc with restricted permissions
	if err := os.MkdirAll(s.layout.DataDir, 0750); err != nil {
		return err
	}
	if rollback != nil && !dataExisted {
		rollback.Defer(func() { _ = os.RemoveAll(s.layout.DataDir) })
	}

	// Change ownership to nrcc:nrcc
	svcUser, err := lookupUser(s.layout.ServiceUser)
	if err != nil {
		return fmt.Errorf("user %s not found: %w", s.layout.ServiceUser, err)
	}

	g, err := lookupGroup(s.layout.ServiceUser)
	if err != nil {
		return fmt.Errorf("group %s not found: %w", s.layout.ServiceUser, err)
	}

	uid, _ := strconv.Atoi(svcUser.Uid)
	gid, _ := strconv.Atoi(g.Gid)

	if err := chown(s.layout.DataDir, uid, gid); err != nil {
		return fmt.Errorf("failed to chown %s: %w", s.layout.DataDir, err)
	}

	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// generateEnvFile creates or updates /etc/nrcc/nrcc.env
func (s *InstallerService) generateEnvFile(decision nodeRedInstallDecision) error {
	return s.generateEnvFileWithRollback(decision, nil)
}

func (s *InstallerService) generateEnvFileWithRollback(decision nodeRedInstallDecision, rollback *installRollback) error {
	manageNodeRed := "true"
	if !decision.Detected && decision.Mode == model.NodeRedInstallModeSkip {
		manageNodeRed = "false"
	}

	before, readErr := os.ReadFile(s.layout.EnvFile)
	existed := readErr == nil
	content, err := s.loadOrCreateEnvFile()
	if err != nil {
		return err
	}

	content = setEnvValue(content, "NRCC_MANAGE_NODE_RED", manageNodeRed)
	content = setEnvValue(content, "NRCC_BOOTSTRAP_INTERACTIVE", "false")
	if decision.Command != "" {
		content = setEnvValue(content, "NODE_RED_CMD", decision.Command)
	}
	if decision.UserDir != "" {
		content = setEnvValue(content, "NODE_RED_USER_DIR", decision.UserDir)
	}
	if decision.SettingsPath != "" {
		content = setEnvValue(content, "NODE_RED_SETTINGS", decision.SettingsPath)
	}
	if npmPath, err := execLookPath("npm"); err == nil {
		content = setEnvValue(content, "NPM_BIN", npmPath)
	} else {
		pterm.Warning.Printfln("npm not found at install time; NPM_BIN not frozen — runtime npm operations may fail under systemd (%v)", err)
	}

	if err := s.writeEnvFile(content); err != nil {
		return err
	}
	if rollback != nil {
		if existed {
			backup := append([]byte(nil), before...)
			rollback.Defer(func() { _ = os.WriteFile(s.layout.EnvFile, backup, 0640) })
		} else {
			rollback.Defer(func() { _ = os.Remove(s.layout.EnvFile) })
		}
	}

	return nil
}

func (s *InstallerService) loadOrCreateEnvFile() (string, error) {
	data, err := os.ReadFile(s.layout.EnvFile)
	if err == nil {
		pterm.Info.Println("Existing nrcc.env updated.")
		return string(data), nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	secret, err := GenerateJWTSecret()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(envFileTemplate, secret), nil
}

func (s *InstallerService) writeEnvFile(content string) error {
	// Write file with restricted permissions (0640)
	if err := os.WriteFile(s.layout.EnvFile, []byte(content), 0640); err != nil {
		return err
	}

	// Change ownership to root:nrcc so nrcc service user can read
	if _, err := lookupUser(s.layout.ServiceUser); err != nil {
		return err
	}
	g, err := lookupGroup(s.layout.ServiceUser)
	if err != nil {
		return err
	}

	gid, _ := strconv.Atoi(g.Gid)
	if err := chown(s.layout.EnvFile, 0, gid); err != nil {
		return err
	}

	return nil
}

func setEnvValue(content, key, value string) string {
	needle := key + "="
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, needle) {
			lines[i] = needle + value
			return strings.Join(lines, "\n")
		}
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + needle + value + "\n"
}

// writeSystemdUnit writes the systemd unit file (always overwrite)
func (s *InstallerService) writeSystemdUnit() error {
	if err := os.WriteFile(s.layout.SystemdUnit, []byte(systemdUnitTemplate), 0644); err != nil {
		return err
	}
	pterm.Success.Println("Service unit updated.")
	return nil
}

// pollServiceStatus waits up to 10 seconds for service to become active
func (s *InstallerService) pollServiceStatus(ctx context.Context) error {
	deadline := time.Now().Add(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := s.systemd.GetServiceStatus(s.layout.ServiceName)
			if err == nil && status == "active" {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("service did not become active within 10 seconds (status: %s)", status)
			}
		}
	}
}
