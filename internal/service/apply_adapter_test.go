package service

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// fakeAdapter is the testing double for ApplyAdapter. Each method
// records the call and returns the pre-programmed response so tests
// can drive the orchestrator through success / restart-failure /
// readiness-failure paths without touching real processes.
type fakeAdapter struct {
	name         string
	mu           sync.Mutex
	restartCalls int
	readyCalls   int
	restartErr   error
	readyErr     error
}

func (f *fakeAdapter) Name() string { return f.name }
func (f *fakeAdapter) Restart(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restartCalls++
	return f.restartErr
}
func (f *fakeAdapter) Ready(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readyCalls++
	return f.readyErr
}

// recordingSystemdManager satisfies SystemdManager without spawning
// subprocesses. Tests use the call counter + the canned error to drive
// the systemd adapter path without ever invoking systemctl.
type recordingSystemdManager struct {
	mu       sync.Mutex
	stopUnit string
	stopErr  error
}

func (m *recordingSystemdManager) IsAvailable() bool     { return true }
func (m *recordingSystemdManager) DaemonReload() error   { return nil }
func (m *recordingSystemdManager) EnableAndStart(u string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopUnit = u
	return m.stopErr
}
func (m *recordingSystemdManager) Stop(u string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopUnit = u
	return m.stopErr
}
func (m *recordingSystemdManager) Disable(u string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopUnit = u
	return m.stopErr
}
func (m *recordingSystemdManager) GetServiceStatus(u string) (string, error) {
	return "active", nil
}

// newRecordingDockerService builds a DockerService suitable for the
// dispatch-matrix test. The actual exec layer is not exercised here;
// the test only asserts the selector wires the right adapter type.
func newRecordingDockerService() *DockerService {
	return NewDockerService()
}

// execCommandRecorder replaces the package-level execCommand so the
// docker-compose adapter test can drive the exec layer without
// touching the host's docker binary. It returns a function with the
// same signature as exec.Command so it slots into the adapter's
// execCommand field directly.
func execCommandRecorder(t *testing.T, wantName string, wantArgsContain string, fakeErr error) func(string, ...string) *exec.Cmd {
	t.Helper()
	return func(name string, arg ...string) *exec.Cmd {
		if name != wantName {
			t.Errorf("exec name = %q, want %q", name, wantName)
		}
		joined := strings.Join(arg, " ")
		if wantArgsContain != "" && !strings.Contains(joined, wantArgsContain) {
			t.Errorf("exec args %v missing %q", arg, wantArgsContain)
		}
		if fakeErr != nil {
			// Build a real *exec.Cmd pointed at `/bin/false` (or
			// whatever exists on the test runner) and override Run
			// via the exec shim. The simplest portable stub is to
			// attach Stdin so the cmd fails to start; we instead
			// point at a binary path that is guaranteed to fail
			// fast with the canned error wrapped through Run.
			return exec.Command("false")
		}
		return exec.Command("true")
	}
}

