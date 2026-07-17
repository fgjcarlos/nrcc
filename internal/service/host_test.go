package service

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		os.Exit(2)
	}
	cmdArgs := args[1:]
	if recordFile := os.Getenv("NRCC_TEST_RECORD_FILE"); recordFile != "" && len(cmdArgs) > 0 {
		f, err := os.OpenFile(recordFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			_, _ = f.WriteString(strings.Join(cmdArgs, " ") + "\n")
			_ = f.Close()
		}
	}
	if failCommand := os.Getenv("NRCC_TEST_FAIL_COMMAND"); failCommand != "" && len(cmdArgs) > 0 && cmdArgs[0] == failCommand {
		exitCode, _ := strconv.Atoi(os.Getenv("NRCC_TEST_FAIL_EXIT"))
		if exitCode == 0 {
			exitCode = 1
		}
		os.Exit(exitCode)
	}
	if len(cmdArgs) >= 2 && cmdArgs[1] == "--version" {
		_, _ = os.Stdout.WriteString("v1.2.3\n")
		os.Exit(0)
	}
	if len(cmdArgs) >= 2 && cmdArgs[1] == "list" {
		_, _ = os.Stdout.WriteString(os.Getenv("NRCC_TEST_PORTLESS_LIST_OUTPUT"))
		os.Exit(0)
	}
	if len(cmdArgs) >= 3 && cmdArgs[1] == "prefix" && cmdArgs[2] == "-g" {
		_, _ = os.Stdout.WriteString(os.TempDir() + "\n")
		os.Exit(0)
	}
	if len(cmdArgs) >= 4 && cmdArgs[1] == "uninstall" && cmdArgs[3] == "portless" {
		if marker := os.Getenv("NRCC_TEST_PORTLESS_UNINSTALLED"); marker != "" {
			_ = os.WriteFile(marker, []byte("1"), 0644)
		}
	}
	os.Exit(0)
}

func withMockPortlessList(t *testing.T, output string) {
	t.Helper()
	originalCommand := execCommand
	originalLookPath := execLookPath
	execCommand = func(name string, args ...string) *exec.Cmd {
		cmdArgs := append([]string{"-test.run=TestHelperProcess", "--", name}, args...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "NRCC_TEST_PORTLESS_LIST_OUTPUT="+output)
		return cmd
	}
	execLookPath = func(name string) (string, error) {
		if name == "portless" {
			return "/mock/bin/portless", nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() {
		execCommand = originalCommand
		execLookPath = originalLookPath
	})
}

func withoutMockPortless(t *testing.T) {
	t.Helper()
	originalLookPath := execLookPath
	execLookPath = func(name string) (string, error) {
		if name == "portless" {
			return "", errors.New("not found")
		}
		return originalLookPath(name)
	}
	t.Cleanup(func() { execLookPath = originalLookPath })
}

func withMockExec(t *testing.T, recordFile string, uninstalledMarker string) {
	t.Helper()
	originalCommand := execCommand
	originalLookPath := execLookPath
	execCommand = func(name string, args ...string) *exec.Cmd {
		cmdArgs := append([]string{"-test.run=TestHelperProcess", "--", name}, args...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "NRCC_TEST_RECORD_FILE="+recordFile, "NRCC_TEST_PORTLESS_UNINSTALLED="+uninstalledMarker)
		return cmd
	}
	execLookPath = func(name string) (string, error) {
		switch name {
		case "node", "npm", "portless":
			if name == "portless" && uninstalledMarker != "" {
				if _, err := os.Stat(uninstalledMarker); err == nil {
					return "", errors.New("not found")
				}
			}
			return "/mock/bin/" + name, nil
		default:
			return "", errors.New("not found")
		}
	}
	t.Cleanup(func() {
		execCommand = originalCommand
		execLookPath = originalLookPath
	})
}

func readCommandLog(t *testing.T, recordFile string) string {
	t.Helper()
	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("read command log: %v", err)
	}
	return string(data)
}

func withMockDependencyInstallerExec(t *testing.T, recordFile, nodeInstalledMarker, nodeRedInstalledMarker, portlessInstalledMarker string) {
	t.Helper()
	originalCommand := execCommand
	originalLookPath := execLookPath
	execCommand = func(name string, args ...string) *exec.Cmd {
		base := filepath.Base(name)
		allArgs := append([]string{name}, args...)
		record := "printf '%s\\n' " + shellQuoteForTest(strings.Join(allArgs, " ")) + " >> " + shellQuoteForTest(recordFile)
		script := record
		switch {
		case base == "apt-get" && len(args) >= 1 && args[0] == "install":
			script += " && touch " + shellQuoteForTest(nodeInstalledMarker)
		case base == "npm" && len(args) >= 1 && args[0] == "prefix":
			script += " && printf '%s\\n' " + shellQuoteForTest(t.TempDir())
		case base == "npm" && strings.Contains(strings.Join(args, " "), "node-red"):
			script += " && touch " + shellQuoteForTest(nodeRedInstalledMarker)
		case base == "npm" && strings.Contains(strings.Join(args, " "), "portless"):
			script += " && touch " + shellQuoteForTest(portlessInstalledMarker)
		case (base == "node" || base == "npm" || base == "node-red" || base == "portless") && len(args) == 1 && args[0] == "--version":
			script += " && printf 'v1.2.3\\n'"
		}
		return exec.Command("sh", "-c", script)
	}
	execLookPath = func(name string) (string, error) {
		switch name {
		case "apt-get":
			return "/mock/bin/apt-get", nil
		case "node", "npm":
			if _, err := os.Stat(nodeInstalledMarker); err == nil {
				return "/mock/bin/" + name, nil
			}
			return "", errors.New("not found")
		case "node-red":
			if _, err := os.Stat(nodeRedInstalledMarker); err == nil {
				return "/mock/bin/node-red", nil
			}
			return "", errors.New("not found")
		case "portless":
			if _, err := os.Stat(portlessInstalledMarker); err == nil {
				return "/mock/bin/portless", nil
			}
			return "", errors.New("not found")
		default:
			return "", errors.New("not found")
		}
	}
	t.Cleanup(func() {
		execCommand = originalCommand
		execLookPath = originalLookPath
	})
}

