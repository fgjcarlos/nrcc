package service

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
)

func TestEnsureSystemUserReturnsErrorWhenUseraddDoesNotCreateUser(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	withMockInstallerUserLookup(t, false, true)
	withMockInstallerExec(t, recordFile, "useradd", "2")

	installer := NewInstallerService(model.DefaultInstallLayout())
	err := installer.ensureSystemUser()
	if err == nil {
		t.Fatal("ensureSystemUser error = nil, want error")
	}
	if !strings.Contains(err.Error(), "user nrcc not found after useradd") {
		t.Fatalf("ensureSystemUser error = %v, want missing user verification error", err)
	}

	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("read record file: %v", err)
	}
	if !strings.Contains(string(data), "useradd") {
		t.Fatalf("useradd was not attempted; commands:\n%s", string(data))
	}
}

func TestEnsureSystemUserCreatesGroupBeforeUser(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	groupExists := false
	userExists := false
	withMockInstallerLookupFuncs(t, func(name string) (*user.User, error) {
		if name == "nrcc" && userExists {
			return &user.User{Uid: "1001", Gid: "1001", Username: name}, nil
		}
		return nil, user.UnknownUserError(name)
	}, func(name string) (*user.Group, error) {
		if name == "nrcc" && groupExists {
			return &user.Group{Gid: "1001", Name: name}, nil
		}
		return nil, user.UnknownGroupError(name)
	})
	withMockInstallerExecFunc(t, func(name string, args ...string) *exec.Cmd {
		if name == "groupadd" {
			groupExists = true
		}
		if name == "useradd" {
			if !groupExists {
				t.Fatal("useradd called before group exists")
			}
			userExists = true
		}
		cmdArgs := append([]string{"-test.run=TestHelperProcess", "--", name}, args...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "NRCC_TEST_RECORD_FILE="+recordFile)
		return cmd
	})

	installer := NewInstallerService(model.DefaultInstallLayout())
	if err := installer.ensureSystemUser(); err != nil {
		t.Fatalf("ensureSystemUser error = %v", err)
	}

	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("read record file: %v", err)
	}
	got := string(data)
	groupIdx := strings.Index(got, "groupadd -f nrcc")
	userIdx := strings.Index(got, "useradd -r -s /usr/sbin/nologin -d /nonexistent -g nrcc nrcc")
	if groupIdx == -1 || userIdx == -1 || groupIdx > userIdx {
		t.Fatalf("commands out of order or missing:\n%s", got)
	}
}

func withMockInstallerUserLookup(t *testing.T, userExists bool, groupExists bool) {
	t.Helper()
	withMockInstallerLookupFuncs(t, func(name string) (*user.User, error) {
		if name == "nrcc" && userExists {
			return &user.User{Uid: "1001", Gid: "1001", Username: name}, nil
		}
		return nil, user.UnknownUserError(name)
	}, func(name string) (*user.Group, error) {
		if name == "nrcc" && groupExists {
			return &user.Group{Gid: "1001", Name: name}, nil
		}
		return nil, user.UnknownGroupError(name)
	})
}

func withMockInstallerLookupFuncs(t *testing.T, userFunc func(string) (*user.User, error), groupFunc func(string) (*user.Group, error)) {
	t.Helper()
	originalLookupUser := lookupUser
	originalLookupGroup := lookupGroup
	lookupUser = userFunc
	lookupGroup = groupFunc
	t.Cleanup(func() {
		lookupUser = originalLookupUser
		lookupGroup = originalLookupGroup
	})
}

func withMockInstallerExec(t *testing.T, recordFile string, failCommand string, failExit string) {
	t.Helper()
	withMockInstallerExecFunc(t, func(name string, args ...string) *exec.Cmd {
		cmdArgs := append([]string{"-test.run=TestHelperProcess", "--", name}, args...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "NRCC_TEST_RECORD_FILE="+recordFile, "NRCC_TEST_FAIL_COMMAND="+failCommand, "NRCC_TEST_FAIL_EXIT="+failExit)
		return cmd
	})
}

