package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

type EnvironmentService struct {
	runner platform.Runner
}

func NewEnvironmentService() EnvironmentService {
	return EnvironmentService{
		runner: platform.NewRunner(),
	}
}

func (s EnvironmentService) DefaultDataDir() (string, error) {
	if value := strings.TrimSpace(os.Getenv("NRCC_DATA_DIR")); value != "" {
		return value, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".nrcc", "data"), nil
}

func (s EnvironmentService) Diagnose(dataDir string) model.EnvironmentReport {
	report := model.EnvironmentReport{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		DataDir: dataDir,
		Checks:  make([]model.EnvironmentCheck, 0, 8),
	}

	report.Checks = append(report.Checks, s.checkDataDir(dataDir))

	nodePath, nodeErr := s.runner.LookPath("node")
	if nodeErr == nil {
		report.NodeInstalled = true
		version, _ := s.runner.Run("", "node", "--version")
		report.Checks = append(report.Checks, model.EnvironmentCheck{
			Name:    "node",
			Status:  model.StatusOK,
			Detail:  fmt.Sprintf("found at %s (%s)", nodePath, strings.TrimSpace(version)),
			Command: "node --version",
		})
	} else {
		report.Checks = append(report.Checks, model.EnvironmentCheck{
			Name:    "node",
			Status:  model.StatusFail,
			Detail:  "Node.js is not installed or not available in PATH",
			Command: "node --version",
		})
	}

	npmPath, npmErr := s.runner.LookPath("npm")
	if npmErr == nil {
		report.NPMInstalled = true
		version, _ := s.runner.Run("", "npm", "--version")
		report.Checks = append(report.Checks, model.EnvironmentCheck{
			Name:    "npm",
			Status:  model.StatusOK,
			Detail:  fmt.Sprintf("found at %s (%s)", npmPath, strings.TrimSpace(version)),
			Command: "npm --version",
		})
	} else {
		report.Checks = append(report.Checks, model.EnvironmentCheck{
			Name:    "npm",
			Status:  model.StatusFail,
			Detail:  "npm is not installed or not available in PATH",
			Command: "npm --version",
		})
	}

	portlessPath, portlessErr := s.runner.LookPath("portless")
	if portlessErr == nil {
		report.PortlessPresent = true
		version, _ := s.runner.Run("", "portless", "--version")
		detail := fmt.Sprintf("found at %s", portlessPath)
		if version != "" {
			detail = fmt.Sprintf("%s (%s)", detail, strings.TrimSpace(version))
		}
		report.Checks = append(report.Checks, model.EnvironmentCheck{
			Name:    "portless",
			Status:  model.StatusOK,
			Detail:  detail,
			Command: "portless --version",
		})
	} else {
		report.Checks = append(report.Checks, model.EnvironmentCheck{
			Name:    "portless",
			Status:  model.StatusWarn,
			Detail:  "portless is not available; the app can still run with a normal localhost URL",
			Command: "portless --version",
		})
	}

	nodeRedPath := filepath.Join(dataDir, "node_modules", "node-red", "package.json")
	if platform.Exists(nodeRedPath) {
		report.NodeRedReady = true
		report.Checks = append(report.Checks, model.EnvironmentCheck{
			Name:   "node-red",
			Status: model.StatusOK,
			Detail: fmt.Sprintf("local installation found at %s", nodeRedPath),
		})
	} else {
		report.Checks = append(report.Checks, model.EnvironmentCheck{
			Name:   "node-red",
			Status: model.StatusWarn,
			Detail: "local Node-RED installation not found in the data directory",
		})
	}

	return report
}

func (s EnvironmentService) Setup(dataDir string, stdin *os.File, stdout *os.File) error {
	if err := platform.EnsureDir(dataDir); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	if err := s.initializeDataDir(dataDir); err != nil {
		return err
	}

	report := s.Diagnose(dataDir)
	printReport(stdout, report)

	if !report.NodeInstalled || !report.NPMInstalled {
		return fmt.Errorf("missing system prerequisites: install Node.js and npm, then run `nrcc setup` again")
	}

	if report.NodeRedReady {
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Node-RED is already installed in the local data directory.")
		return nil
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprintf(stdout, "Node-RED will be installed into %s.\n", dataDir)
	confirmed, err := promptYesNo(stdin, stdout, "Continue with local Node-RED installation?", true)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(stdout, "Setup cancelled.")
		return nil
	}

	if _, err := s.runner.Run(dataDir, "npm", "install", "node-red"); err != nil {
		return fmt.Errorf("install local node-red: %w", err)
	}

	fmt.Fprintln(stdout, "Local Node-RED installation completed.")
	return nil
}