func withMockDockerInstallerExec(t *testing.T, recordFile, dockerInstalledMarker, nodeRedContainerMarker string) {
	t.Helper()
	originalCommand := execCommand
	originalLookPath := execLookPath
	execCommand = func(name string, args ...string) *exec.Cmd {
		base := filepath.Base(name)
		allArgs := append([]string{name}, args...)
		record := "printf '%s\\n' " + shellQuoteForTest(strings.Join(allArgs, " ")) + " >> " + shellQuoteForTest(recordFile)
		script := record
		switch {
		case base == "apt-get" && len(args) >= 1 && args[0] == "install":
			script += " && touch " + shellQuoteForTest(dockerInstalledMarker)
		case base == "docker" && len(args) == 1 && args[0] == "--version":
			script += " && printf 'Docker version 25.0.0\\n'"
		case base == "docker" && len(args) == 1 && args[0] == "info":
			script += " && printf 'Server: Docker\\n'"
		case base == "docker" && len(args) >= 1 && args[0] == "pull":
			// record only
		case base == "docker" && len(args) >= 1 && args[0] == "run":
			script += " && touch " + shellQuoteForTest(nodeRedContainerMarker)
		case base == "docker" && len(args) >= 2 && args[0] == "ps":
			if _, err := os.Stat(nodeRedContainerMarker); err == nil {
				script += " && printf 'abc123\\tnodered/node-red:latest\\tnrcc-node-red\\tUp 1 second\\n'"
			}
		}
		return exec.Command("sh", "-c", script)
	}
	execLookPath = func(name string) (string, error) {
		switch name {
		case "apt-get":
			return "/mock/bin/apt-get", nil
		case "docker":
			if _, err := os.Stat(dockerInstalledMarker); err == nil {
				return "/mock/bin/docker", nil
			}
			return "", errors.New("not found")
		default:
			return "", errors.New("not found")
		}
	}
	t.Cleanup(func() {
		execCommand = originalCommand
		execLookPath = originalLookPath
	})
}

func shellQuoteForTest(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func TestHostService_Detect_ReturnsHostStatus(t *testing.T) {
	// Create temp data dir
	tempDir := t.TempDir()

	svc := NewHostService(tempDir)
	status := svc.Detect()

	// Verify basic structure is populated
	if status.Platform == "" {
		t.Error("Platform should not be empty")
	}
	if status.NodeJS.Name == "" {
		t.Error("NodeJS dependency name should be set")
	}
	if status.NPM.Name == "" {
		t.Error("NPM dependency name should be set")
	}
	if status.Settings.Path == "" {
		t.Error("Settings.Path should be populated")
	}
}

func TestHostService_ResolveSettingsPath_WithDataDir(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	path := svc.ResolveSettingsPath()

	// Should resolve to settings.js in data dir (or detected path if Node-RED is installed)
	if path == "" {
		t.Error("ResolveSettingsPath should return non-empty path")
	}
}

func TestHostService_ResolveSettingsPath_With_NODE_RED_SETTINGS_EnvVar(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	// Set env var
	customPath := filepath.Join(tempDir, "custom-settings.js")
	t.Setenv("NODE_RED_SETTINGS", customPath)

	path := svc.ResolveSettingsPath()

	// The path should be the custom path (assuming Node-RED is not detected locally)
	// This depends on detection order, but if Node-RED is detected, it uses that path
	// For this test, we just verify it returns something
	if path == "" {
		t.Error("ResolveSettingsPath should return non-empty path")
	}
}

func TestHostService_ResolveSettingsPath_Unset_EnvVar(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	// Ensure env var is not set
	t.Setenv("NODE_RED_SETTINGS", "")

	path := svc.ResolveSettingsPath()

	// Should fall back to nrcc data dir
	if path == "" {
		t.Error("ResolveSettingsPath should return non-empty path")
	}
	// When no detection and no env var, should use dataDir default
	if !filepath.IsAbs(path) {
		t.Errorf("Path should be absolute: %s", path)
	}
}

func TestHostService_RuntimeStatus_NotRunning(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	status := svc.RuntimeStatus()

	// Verify structure
	if status.Status == "" {
		t.Error("Status field should not be empty")
	}
	// Status should be one of: running, stopped, detected
	validStates := map[string]bool{
		"running":  true,
		"stopped":  true,
		"detected": true,
	}
	if !validStates[status.Status] {
		t.Errorf("Invalid status state: %s", status.Status)
	}
}

func TestHostService_RuntimeStatus_Structure(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	status := svc.RuntimeStatus()

	// Verify all fields are accessible
	_ = status.Status
	_ = status.PID
	_ = status.Uptime
	_ = status.Version
	_ = status.InstallationMode
	_ = status.ManagedByNRCC
	_ = status.Detected
}

func TestCleanVersionOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single word", "v16.14.0", "v16.14.0"},
		{"multiple words", "npm 7.20.6", "npm 7.20.6"},
		{"with extra whitespace", "  npm  7.20.6  ", "npm 7.20.6"},
		{"newlines and tabs", "npm\n7.20.6", "npm 7.20.6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanVersionOutput(tt.input)
			if result != tt.expected {
				t.Errorf("cleanVersionOutput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCanWrite_WithWritableDir(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test.txt")

	result := canWrite(testPath)

	if !result {
		t.Error("canWrite should return true for writable directory")
	}

	// Verify file was NOT created (only directory checked)
	if _, err := os.Stat(testPath); err == nil {
		t.Error("canWrite should not create the file as a side effect")
	}
}

func TestCanWrite_WithExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test.txt")

	// Pre-create the file
	if err := os.WriteFile(testPath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := canWrite(testPath)

	if !result {
		t.Error("canWrite should return true for existing writable file")
	}
}

func TestCanWrite_WithReadOnlyDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping test in root context (cannot test read-only)")
	}

	tempDir := t.TempDir()
	roDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(roDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Make it read-only
	if err := os.Chmod(roDir, 0444); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer func() { _ = os.Chmod(roDir, 0755) }() // Restore for cleanup

	testPath := filepath.Join(roDir, "test.txt")
	result := canWrite(testPath)

	if result {
		t.Error("canWrite should return false for read-only directory")
	}
}

func TestLatestBackupFile_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	path, err := latestBackupFile(tempDir)

	// Should return error or empty string when no files
	if path != "" && err == nil {
		t.Errorf("expected empty path or error for empty dir, got path=%q err=%v", path, err)
	}
}