func withMockInstallerExecFunc(t *testing.T, execFunc func(string, ...string) *exec.Cmd) {
	t.Helper()
	originalCommand := execCommand
	execCommand = execFunc
	t.Cleanup(func() {
		execCommand = originalCommand
	})
}

func TestEnsureInstalledBinaryCopiesCurrentExecutable(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "nrcc-linux-amd64")
	destination := filepath.Join(root, "usr", "local", "bin", "nrcc")
	if err := os.WriteFile(source, []byte("fake nrcc binary"), 0700); err != nil {
		t.Fatalf("write source binary: %v", err)
	}

	layout := model.DefaultInstallLayout()
	layout.BinaryPath = destination
	installer := NewInstallerService(layout)

	originalExecutablePath := executablePath
	executablePath = func() (string, error) { return source, nil }
	t.Cleanup(func() { executablePath = originalExecutablePath })

	if err := installer.ensureInstalledBinary(); err != nil {
		t.Fatalf("ensureInstalledBinary error = %v", err)
	}

	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read installed binary: %v", err)
	}
	if string(data) != "fake nrcc binary" {
		t.Fatalf("installed binary content = %q", string(data))
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat installed binary: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("installed binary mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestEnsureInstalledBinarySkipsWhenAlreadyAtDestination(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "usr", "local", "bin", "nrcc")
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		t.Fatalf("create destination dir: %v", err)
	}
	if err := os.WriteFile(destination, []byte("existing"), 0755); err != nil {
		t.Fatalf("write destination binary: %v", err)
	}

	layout := model.DefaultInstallLayout()
	layout.BinaryPath = destination
	installer := NewInstallerService(layout)

	originalExecutablePath := executablePath
	executablePath = func() (string, error) { return destination, nil }
	t.Cleanup(func() { executablePath = originalExecutablePath })

	if err := installer.ensureInstalledBinary(); err != nil {
		t.Fatalf("ensureInstalledBinary error = %v", err)
	}

	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read installed binary: %v", err)
	}
	if string(data) != "existing" {
		t.Fatalf("destination should not be overwritten when already installed, got %q", string(data))
	}
}

func TestInstallPortlessAddonsRunsRequestedSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	withMockExec(t, recordFile, "")

	layout := model.DefaultInstallLayout()
	layout.DataDir = t.TempDir()
	installer := NewInstallerService(layout)

	err := installer.installPortlessAddons(model.InstallOpts{
		Layout:             layout,
		WithPortless:       true,
		PortlessQuickSetup: true,
		PortlessTrust:      true,
	})
	if err != nil {
		t.Fatalf("installPortlessAddons error = %v", err)
	}

	data, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("read record file: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"/mock/bin/npm install -g portless",
		"/mock/bin/portless alias nrcc 3001",
		"/mock/bin/portless alias node-red 1880",
		"/mock/bin/portless proxy start",
		"/mock/bin/portless trust",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestGenerateEnvFileDisablesInteractiveBootstrap(t *testing.T) {
	withMockInstallerUserLookup(t, true, true)
	originalChown := chown
	chown = func(string, int, int) error { return nil }
	t.Cleanup(func() { chown = originalChown })

	root := t.TempDir()
	layout := model.DefaultInstallLayout()
	layout.ConfigDir = filepath.Join(root, "etc", "nrcc")
	layout.EnvFile = filepath.Join(layout.ConfigDir, "nrcc.env")
	layout.DataDir = filepath.Join(root, "var", "lib", "nrcc")

	if err := os.MkdirAll(layout.ConfigDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	installer := NewInstallerService(layout)
	if err := installer.generateEnvFile(nodeRedInstallDecision{Detected: true, Mode: model.NodeRedInstallModeSkip}); err != nil {
		t.Fatalf("generateEnvFile error = %v", err)
	}

	data, err := os.ReadFile(layout.EnvFile)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "NRCC_BOOTSTRAP_INTERACTIVE=false") {
		t.Fatalf("env file does not disable interactive bootstrap:\n%s", content)
	}
	if !strings.Contains(content, "NRCC_MANAGE_NODE_RED=true") {
		t.Fatalf("env file should keep managed Node-RED enabled:\n%s", content)
	}
}

func TestGenerateEnvFileDisablesManagedNodeRedWhenSkippedWithoutDetection(t *testing.T) {
	withMockInstallerUserLookup(t, true, true)
	originalChown := chown
	chown = func(string, int, int) error { return nil }
	t.Cleanup(func() { chown = originalChown })

	root := t.TempDir()
	layout := model.DefaultInstallLayout()
	layout.ConfigDir = filepath.Join(root, "etc", "nrcc")
	layout.EnvFile = filepath.Join(layout.ConfigDir, "nrcc.env")
	layout.DataDir = filepath.Join(root, "var", "lib", "nrcc")

	if err := os.MkdirAll(layout.ConfigDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	installer := NewInstallerService(layout)
	if err := installer.generateEnvFile(nodeRedInstallDecision{Mode: model.NodeRedInstallModeSkip}); err != nil {
		t.Fatalf("generateEnvFile error = %v", err)
	}

	data, err := os.ReadFile(layout.EnvFile)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "NRCC_MANAGE_NODE_RED=false") {
		t.Fatalf("env file should disable managed Node-RED when install skipped:\n%s", content)
	}
}

func TestGenerateEnvFilePersistsDetectedNativeNodeRedCommand(t *testing.T) {
	withMockInstallerUserLookup(t, true, true)
	originalChown := chown
	chown = func(string, int, int) error { return nil }
	t.Cleanup(func() { chown = originalChown })

	root := t.TempDir()
	layout := model.DefaultInstallLayout()
	layout.ConfigDir = filepath.Join(root, "etc", "nrcc")
	layout.EnvFile = filepath.Join(layout.ConfigDir, "nrcc.env")
	layout.DataDir = filepath.Join(root, "var", "lib", "nrcc")

	if err := os.MkdirAll(layout.ConfigDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	installer := NewInstallerService(layout)
	if err := installer.generateEnvFile(nodeRedInstallDecision{Detected: true, Mode: model.NodeRedInstallModeSkip, Command: "/usr/local/bin/node-red"}); err != nil {
		t.Fatalf("generateEnvFile error = %v", err)
	}

	data, err := os.ReadFile(layout.EnvFile)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "NODE_RED_CMD=/usr/local/bin/node-red") {
		t.Fatalf("env file should persist detected native Node-RED command:\n%s", content)
	}
	if strings.Contains(content, "NODE_RED_CMD=node-red") {
		t.Fatalf("env file should replace fallback node-red command:\n%s", content)
	}
}

func TestGenerateEnvFileUpdatesExistingNodeRedCommand(t *testing.T) {
	withMockInstallerUserLookup(t, true, true)
	originalChown := chown
	chown = func(string, int, int) error { return nil }
	t.Cleanup(func() { chown = originalChown })

	root := t.TempDir()
	layout := model.DefaultInstallLayout()
	layout.ConfigDir = filepath.Join(root, "etc", "nrcc")
	layout.EnvFile = filepath.Join(layout.ConfigDir, "nrcc.env")
	layout.DataDir = filepath.Join(root, "var", "lib", "nrcc")

	if err := os.MkdirAll(layout.ConfigDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	existing := strings.Join([]string{
		"PORT=3001",
		"JWT_SECRET=test-secret",
		"NRCC_MANAGE_NODE_RED=true",
		"NRCC_BOOTSTRAP_INTERACTIVE=true",
		"NODE_RED_CMD=node-red",
		"",
	}, "\n")
	if err := os.WriteFile(layout.EnvFile, []byte(existing), 0640); err != nil {
		t.Fatalf("seed env file: %v", err)
	}

	installer := NewInstallerService(layout)
	if err := installer.generateEnvFile(nodeRedInstallDecision{Mode: model.NodeRedInstallModeNative, Command: "/usr/local/bin/node-red"}); err != nil {
		t.Fatalf("generateEnvFile error = %v", err)
	}

	data, err := os.ReadFile(layout.EnvFile)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "NRCC_BOOTSTRAP_INTERACTIVE=false") {
		t.Fatalf("env file should force non-interactive bootstrap:\n%s", content)
	}
	if !strings.Contains(content, "NODE_RED_CMD=/usr/local/bin/node-red") {
		t.Fatalf("env file should update existing Node-RED command:\n%s", content)
	}
	if strings.Count(content, "NODE_RED_CMD=") != 1 {
		t.Fatalf("env file should contain a single NODE_RED_CMD entry:\n%s", content)
	}
}

func TestAvailableNodeRedInstallOptions(t *testing.T) {
	tests := []struct {
		name   string
		status model.HostStatus
		want   []string
	}{
		{name: "skip only", want: []string{"skip"}},
		{name: "native available", status: model.HostStatus{NPM: model.DependencyStatus{Installed: true}}, want: []string{"skip", "native"}},
		{name: "docker available", status: model.HostStatus{Docker: model.DependencyStatus{Installed: true}}, want: []string{"skip", "docker"}},
		{name: "native and docker available", status: model.HostStatus{NPM: model.DependencyStatus{Installed: true}, Docker: model.DependencyStatus{Installed: true}}, want: []string{"skip", "native", "docker"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := availableNodeRedInstallOptions(tt.status)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("availableNodeRedInstallOptions() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNativeNodeRedCommand(t *testing.T) {
	tests := []struct {
		name   string
		status model.HostStatus
		want   string
	}{
		{
			name: "prefers native environment executable",
			status: model.HostStatus{
				NodeRed:       model.NodeRedEnvironment{Mode: model.InstallationModeNative, Executable: "/custom/node-red"},
				NodeRedBinary: model.DependencyStatus{Installed: true, Command: "/usr/bin/node-red"},
			},
			want: "/custom/node-red",
		},
		{
			name: "falls back to detected binary command",
			status: model.HostStatus{
				NodeRedBinary: model.DependencyStatus{Installed: true, Command: "/usr/bin/node-red"},
			},
			want: "/usr/bin/node-red",
		},
		{
			name: "ignores docker executable",
			status: model.HostStatus{
				NodeRed: model.NodeRedEnvironment{Mode: model.InstallationModeDocker, Executable: "/container/node-red"},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nativeNodeRedCommand(tt.status)
			if got != tt.want {
				t.Fatalf("nativeNodeRedCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAllNodeRedInstallOptions(t *testing.T) {
	want := []string{"skip", "native", "docker"}
	got := allNodeRedInstallOptions()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("allNodeRedInstallOptions() = %#v, want %#v", got, want)
	}
}

func TestResolveNodeRedInstallDecision(t *testing.T) {
	installer := NewInstallerService(model.DefaultInstallLayout())
	originalDetectHostStatus := detectHostStatus
	originalSelectNodeRedInstallMode := selectNodeRedInstallMode
	originalInstallNodeRedNative := installNodeRedNative
	originalInstallNodeRedDocker := installNodeRedDocker
	t.Cleanup(func() {
		detectHostStatus = originalDetectHostStatus
		selectNodeRedInstallMode = originalSelectNodeRedInstallMode
		installNodeRedNative = originalInstallNodeRedNative
		installNodeRedDocker = originalInstallNodeRedDocker
	})

	tests := []struct {
		name       string
		opts       model.InstallOpts
		status     model.HostStatus
		selectMode model.NodeRedInstallMode
		want       nodeRedInstallDecision
		wantPrompt bool
	}{
		{
			name:   "detected skips prompt",
			status: model.HostStatus{NodeRed: model.NodeRedEnvironment{Detected: true, Mode: model.InstallationModeNative, Executable: "/usr/local/bin/node-red"}},
			want:   nodeRedInstallDecision{Mode: model.NodeRedInstallModeSkip, Detected: true, Command: "/usr/local/bin/node-red"},
		},
		{
			name:   "flag native wins",
			opts:   model.InstallOpts{NodeRedMode: model.NodeRedInstallModeNative},
			status: model.HostStatus{NodeRedBinary: model.DependencyStatus{Installed: true, Command: "/usr/bin/node-red"}},
			want:   nodeRedInstallDecision{Mode: model.NodeRedInstallModeNative, Detected: true, Command: "/usr/bin/node-red"},
		},
		{
			name:   "non interactive defaults to skip",
			status: model.HostStatus{Interactive: false},
			want:   nodeRedInstallDecision{Mode: model.NodeRedInstallModeSkip},
		},
		{
			name:   "skip prompt defaults to skip",
			opts:   model.InstallOpts{SkipPrompt: true},
			status: model.HostStatus{Interactive: true},
			want:   nodeRedInstallDecision{Mode: model.NodeRedInstallModeSkip},
		},
		{
			name:       "interactive prompts for available options",
			status:     model.HostStatus{Interactive: true},
			selectMode: model.NodeRedInstallModeDocker,
			want:       nodeRedInstallDecision{Mode: model.NodeRedInstallModeDocker},
			wantPrompt: true,
		},
		{
			name:       "interactive still offers explicit modes when host lacks helpers",
			status:     model.HostStatus{Interactive: true},
			selectMode: model.NodeRedInstallModeNative,
			want:       nodeRedInstallDecision{Mode: model.NodeRedInstallModeNative},
			wantPrompt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detectHostStatus = func(*HostService) model.HostStatus {
				return tt.status
			}

			prompted := false
			selectNodeRedInstallMode = func(options []string) (model.NodeRedInstallMode, error) {
				prompted = true
				if tt.wantPrompt {
					wantOptions := allNodeRedInstallOptions()
					if !reflect.DeepEqual(options, wantOptions) {
						t.Fatalf("prompt options = %#v, want %#v", options, wantOptions)
					}
				}
				return tt.selectMode, nil
			}

			got, err := installer.resolveNodeRedInstallDecision(&HostService{}, tt.opts)
			if err != nil {
				t.Fatalf("resolveNodeRedInstallDecision error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveNodeRedInstallDecision() = %#v, want %#v", got, tt.want)
			}
			if prompted != tt.wantPrompt {
				t.Fatalf("prompted = %v, want %v", prompted, tt.wantPrompt)
			}
		})
	}
}

func TestApplyNodeRedInstallDecisionCapturesInstalledNativeCommand(t *testing.T) {
	installer := NewInstallerService(model.DefaultInstallLayout())
	originalDetectHostStatus := detectHostStatus
	originalInstallNodeRedNative := installNodeRedNative
	t.Cleanup(func() {
		detectHostStatus = originalDetectHostStatus
		installNodeRedNative = originalInstallNodeRedNative
	})

	installed := false
	installNodeRedNative = func(*HostService) error {
		installed = true
		return nil
	}
	detectHostStatus = func(*HostService) model.HostStatus {
		if !installed {
			return model.HostStatus{}
		}
		return model.HostStatus{NodeRed: model.NodeRedEnvironment{Mode: model.InstallationModeNative, Executable: "/usr/local/bin/node-red"}}
	}

	got, err := installer.applyNodeRedInstallDecision(&HostService{}, nodeRedInstallDecision{Mode: model.NodeRedInstallModeNative})
	if err != nil {
		t.Fatalf("applyNodeRedInstallDecision error = %v", err)
	}
	if !installed {
		t.Fatal("native install was not attempted")
	}
	if got.Command != "/usr/local/bin/node-red" {
		t.Fatalf("applyNodeRedInstallDecision command = %q, want %q", got.Command, "/usr/local/bin/node-red")
	}
}
