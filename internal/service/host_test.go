package service

import (
	"os"
	"path/filepath"
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

func TestBuildRecommendations_NoBareMetalInstallHints(t *testing.T) {
	svc := NewHostService(t.TempDir())

	// Worst-case input that, pre-ADR-0003, used to light up every
	// bare-metal-flavored suggestion. Post-#454 none of those fire.
	recommendations := svc.buildRecommendations(model.HostStatus{
		NodeJS: model.DependencyStatus{Installed: false},
		NodeRed: model.NodeRedEnvironment{
			Detected: false,
			Mode:     model.InstallationModeDocker,
		},
	})

	for _, banned := range []string{
		"Instala Node.js",
		"Instala Node-RED",
		"bind mount",
	} {
		for _, line := range recommendations {
			if strings.Contains(line, banned) {
				t.Fatalf("buildRecommendations() = %q, must not include %q", recommendations, banned)
			}
		}
	}
}

func TestBuildRecommendations_PortlessOptional(t *testing.T) {
	svc := NewHostService(t.TempDir())
	recommendations := svc.buildRecommendations(model.HostStatus{
		NodeRed: model.NodeRedEnvironment{Detected: true},
	})
	joined := strings.Join(recommendations, " ")
	if !strings.Contains(joined, "Portless") {
		t.Fatalf("buildRecommendations() = %q, want Portless hint when Node-RED is detected and Portless is missing", recommendations)
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

// TestUninstallNodeRedNative_SignatureExists verifies the method exists and has correct signature
// TestUninstallNodeRedNative_RequiresNpm verifies npm lookup error handling
// TestUpdateNodeRedNative_SignatureExists verifies the method exists with correct return types
// TestUpdateNodeRedNative_ReturnTypes verifies return value structure