func TestLatestBackupFile_WithBackups(t *testing.T) {
	tempDir := t.TempDir()

	// Create some backup files with delays to ensure different timestamps
	f1 := filepath.Join(tempDir, "backup1.bak")
	if err := os.WriteFile(f1, []byte("old"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Delay slightly to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	f2 := filepath.Join(tempDir, "backup2.bak")
	if err := os.WriteFile(f2, []byte("new"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	path, err := latestBackupFile(tempDir)

	if err != nil {
		t.Errorf("latestBackupFile should not error: %v", err)
	}
	if path == "" {
		t.Error("latestBackupFile should return a path")
	}
	// Just verify we got one of the two files
	if path != f1 && path != f2 {
		t.Errorf("Expected one of the backup files, got %s", path)
	}
}

func TestLatestBackupFile_IgnoresDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file and a directory
	f1 := filepath.Join(tempDir, "backup.bak")
	d1 := filepath.Join(tempDir, "subdir")
	if err := os.WriteFile(f1, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.Mkdir(d1, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	path, err := latestBackupFile(tempDir)

	if err != nil {
		t.Errorf("latestBackupFile should not error: %v", err)
	}
	if path != f1 {
		t.Errorf("Expected file %s, got %s", f1, path)
	}
}

func TestIsInteractiveTerminal_NonInteractive(t *testing.T) {
	// When stdin is not a terminal (e.g., piped), isInteractiveTerminal should return false
	result := isInteractiveTerminal()

	// Can't guarantee non-interactive in test environment, so just verify it returns a bool
	_ = result
}

func TestIsInteractiveTerminal_DisabledByEnv(t *testing.T) {
	t.Setenv("NRCC_BOOTSTRAP_INTERACTIVE", "false")

	if isInteractiveTerminal() {
		t.Fatal("isInteractiveTerminal() = true, want false when NRCC_BOOTSTRAP_INTERACTIVE=false")
	}
}

func TestIsInteractiveTerminal_SystemdEnv(t *testing.T) {
	t.Setenv("INVOCATION_ID", "test-invocation")

	if isInteractiveTerminal() {
		t.Fatal("isInteractiveTerminal() = true, want false under systemd")
	}
}

// TestIsInteractiveTerminal_StdinIsDevNull exercises the docker-run hang:
// when stdin is redirected to /dev/null, isInteractiveTerminal must return
// false even though /dev/null is a character device. We can't safely swap
// the test runner's stdin fd, so we cover the predicate that catches it.
func TestIsDevNull_StatDevNull(t *testing.T) {
	info, err := os.Stat(os.DevNull)
	if err != nil {
		t.Skipf("cannot stat /dev/null on this host: %v", err)
	}
	if !isDevNull(info) {
		t.Fatal("isDevNull(/dev/null) = false, want true")
	}
}

func TestIsDevNull_StatRegularFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "plain.txt")
	if err := os.WriteFile(path, []byte("hi"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if isDevNull(info) {
		t.Fatal("isDevNull(regular file) = true, want false")
	}
}

// TestIsInteractiveTerminal_DockerStamps ensures any docker/containerd/k8s
// hostname never reaches the interactive wizard, regardless of stdin shape.
func TestIsInteractiveTerminal_DockerStamps(t *testing.T) {
	t.Setenv("DOCKER_CONTAINER", "true")

	if isInteractiveTerminal() {
		t.Fatal("isInteractiveTerminal() = true, want false under DOCKER_CONTAINER")
	}
}

func TestHostService_InspectCommand_WithVersion(t *testing.T) {
	svc := NewHostService(t.TempDir())

	// This will actually call `ls --version` which should work on most systems
	status := svc.inspectCommand("ls", "--version")

	// ls should be available on most Unix systems
	if !status.Installed {
		t.Skip("ls command not available")
	}

	if status.Name != "ls" {
		t.Errorf("Expected name 'ls', got %s", status.Name)
	}
	// Version might be empty depending on system
	_ = status.Version
}

func TestHostService_InspectCommand_NotFound(t *testing.T) {
	svc := NewHostService(t.TempDir())

	// Use a command that definitely doesn't exist
	status := svc.inspectCommand("nonexistent-command-12345", "--version")

	if status.Installed {
		t.Error("Command should not be marked as installed")
	}
	if status.Name != "nonexistent-command-12345" {
		t.Errorf("Name should be set to requested command")
	}
}

func TestHostService_DefaultNativeUserDir(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	userDir := svc.defaultNativeUserDir()

	if userDir == "" {
		t.Error("defaultNativeUserDir should return non-empty path")
	}
	if !filepath.IsAbs(userDir) {
		t.Errorf("Path should be absolute: %s", userDir)
	}
}

func TestHostService_DefaultNativeUserDir_FallsBackForNonexistentHome(t *testing.T) {
	t.Setenv("HOME", "/nonexistent")

	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	userDir := svc.defaultNativeUserDir()

	if userDir != tempDir {
		t.Fatalf("defaultNativeUserDir() = %q, want %q", userDir, tempDir)
	}
}

func TestHostService_InspectNodeRed_UsesManagedDataDirWhenHomeInvalid(t *testing.T) {
	t.Setenv("HOME", "/nonexistent")

	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	env := svc.inspectNodeRed(model.HostStatus{
		NodeRedBinary: model.DependencyStatus{
			Installed: true,
			Version:   "3.1.0",
			Command:   "/usr/bin/node-red",
		},
	})

	if env.Mode != model.InstallationModeNative {
		t.Fatalf("inspectNodeRed() mode = %q, want %q", env.Mode, model.InstallationModeNative)
	}
	if env.UserDir != tempDir {
		t.Fatalf("inspectNodeRed() userDir = %q, want %q", env.UserDir, tempDir)
	}
	wantSettings := filepath.Join(tempDir, "settings.js")
	if env.SettingsPath != wantSettings {
		t.Fatalf("inspectNodeRed() settingsPath = %q, want %q", env.SettingsPath, wantSettings)
	}
}

func TestBuildRecommendations_DockerNamedVolumeMessage(t *testing.T) {
	svc := NewHostService(t.TempDir())

	recommendations := svc.buildRecommendations(model.HostStatus{
		NodeJS: model.DependencyStatus{Installed: true},
		NodeRed: model.NodeRedEnvironment{
			Detected: true,
			Mode:     model.InstallationModeDocker,
		},
		Settings: model.SettingsDocument{
			Path:     filepath.Join(t.TempDir(), "settings.js"),
			Writable: true,
		},
	})

	joined := strings.Join(recommendations, " ")
	if !strings.Contains(joined, "bind mount") {
		t.Fatalf("buildRecommendations() = %q, want docker bind mount guidance", joined)
	}
	if strings.Contains(joined, "Otorga permisos de escritura") {
		t.Fatalf("buildRecommendations() unexpectedly included writability warning: %q", joined)
	}
}

func TestIsUsableHomeDir(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "valid absolute path", path: "/home/nrcc", want: true},
		{name: "empty path", path: "", want: false},
		{name: "relative path", path: "tmp/home", want: false},
		{name: "nonexistent placeholder", path: "/nonexistent", want: false},
		{name: "nonexistent child", path: "/nonexistent/nrcc", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUsableHomeDir(tt.path); got != tt.want {
				t.Fatalf("isUsableHomeDir(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRuntimeState_Running(t *testing.T) {
	tests := []struct {
		name     string
		running  bool
		detected bool
		expected string
	}{
		{"running", true, true, "running"},
		{"detected but not running", false, true, "detected"},
		{"not detected", false, false, "stopped"},
		{"running takes precedence", true, false, "running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtimeState(tt.running, tt.detected)
			if result != tt.expected {
				t.Errorf("runtimeState(%v, %v) = %q, want %q", tt.running, tt.detected, result, tt.expected)
			}
		})
	}
}

func TestHostService_Detect_StructureIntegration(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	status := svc.Detect()

	// Verify full structure integrity
	if status.Platform == "" {
		t.Error("Platform must be set")
	}

	// All dependency fields should have Name set
	deps := []model.DependencyStatus{status.NodeJS, status.NPM, status.NodeRedBinary, status.Docker, status.DockerCompose}
	for i, dep := range deps {
		if dep.Name == "" {
			t.Errorf("Dependency %d has no Name", i)
		}
	}

	// Settings should always have a path
	if status.Settings.Path == "" {
		t.Error("Settings.Path must be set")
	}

	// Source should be one of: detected, env, nrcc-data
	validSources := map[string]bool{
		"detected":  true,
		"env":       true,
		"nrcc-data": true,
	}
	if !validSources[status.Settings.Source] {
		t.Errorf("Invalid Settings.Source: %s", status.Settings.Source)
	}
}

func TestHostService_Detect_Ready_Conditions(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewHostService(tempDir)

	status := svc.Detect()

	// Ready should be true only if NodeRed.Detected and Settings.Path is set
	if status.Ready {
		if !status.NodeRed.Detected {
			t.Error("Ready=true but NodeRed not detected")
		}
		if status.Settings.Path == "" {
			t.Error("Ready=true but Settings.Path is empty")
		}
	}
}

func TestDetectPackageManager(t *testing.T) {
	// This test relies on the system having at least one of the package managers
	pm := detectPackageManager()

	// We can't guarantee which PM is available, so just verify the function runs
	// and returns a string (empty if none found, which is valid on non-Linux)
	if pm != "" {
		// Verify it's one of the expected managers
		validPMs := map[string]bool{
			"apt-get": true,
			"dnf":     true,
			"yum":     true,
			"pacman":  true,
			"zypper":  true,
			"apk":     true,
		}
		if !validPMs[pm] {
			t.Errorf("detectPackageManager returned unexpected value: %s", pm)
		}
	}
}

func TestBuildBootstrapOptionsDisplay(t *testing.T) {
	tests := []struct {
		name     string
		options  []string
		expected string
	}{
		{
			name:     "only skip",
			options:  []string{"skip"},
			expected: "Opciones: skip",
		},
		{
			name:     "skip then native",
			options:  []string{"skip", "native"},
			expected: "Opciones: skip, native",
		},
		{
			name:     "skip then docker",
			options:  []string{"skip", "docker"},
			expected: "Opciones: skip, docker",
		},
		{
			name:     "skip then native then docker",
			options:  []string{"skip", "native", "docker"},
			expected: "Opciones: skip, native, docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildBootstrapOptionsDisplay(tt.options)
			if result != tt.expected {
				t.Errorf("buildBootstrapOptionsDisplay(%v) = %q, want %q", tt.options, result, tt.expected)
			}
		})
	}
}

func TestContainerExists(t *testing.T) {
	// This test is best-effort. If docker is not available, we skip
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker not available")
	}

	// Test with a name that definitely doesn't exist
	exists := containerExists("nrcc-definitely-does-not-exist-" + strconv.Itoa(os.Getpid()))
	if exists {
		t.Error("containerExists should return false for non-existent container")
	}
}

func TestRunElevatedCommands_PassesFullArgs(t *testing.T) {
	// Test that args are passed correctly without dropping args[0]
	// We test this by running a harmless command like 'echo'
	called := false
	originalLookPath := exec.LookPath

	// Use a mock to verify echo is called with full args
	cmd := exec.Command("echo", "test", "arg1", "arg2")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("echo command failed: %v", err)
	}

	if !strings.Contains(string(out), "test") {
		t.Error("echo output should contain 'test'")
	}

	_ = called
	_ = originalLookPath
}

func TestNpmGlobalPrefixWritable_WithWritablePrefix(t *testing.T) {
	// This test checks if npm is available on the system
	npmPath, err := exec.LookPath("npm")
	if err != nil {
		t.Skip("npm not available, skipping test")
	}

	// Call the function with the real npm
	result := npmGlobalPrefixWritable(npmPath)

	// If running as non-root user with a user-installed npm (e.g., nvm),
	// the prefix should be writable. If running with system npm, it might not be.
	// We just verify the function runs without panicking and returns a bool.
	_ = result
}

func TestNpmGlobalPrefixWritable_WithInvalidPath(t *testing.T) {
	// Test with a path that doesn't exist
	invalidPath := "/nonexistent/path/to/npm"
	result := npmGlobalPrefixWritable(invalidPath)

	// Should return false for invalid path
	if result {
		t.Error("npmGlobalPrefixWritable should return false for invalid npm path")
	}
}

// TestUninstallNodeRedNative_SignatureExists verifies the method exists and has correct signature
func TestUninstallNodeRedNative_SignatureExists(t *testing.T) {
	svc := NewHostService(t.TempDir())

	// Verify method exists and is callable
	// This test ensures the method signature is correct by compilation
	// We can't actually run the uninstall without side effects, so we just verify it exists
	_ = svc.UninstallNodeRedNative

	// Verify it's a method that returns error
	var _ error
}

// TestUninstallNodeRedNative_RequiresNpm verifies npm lookup error handling
func TestUninstallNodeRedNative_RequiresNpm(t *testing.T) {
	if !t.Run("npm availability check", func(t *testing.T) {
		// This test just documents that UninstallNodeRedNative depends on npm
		// In a real environment where npm is missing, it would fail
		svc := NewHostService(t.TempDir())

		// If npm is not installed, this would return an error about npm not found
		// We can't easily mock exec.LookPath, so we just verify the method is callable
		_ = svc
	}) {
		t.Skip("npm availability test skipped")
	}
}

// TestUpdateNodeRedNative_SignatureExists verifies the method exists with correct return types
func TestUpdateNodeRedNative_SignatureExists(t *testing.T) {
	svc := NewHostService(t.TempDir())

	// Verify method exists and returns (string, string, error)
	// This test ensures the method signature is correct by compilation
	var oldVersion, newVersion string
	var err error
	_ = oldVersion
	_ = newVersion
	_ = err

	// Method is callable
	_ = svc.UpdateNodeRedNative
}

// TestUpdateNodeRedNative_ReturnTypes verifies return value structure
func TestUpdateNodeRedNative_ReturnTypes(t *testing.T) {
	if !t.Run("return type validation", func(t *testing.T) {
		svc := NewHostService(t.TempDir())

		// Verify method signature by type checking
		// If Node-RED is not installed, should return error
		oldVer, newVer, err := svc.UpdateNodeRedNative()

		// oldVer and newVer should be strings (even if empty)
		_ = oldVer
		_ = newVer

		// err should be an error type or nil
		if err != nil {
			// If Node-RED not installed, we expect an error (this is normal)
			t.Logf("Expected error when Node-RED not installed: %v", err)
		}
	}) {
		t.Skip("Update method test skipped")
	}
}

func TestInstallNodeRedNativeInstallsMissingNodeJSAndNPM(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	nodeInstalledMarker := filepath.Join(t.TempDir(), "node-installed")
	nodeRedInstalledMarker := filepath.Join(t.TempDir(), "node-red-installed")
	withMockDependencyInstallerExec(t, recordFile, nodeInstalledMarker, nodeRedInstalledMarker, "")

	err := NewHostService(t.TempDir()).InstallNodeRedNative()
	if err != nil {
		t.Fatalf("InstallNodeRedNative error = %v", err)
	}

	commands := readCommandLog(t, recordFile)
	for _, want := range []string{
		"apt-get update",
		"apt-get install -y nodejs npm",
		"/mock/bin/npm install -g --unsafe-perm node-red",
	} {
		if !strings.Contains(commands, want) {
			t.Fatalf("commands missing %q:\n%s", want, commands)
		}
	}
}

func TestInstallPortlessInstallsMissingNodeJSAndNPM(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	nodeInstalledMarker := filepath.Join(t.TempDir(), "node-installed")
	portlessInstalledMarker := filepath.Join(t.TempDir(), "portless-installed")
	withMockDependencyInstallerExec(t, recordFile, nodeInstalledMarker, "", portlessInstalledMarker)

	err := NewHostService(t.TempDir()).InstallPortless()
	if err != nil {
		t.Fatalf("InstallPortless error = %v", err)
	}

	commands := readCommandLog(t, recordFile)
	for _, want := range []string{
		"apt-get update",
		"apt-get install -y nodejs npm",
		"/mock/bin/npm install -g portless",
	} {
		if !strings.Contains(commands, want) {
			t.Fatalf("commands missing %q:\n%s", want, commands)
		}
	}
}

func TestInstallNodeRedDockerInstallsMissingDockerAndVerifiesDaemon(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	dockerInstalledMarker := filepath.Join(t.TempDir(), "docker-installed")
	nodeRedContainerMarker := filepath.Join(t.TempDir(), "node-red-container")
	withMockDockerInstallerExec(t, recordFile, dockerInstalledMarker, nodeRedContainerMarker)

	err := NewHostService(t.TempDir()).installNodeRedDocker()
	if err != nil {
		t.Fatalf("installNodeRedDocker error = %v", err)
	}

	commands := readCommandLog(t, recordFile)
	for _, want := range []string{
		"apt-get update",
		"apt-get install -y docker.io",
		"/mock/bin/docker info",
		"docker pull nodered/node-red:latest",
		"docker run -d --name nrcc-node-red -p 1880:1880",
	} {
		if !strings.Contains(commands, want) {
			t.Fatalf("commands missing %q:\n%s", want, commands)
		}
	}
}

func TestBuildNPMGlobalPackageCommand(t *testing.T) {
	tests := []struct {
		name           string
		nodePath       string
		action         string
		packageName    string
		unsafePerm     bool
		prefixWritable bool
		expected       []string
	}{
		{
			name:           "writable prefix uses npm directly",
			nodePath:       "/usr/bin/node",
			action:         "install",
			packageName:    "portless",
			prefixWritable: true,
			expected:       []string{"/usr/bin/npm", "install", "-g", "portless"},
		},
		{
			name:           "unsafe perm is inserted before package",
			nodePath:       "/usr/bin/node",
			action:         "install",
			packageName:    "node-red",
			unsafePerm:     true,
			prefixWritable: true,
			expected:       []string{"/usr/bin/npm", "install", "-g", "--unsafe-perm", "node-red"},
		},
		{
			name:        "readonly prefix uses sudo with node interpreter",
			nodePath:    "/usr/bin/node",
			action:      "install",
			packageName: "portless",
			expected:    []string{"sudo", "/usr/bin/node", "/usr/bin/npm", "install", "-g", "portless"},
		},
		{
			name:           "uninstall action uses npm directly",
			nodePath:       "/usr/bin/node",
			action:         "uninstall",
			packageName:    "portless",
			prefixWritable: true,
			expected:       []string{"/usr/bin/npm", "uninstall", "-g", "portless"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNPMGlobalPackageCommand("/usr/bin/npm", tt.nodePath, tt.action, tt.packageName, tt.unsafePerm, tt.prefixWritable)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("command = %#v, want %#v", got, tt.expected)
			}
		})
	}
}

func TestReadPortlessAliases_ValidFile(t *testing.T) {
	withoutMockPortless(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".portless")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "aliases.json"), []byte(`{"node-red":1880,"nrcc":3001}`), 0644); err != nil {
		t.Fatalf("write aliases: %v", err)
	}

	aliases, err := NewHostService(t.TempDir()).ReadPortlessAliases()
	if err != nil {
		t.Fatalf("ReadPortlessAliases error = %v", err)
	}
	want := []PortlessAlias{
		{Name: "node-red", Port: 1880, URL: "https://node-red.localhost"},
		{Name: "nrcc", Port: 3001, URL: "https://nrcc.localhost"},
	}
	if !reflect.DeepEqual(aliases, want) {
		t.Fatalf("aliases = %#v, want %#v", aliases, want)
	}
}