func (s EnvironmentService) StartPreflight(dataDir string) error {
	report := s.Diagnose(dataDir)
	if !report.NodeInstalled || !report.NPMInstalled {
		return fmt.Errorf("missing Node.js or npm; run `nrcc doctor` or `nrcc setup`")
	}
	if !report.NodeRedReady {
		return fmt.Errorf("local Node-RED is not installed in %s; run `nrcc setup`", dataDir)
	}
	return nil
}

func (s EnvironmentService) initializeDataDir(dataDir string) error {
	if err := platform.EnsureDir(filepath.Join(dataDir, "backups")); err != nil {
		return err
	}
	if err := platform.EnsureDir(filepath.Join(dataDir, "logs")); err != nil {
		return err
	}
	if err := platform.EnsureDir(filepath.Join(dataDir, "manifests")); err != nil {
		return err
	}

	packageJSON := map[string]any{
		"name":    "nrcc-local-runtime",
		"private": true,
		"version": "0.1.0",
	}
	packagePath := filepath.Join(dataDir, "package.json")
	if !platform.Exists(packagePath) {
		if err := platform.WriteJSONAtomic(packagePath, packageJSON); err != nil {
			return fmt.Errorf("write package.json: %w", err)
		}
	}

	configPath := filepath.Join(dataDir, "config.json")
	if !platform.Exists(configPath) {
		if err := platform.WriteJSONAtomic(configPath, DefaultAppConfig()); err != nil {
			return fmt.Errorf("write config.json: %w", err)
		}
	}

	if err := platform.WriteFileIfMissing(filepath.Join(dataDir, ".env.managed"), []byte(""), 0o600); err != nil {
		return fmt.Errorf("write .env.managed: %w", err)
	}

	settingsPath := filepath.Join(dataDir, "settings.js")
	if !platform.Exists(settingsPath) {
		settings, err := renderSettings(DefaultAppConfig())
		if err != nil {
			return err
		}
		if err := platform.WriteFileAtomic(settingsPath, []byte(settings), 0o644); err != nil {
			return fmt.Errorf("write settings.js: %w", err)
		}
	}

	return nil
}

func (s EnvironmentService) checkDataDir(dataDir string) model.EnvironmentCheck {
	if !platform.Exists(dataDir) {
		return model.EnvironmentCheck{
			Name:   "data-dir",
			Status: model.StatusWarn,
			Detail: fmt.Sprintf("data directory does not exist yet: %s", dataDir),
		}
	}

	if !platform.IsWritableDir(dataDir) {
		return model.EnvironmentCheck{
			Name:   "data-dir",
			Status: model.StatusFail,
			Detail: fmt.Sprintf("data directory is not writable: %s", dataDir),
		}
	}

	return model.EnvironmentCheck{
		Name:   "data-dir",
		Status: model.StatusOK,
		Detail: fmt.Sprintf("data directory is ready: %s", dataDir),
	}
}

func promptYesNo(stdin *os.File, stdout *os.File, label string, defaultYes bool) (bool, error) {
	reader := bufio.NewReader(stdin)
	suffix := "[Y/n]"
	if !defaultYes {
		suffix = "[y/N]"
	}

	fmt.Fprintf(stdout, "%s %s ", label, suffix)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer := strings.TrimSpace(strings.ToLower(input))
	if answer == "" {
		return defaultYes, nil
	}

	return answer == "y" || answer == "yes", nil
}

func printReport(stdout *os.File, report model.EnvironmentReport) {
	fmt.Fprintf(stdout, "OS: %s/%s\n", report.OS, report.Arch)
	fmt.Fprintf(stdout, "Data dir: %s\n", report.DataDir)
	for _, check := range report.Checks {
		fmt.Fprintf(stdout, "- [%s] %s: %s\n", check.Status, check.Name, check.Detail)
	}
}