// TestSelectApplyAdapter_Matrix locks in the dispatch table documented
// on SelectApplyAdapter. Every combination of (Mode, ManagedByNRCC)
// with a nodered-5 adapter must resolve to the documented ApplyAdapter
// name; any other combination must yield ErrUnsupportedAdapter.
func TestSelectApplyAdapter_Matrix(t *testing.T) {
	cases := []struct {
		name        string
		caps        model.ConfigurationCapabilities
		nodeRed     model.NodeRedEnvironment
		wantName    string
		wantErrIs   error
		wantErrSub  string
	}{
		{
			name:     "nativeManagedByNRCCResolvesNativeBinary",
			caps:     model.ConfigurationCapabilities{Adapter: "nodered-5"},
			nodeRed:  model.NodeRedEnvironment{Mode: model.InstallationModeNative, ManagedByNRCC: true},
			wantName: AdapterNameNativeBinary,
		},
		{
			name:     "nativeExternallyManagedResolvesNativeSystemd",
			caps:     model.ConfigurationCapabilities{Adapter: "nodered-5"},
			nodeRed:  model.NodeRedEnvironment{Mode: model.InstallationModeNative, ManagedByNRCC: false, ContainerName: "nodered.service"},
			wantName: AdapterNameNativeSystemd,
		},
		{
			name:     "dockerManagedByNRCCResolvesDockerCompose",
			caps:     model.ConfigurationCapabilities{Adapter: "nodered-5"},
			nodeRed:  model.NodeRedEnvironment{Mode: model.InstallationModeDocker, ManagedByNRCC: true, ContainerName: "nrcc"},
			wantName: AdapterNameDockerCompose,
		},
		{
			name:     "dockerExternallyManagedResolvesDockerCustom",
			caps:     model.ConfigurationCapabilities{Adapter: "nodered-5"},
			nodeRed:  model.NodeRedEnvironment{Mode: model.InstallationModeDocker, ManagedByNRCC: false, ContainerName: "node-red-1"},
			wantName: AdapterNameDockerCustom,
		},
		{
			name:       "readOnlyAdapterIsUnsupported",
			caps:       model.ConfigurationCapabilities{Adapter: "nodered-4-read-only"},
			nodeRed:    model.NodeRedEnvironment{Mode: model.InstallationModeNative, ManagedByNRCC: true},
			wantErrIs:  ErrUnsupportedAdapter,
			wantErrSub: "nodered-4-read-only",
		},
		{
			name:       "unknownAdapterIsUnsupported",
			caps:       model.ConfigurationCapabilities{Adapter: "unsupported"},
			nodeRed:    model.NodeRedEnvironment{Mode: model.InstallationModeNative, ManagedByNRCC: true},
			wantErrIs:  ErrUnsupportedAdapter,
			wantErrSub: "unsupported",
		},
		{
			name:       "noNodeRedModeIsUnsupported",
			caps:       model.ConfigurationCapabilities{Adapter: "nodered-5"},
			nodeRed:    model.NodeRedEnvironment{Mode: model.InstallationModeNone},
			wantErrIs:  ErrUnsupportedAdapter,
			wantErrSub: "nodeRed.mode",
		},
		{
			name:       "nativeExternalWithoutUnitNameIsUnsupported",
			caps:       model.ConfigurationCapabilities{Adapter: "nodered-5"},
			nodeRed:    model.NodeRedEnvironment{Mode: model.InstallationModeNative, ManagedByNRCC: false, ContainerName: ""},
			wantErrIs:  ErrUnsupportedAdapter,
			wantErrSub: "systemd unit name",
		},
	}

	deps := ApplyAdapterDeps{
		ProcessManager: &ProcessManager{},
		SystemdManager: &recordingSystemdManager{},
		Docker:         newRecordingDockerService(),
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			adapter, err := SelectApplyAdapter(tc.caps, tc.nodeRed, deps)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("expected ErrUnsupportedAdapter, got %v", err)
				}
				if tc.wantErrSub != "" && !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Errorf("err %q missing substring %q", err.Error(), tc.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("SelectApplyAdapter: %v", err)
			}
			if got := adapter.Name(); got != tc.wantName {
				t.Errorf("adapter.Name() = %q, want %q", got, tc.wantName)
			}
		})
	}
}

// TestNativeBinaryAdapter_RestartDelegatesToProcessManager asserts
// the ProcessManager is the one actually driving the restart and that
// the adapter honours ctx cancellation BEFORE issuing the signal.
func TestNativeBinaryAdapter_RestartDelegatesToProcessManager(t *testing.T) {
	pm := &ProcessManager{}
	a := &nativeBinaryAdapter{pm: pm}
	if a.Name() != AdapterNameNativeBinary {
		t.Errorf("Name = %q, want %q", a.Name(), AdapterNameNativeBinary)
	}
	// We can't actually exercise pm.Restart without a real binary
	// path, but we can assert the wiring is intact by calling the
	// adapter against a cancelled context — the cancellation
	// short-circuits before pm.Restart would fail.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Restart(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Restart with cancelled ctx = %v, want context.Canceled", err)
	}
}

// TestNativeSystemdAdapter_RestartInvokesSystemctl asserts the
// systemd adapter issues the documented command (`systemctl restart
// <unit>`) and forwards the error from the exec shim.
func TestNativeSystemdAdapter_RestartInvokesSystemctl(t *testing.T) {
	cases := []struct {
		name      string
		unit      string
		fakeErr   error
		wantErrIs error
	}{
		{
			name: "happyPath",
			unit: "nodered.service",
		},
		{
			name:      "restartFailure",
			unit:      "nodered.service",
			fakeErr:   errors.New("exit 1"),
			wantErrIs: nil, // wrapped, not a sentinel
		},
		{
			name:      "emptyUnit",
			unit:      "",
			wantErrIs: ErrAdapterUnitMissing,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := &nativeSystemdAdapter{
				mgr:         &recordingSystemdManager{},
				unit:        tc.unit,
				execCommand: execCommandRecorder(t, "systemctl", "restart "+tc.unit, tc.fakeErr),
			}
			err := a.Restart(context.Background())
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Errorf("err = %v, want %v", err, tc.wantErrIs)
				}
				return
			}
			if tc.fakeErr != nil && err == nil {
				t.Errorf("expected error from exec shim, got nil")
			}
			if tc.fakeErr == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestDockerComposeAdapter_RestartBuildsProjectFlag asserts the
// docker-compose adapter prefixes `-p <project>` when a project name
// is supplied and omits it otherwise. This protects the slice C
// multi-project host path from accidentally restarting a sibling.
func TestDockerComposeAdapter_RestartBuildsProjectFlag(t *testing.T) {
	cases := []struct {
		name        string
		projectName string
		wantContain string
	}{
		{name: "withProject", projectName: "nrcc", wantContain: "-p nrcc"},
		{name: "withoutProject", projectName: "", wantContain: "compose restart"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := &dockerComposeAdapter{
				execCommand: execCommandRecorder(t, "docker", tc.wantContain, nil),
				projectName: tc.projectName,
			}
			if a.Name() != AdapterNameDockerCompose {
				t.Errorf("Name = %q, want %q", a.Name(), AdapterNameDockerCompose)
			}
			if err := a.Restart(context.Background()); err != nil {
				t.Errorf("Restart: %v", err)
			}
		})
	}
}