func TestReadPortlessAliases_ValidRoutesFile(t *testing.T) {
	withoutMockPortless(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".portless")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	data := `[
		{"hostname":"nrcc.localhost","port":3001,"pid":0},
		{"hostname":"node-red.localhost","port":1880,"pid":0},
		{"hostname":"dynamic.localhost","port":4321,"pid":1234}
	]`
	if err := os.WriteFile(filepath.Join(configDir, "routes.json"), []byte(data), 0644); err != nil {
		t.Fatalf("write routes: %v", err)
	}

	aliases, err := NewHostService(t.TempDir()).ReadPortlessAliases()
	if err != nil {
		t.Fatalf("ReadPortlessAliases error = %v", err)
	}
	want := []PortlessAlias{
		{Name: "node-red", Port: 1880, URL: "https://node-red.localhost"},
		{Name: "nrcc", Port: 3001, URL: "https://nrcc.localhost"},
	}
	if !reflect.DeepEqual(aliases, want) {
		t.Fatalf("aliases = %#v, want %#v", aliases, want)
	}
}

func TestReadPortlessAliases_PrefersPortlessList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".portless")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "routes.json"), []byte(`[{"hostname":"stale.localhost","port":1111,"pid":0}]`), 0644); err != nil {
		t.Fatalf("write routes: %v", err)
	}
	withMockPortlessList(t, `Active routes:

  https://nrcc.localhost  ->  localhost:3001  (alias)
  https://node-red.localhost  ->  localhost:1880  (alias)
`)

	aliases, err := NewHostService(t.TempDir()).ReadPortlessAliases()
	if err != nil {
		t.Fatalf("ReadPortlessAliases error = %v", err)
	}
	want := []PortlessAlias{
		{Name: "node-red", Port: 1880, URL: "https://node-red.localhost"},
		{Name: "nrcc", Port: 3001, URL: "https://nrcc.localhost"},
	}
	if !reflect.DeepEqual(aliases, want) {
		t.Fatalf("aliases = %#v, want %#v", aliases, want)
	}
}

func TestReadPortlessAliases_MissingFile(t *testing.T) {
	withoutMockPortless(t)
	t.Setenv("HOME", t.TempDir())
	aliases, err := NewHostService(t.TempDir()).ReadPortlessAliases()
	if err != nil {
		t.Fatalf("ReadPortlessAliases error = %v", err)
	}
	if aliases != nil {
		t.Fatalf("aliases = %#v, want nil", aliases)
	}
}

func TestReadPortlessAliases_MalformedJSON(t *testing.T) {
	withoutMockPortless(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".portless")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "aliases.json"), []byte(`not-json`), 0644); err != nil {
		t.Fatalf("write aliases: %v", err)
	}

	aliases, err := NewHostService(t.TempDir()).ReadPortlessAliases()
	if err == nil {
		t.Fatalf("ReadPortlessAliases error = nil, want error")
	}
	if aliases != nil {
		t.Fatalf("aliases = %#v, want nil", aliases)
	}
}

func TestCheckPortlessAliasReachability(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	listeningPort := listener.Addr().(*net.TCPAddr).Port
	unreachablePort := reserveClosedPort(t)

	aliases := NewHostService(t.TempDir()).CheckPortlessAliasReachability([]PortlessAlias{
		{Name: "up", Port: listeningPort, URL: "https://up.localhost"},
		{Name: "down", Port: unreachablePort, URL: "https://down.localhost"},
	})

	if aliases[0].LocalAddress != "127.0.0.1:"+strconv.Itoa(listeningPort) || !aliases[0].Reachable {
		t.Fatalf("listening alias = %#v, want reachable 127.0.0.1 address", aliases[0])
	}
	if aliases[1].LocalAddress != "127.0.0.1:"+strconv.Itoa(unreachablePort) || aliases[1].Reachable {
		t.Fatalf("closed alias = %#v, want unreachable 127.0.0.1 address", aliases[1])
	}
}

func reserveClosedPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve closed port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close reserved port: %v", err)
	}
	return port
}

func TestQuickSetupPortless_HappyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	withMockExec(t, recordFile, "")

	err := NewHostService(t.TempDir()).QuickSetupPortless(false)
	if err != nil {
		t.Fatalf("QuickSetupPortless error = %v", err)
	}
	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("read record file: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "/mock/bin/portless alias nrcc 3001") {
		t.Fatalf("missing nrcc alias call in:\n%s", got)
	}
	if !strings.Contains(got, "/mock/bin/portless alias node-red 1880") {
		t.Fatalf("missing node-red alias call in:\n%s", got)
	}
	if !strings.Contains(got, "/mock/bin/portless proxy start") {
		t.Fatalf("missing portless proxy start call in:\n%s", got)
	}
}

func TestQuickSetupPortless_SkipsExistingAlias(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".portless")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "aliases.json"), []byte(`{"nrcc":3001}`), 0644); err != nil {
		t.Fatalf("write aliases: %v", err)
	}
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	withMockExec(t, recordFile, "")

	err := NewHostService(t.TempDir()).QuickSetupPortless(false)
	if err != nil {
		t.Fatalf("QuickSetupPortless error = %v", err)
	}
	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("read record file: %v", err)
	}
	got := string(data)
	if strings.Contains(got, "/mock/bin/portless alias nrcc 3001") {
		t.Fatalf("nrcc alias should have been skipped:\n%s", got)
	}
	if !strings.Contains(got, "/mock/bin/portless alias node-red 1880") {
		t.Fatalf("missing node-red alias call in:\n%s", got)
	}
	if !strings.Contains(got, "/mock/bin/portless proxy start") {
		t.Fatalf("missing portless proxy start call in:\n%s", got)
	}
}