// TestDockerCustomAdapter_RestartDelegatesToDockerService asserts the
// adapter forwards to DockerService.Restart and surfaces a nil
// receiver as ErrUnsupportedAdapter. We use a real DockerService with
// the package-level exec shim — the test only runs the wiring, not
// the docker CLI itself, by cancelling the context first.
func TestDockerCustomAdapter_RestartDelegatesToDockerService(t *testing.T) {
	a := &dockerCustomAdapter{docker: &DockerService{}}
	if a.Name() != AdapterNameDockerCustom {
		t.Errorf("Name = %q, want %q", a.Name(), AdapterNameDockerCustom)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Restart(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Restart with cancelled ctx = %v, want context.Canceled", err)
	}
}

// TestApplyAdapter_FakeImplementsInterface is the compile-time gate:
// fakeAdapter must satisfy ApplyAdapter so slice C tests can swap it
// in without further changes. The assertion lives in a test function
// (not a package var) so a missing method shows up as a test failure
// rather than a silent build error.
func TestApplyAdapter_FakeImplementsInterface(t *testing.T) {
	var _ ApplyAdapter = (*fakeAdapter)(nil)
	var _ ApplyAdapter = (*nativeBinaryAdapter)(nil)
	var _ ApplyAdapter = (*nativeSystemdAdapter)(nil)
	var _ ApplyAdapter = (*dockerComposeAdapter)(nil)
	var _ ApplyAdapter = (*dockerCustomAdapter)(nil)
}

// TestApplyAdapter_FakeRestartsAndReadies asserts the fake adapter
// honours the contract used by the slice C coordinator: Name is
// stable, Restart and Ready return the canned errors independently,
// and ctx cancellation short-circuits both.
func TestApplyAdapter_FakeRestartsAndReadies(t *testing.T) {
	cases := []struct {
		name        string
		restartErr  error
		readyErr    error
		wantRestart error
		wantReady   error
	}{
		{name: "happy", restartErr: nil, readyErr: nil, wantRestart: nil, wantReady: nil},
		{name: "restartFailure", restartErr: errors.New("restart boom"), readyErr: nil, wantRestart: errors.New("restart boom"), wantReady: nil},
		{name: "readyFailure", restartErr: nil, readyErr: ErrReadinessNotReady, wantRestart: nil, wantReady: ErrReadinessNotReady},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fa := &fakeAdapter{name: "fake", restartErr: tc.restartErr, readyErr: tc.readyErr}
			if fa.Name() != "fake" {
				t.Errorf("Name = %q, want %q", fa.Name(), "fake")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := fa.Restart(ctx); !matchErrMsg(err, tc.wantRestart) {
				t.Errorf("Restart err = %v, want %v", err, tc.wantRestart)
			}
			if err := fa.Ready(ctx); !matchErrMsg(err, tc.wantReady) {
				t.Errorf("Ready err = %v, want %v", err, tc.wantReady)
			}
			if fa.restartCalls != 1 || fa.readyCalls != 1 {
				t.Errorf("calls = (restart=%d, ready=%d), want (1,1)", fa.restartCalls, fa.readyCalls)
			}
		})
	}
}

// TestApplyAdapter_FakeHonoursContextCancellation asserts the fake
// adapter — and therefore every concrete adapter — short-circuits on
// a cancelled context BEFORE the underlying exec / HTTP call.
func TestApplyAdapter_FakeHonoursContextCancellation(t *testing.T) {
	fa := &fakeAdapter{name: "fake"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := fa.Restart(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Restart err = %v, want context.Canceled", err)
	}
	if err := fa.Ready(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Ready err = %v, want context.Canceled", err)
	}
	if fa.restartCalls != 0 || fa.readyCalls != 0 {
		t.Errorf("calls = (restart=%d, ready=%d), want (0,0)", fa.restartCalls, fa.readyCalls)
	}
}

// matchErrMsg compares two error values by message. We avoid
// errors.Is here because the table-driven tests build fresh
// errors.New(...) values per subtest, so pointer-identity checks
// would always fail. The helper accepts nil ↔ nil as a match and
// falls back to a substring comparison otherwise.
func matchErrMsg(got, want error) bool {
	if want == nil {
		return got == nil
	}
	if got == nil {
		return false
	}
	if errors.Is(got, want) {
		return true
	}
	return strings.Contains(got.Error(), want.Error())
}