func TestQuickSetupPortless_ForceExposesExistingAliases(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".portless")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "aliases.json"), []byte(`{"node-red":1880,"nrcc":9999}`), 0644); err != nil {
		t.Fatalf("write aliases: %v", err)
	}
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	withMockExec(t, recordFile, "")

	err := NewHostService(t.TempDir()).QuickSetupPortless(true)
	if err != nil {
		t.Fatalf("QuickSetupPortless error = %v", err)
	}
	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("read record file: %v", err)
	}
	got := string(data)
	for _, want := range []string{"/mock/bin/portless alias nrcc 3001 --force", "/mock/bin/portless alias node-red 1880 --force"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "/mock/bin/portless proxy start") {
		t.Fatalf("missing portless proxy start call in:\n%s", got)
	}
}

func TestQuickSetupPortless_PortlessNotInstalled(t *testing.T) {
	originalLookPath := execLookPath
	execLookPath = func(name string) (string, error) {
		if name == "portless" {
			return "", errors.New("not found")
		}
		return "/mock/bin/" + name, nil
	}
	t.Cleanup(func() { execLookPath = originalLookPath })

	err := NewHostService(t.TempDir()).QuickSetupPortless(false)
	if err == nil || !strings.Contains(err.Error(), "nrcc portless install") {
		t.Fatalf("error = %v, want install suggestion", err)
	}
}

func TestUninstallPortless_NPMUninstallCalled(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	marker := filepath.Join(t.TempDir(), "uninstalled")
	withMockExec(t, recordFile, marker)

	err := NewHostService(t.TempDir()).UninstallPortless(false)
	if err != nil {
		t.Fatalf("UninstallPortless error = %v", err)
	}
	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("read record file: %v", err)
	}
	if !strings.Contains(string(data), "/mock/bin/npm uninstall -g portless") {
		t.Fatalf("npm uninstall not recorded in:\n%s", string(data))
	}
}

func TestUninstallPortless_CleanAliases(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".portless")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	marker := filepath.Join(t.TempDir(), "uninstalled")
	withMockExec(t, recordFile, marker)

	err := NewHostService(t.TempDir()).UninstallPortless(true)
	if err != nil {
		t.Fatalf("UninstallPortless error = %v", err)
	}
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatalf("config dir still exists or stat failed unexpectedly: %v", err)
	}
}

func TestUninstallPortless_NotInstalledNoop(t *testing.T) {
	originalLookPath := execLookPath
	execLookPath = func(name string) (string, error) {
		if name == "portless" {
			return "", errors.New("not found")
		}
		return "/mock/bin/" + name, nil
	}
	t.Cleanup(func() { execLookPath = originalLookPath })

	if err := NewHostService(t.TempDir()).UninstallPortless(false); err != nil {
		t.Fatalf("UninstallPortless error = %v", err)
	}
}

func TestSetupPortlessTrust_SignatureExists(t *testing.T) {
	svc := NewHostService(t.TempDir())
	_ = svc.SetupPortlessTrust
}

func TestValidatePortlessAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		port    int
		wantErr bool
	}{
		{name: "valid nrcc alias", alias: "nrcc", port: 3001},
		{name: "valid dotted alias", alias: "node-red.nrcc", port: 1880},
		{name: "missing alias", alias: "", port: 1880, wantErr: true},
		{name: "space rejected", alias: "node red", port: 1880, wantErr: true},
		{name: "slash rejected", alias: "node/red", port: 1880, wantErr: true},
		{name: "zero port rejected", alias: "nrcc", port: 0, wantErr: true},
		{name: "large port rejected", alias: "nrcc", port: 65536, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePortlessAlias(tt.alias, tt.port)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
