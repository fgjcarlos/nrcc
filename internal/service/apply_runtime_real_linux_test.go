//go:build linux

package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

const realAcceptanceEnv = "NRCC_APPLY_REAL_ACCEPT"

type realAcceptanceAudit struct {
	mu     sync.Mutex
	events []recordedAudit
}

func (r *realAcceptanceAudit) Log(_ *http.Request, _, action, _, result string, meta map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	event := recordedAudit{Action: action, Result: result, MetaCopy: make(map[string]string, len(meta))}
	for key, value := range meta {
		event.MetaCopy[key] = value
	}
	r.events = append(r.events, event)
}

func (r *realAcceptanceAudit) snapshot() []recordedAudit {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedAudit(nil), r.events...)
}

type realAcceptanceFixture struct {
	root, userDir, settings string
	port                    string
	binary                  string
	config                  *ConfigService
	manager                 *ProcessManager
	adapter                 ApplyAdapter
	recorder                *realAcceptanceAudit
	coordinator             *ApplyCoordinator
	baseline                []byte
	baselineConfig          model.NodeRedConfig
}

func TestRealAcceptanceOwnedListenerProofRejectsForeignProcess(t *testing.T) {
	pid, port, stop := startTestOwnedListener(t)
	defer stop()
	if err := proveOwnedListener(os.Getpid(), port); err == nil {
		t.Fatalf("accepted loopback listener on port %s owned by child PID %d as parent PID %d", port, pid, os.Getpid())
	}
	if err := proveOwnedListener(pid, port); err != nil {
		t.Fatalf("rejected listener held by exact child PID %d: %v", pid, err)
	}
}

// TestRealAcceptanceOwnedListenerHelperProcess is a test-owned child used
// only to exercise listener/PID identity without relying on host services.
func TestRealAcceptanceOwnedListenerHelperProcess(t *testing.T) {
	if os.Getenv("NRCC_TEST_OWNED_LISTENER_HELPER") != "1" {
		return
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("child listener: %v", err)
	}
	defer func() { _ = listener.Close() }()
	fmt.Println(listener.Addr().(*net.TCPAddr).Port)
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func startTestOwnedListener(t *testing.T) (int, string, func()) {
	t.Helper()
	// #nosec G204,G702 -- test fixture: re-executes the test binary itself, not a tainted input.
	cmd := exec.Command(os.Args[0], "-test.run=^TestRealAcceptanceOwnedListenerHelperProcess$")
	cmd.Env = append(os.Environ(), "NRCC_TEST_OWNED_LISTENER_HELPER=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("create test-owned listener stdout: %v", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("create test-owned listener stdin: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start test-owned listener process: %v", err)
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	line := make(chan string, 1)
	readErr := make(chan error, 1)
	go func() {
		value, err := bufio.NewReader(stdout).ReadString('\n')
		if err != nil {
			readErr <- err
			return
		}
		line <- strings.TrimSpace(value)
	}()
	var stopOnce sync.Once
	stopDone := make(chan struct{})
	stop := func() {
		stopOnce.Do(func() {
			defer close(stopDone)
			_ = stdin.Close()
			select {
			case <-waited:
			case <-time.After(5 * time.Second):
				_ = cmd.Process.Kill()
				select {
				case <-waited:
				case <-time.After(5 * time.Second):
					t.Errorf("test-owned listener PID %d did not reap after kill", cmd.Process.Pid)
				}
			}
		})
		<-stopDone
	}
	t.Cleanup(stop)
	select {
	case value := <-line:
		return cmd.Process.Pid, value, stop
	case err := <-readErr:
		stop()
		t.Fatalf("read test-owned listener port: %v", err)
	case <-time.After(5 * time.Second):
		stop()
		t.Fatal("test-owned listener did not publish its port")
	}
	return 0, "", stop
}

// proveOwnedListener currently checks that the exact endpoint responds;
// the regression requires PID-to-socket ownership instead of liveness alone.
func proveOwnedListener(pid int, port string) error {
	if pid <= 0 {
		return fmt.Errorf("invalid owned PID %d", pid)
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		return fmt.Errorf("invalid loopback port %q", port)
	}
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := os.ReadDir(fdDir)
	if err != nil {
		return fmt.Errorf("read owned PID %d file descriptors: %w", pid, err)
	}
	ownedSockets := make(map[string]struct{})
	for _, fd := range fds {
		target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
		if err != nil {
			continue
		}
		if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
			ownedSockets[strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")] = struct{}{}
		}
	}
	if len(ownedSockets) == 0 {
		return fmt.Errorf("owned PID %d has no inspectable sockets", pid)
	}
	tcpTable, err := os.Open("/proc/net/tcp")
	if err != nil {
		return fmt.Errorf("read IPv4 TCP table: %w", err)
	}
	defer func() { _ = tcpTable.Close() }()
	wantLocal := fmt.Sprintf("0100007F:%04X", portNumber)
	scanner := bufio.NewScanner(tcpTable)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 || fields[1] != wantLocal || fields[3] != "0A" {
			continue
		}
		if _, ok := ownedSockets[fields[9]]; ok {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan IPv4 TCP table: %w", err)
	}
	return fmt.Errorf("loopback listener 127.0.0.1:%s is not owned by PID %d", port, pid)
}

func TestRealAcceptanceCleanupDrainsPausedApply(t *testing.T) {
	gate := &releaseGate{done: make(chan struct{})}
	workDone := make(chan struct{})
	go func() {
		<-gate.done
		close(workDone)
	}()
	var cancelled bool
	if err := drainApplyWork(func() { cancelled = true }, gate.Release, workDone, 25*time.Millisecond); err != nil {
		t.Fatalf("cleanup did not release and drain paused apply: %v", err)
	}
	if !cancelled {
		t.Fatal("cleanup did not cancel the apply caller")
	}
}

type releaseGate struct {
	done chan struct{}
	once sync.Once
}

func newReleaseGate() *releaseGate { return &releaseGate{done: make(chan struct{})} }

func (g *releaseGate) Release() { g.once.Do(func() { close(g.done) }) }

// drainApplyWork is the bounded test cleanup barrier. Its ordering matters:
// release the phase barrier before waiting, so assertion failures cannot
// strand a coordinator call ahead of ProcessManager teardown.
func drainApplyWork(cancel context.CancelFunc, release func(), done <-chan struct{}, timeout time.Duration) error {
	cancel()
	release()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.New("apply worker did not drain")
	}
}

func TestApplyRuntimeRealAcceptance(t *testing.T) {
	if os.Getenv(realAcceptanceEnv) != "1" {
		t.Skipf("real Node-RED acceptance is opt-in; set %s=1 to run owned-process tests", realAcceptanceEnv)
	}
	binary := os.Getenv("NODE_RED_CMD")
	if binary == "" {
		var err error
		binary, err = exec.LookPath("node-red")
		if err != nil {
			t.Fatalf("real acceptance prerequisite unavailable: resolve node-red from PATH: %v", err)
		}
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatalf("resolve Node-RED executable: %v", err)
	}
	// #nosec G204,G702 -- test prerequisite: NODE_RED_CMD or exec.LookPath resolves the operator-provided Node-RED binary, not request input.
	version, err := exec.Command(binary, "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(version), "Node-RED v") {
		t.Fatalf("real acceptance prerequisite unavailable: %q --version: %v (%s)", binary, err, strings.TrimSpace(string(version)))
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("real acceptance prerequisite unavailable: resolve node from PATH: %v", err)
	}
	// #nosec G204,G702 -- test prerequisite: exec.LookPath resolves the operator-provided node binary, not request input.
	nodeVersion, err := exec.Command(node, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("Node version probe: %v (%s)", err, strings.TrimSpace(string(nodeVersion)))
	}
	t.Logf("real runtime: %s; %s", strings.TrimSpace(string(version)), strings.TrimSpace(string(nodeVersion)))

	t.Run("failed_TLS_candidate_restores_bytes_and_structured_state", func(t *testing.T) {
		f := newRealAcceptanceFixture(t, binary)
		missingKey := filepath.Join(f.root, "missing-key.pem")
		missingCert := filepath.Join(f.root, "missing-cert.pem")
		// #nosec G304,G703 -- test fixture: TLS material paths are built under the fixture temp root, not user input.
		beforeKey, keyErr := os.ReadFile(missingKey)
		// #nosec G304,G703 -- test fixture: TLS material paths are built under the fixture temp root, not user input.
		beforeCert, certErr := os.ReadFile(missingCert)
		if !errors.Is(keyErr, os.ErrNotExist) || !errors.Is(certErr, os.ErrNotExist) || len(beforeKey) != 0 || len(beforeCert) != 0 {
			t.Fatalf("TLS fixture paths must start absent beneath temp root: key=%v cert=%v", keyErr, certErr)
		}
		candidate := f.baselineConfig
		candidate.UIPort++
		candidate.Https = &model.HttpsConfig{Key: missingKey, Cert: missingCert}
		result, applyErr := f.coordinator.Apply(context.Background(), f.request(&candidate))
		if applyErr == nil || result.Path != "" {
			t.Fatalf("candidate with missing TLS material unexpectedly succeeded: result=%+v", result)
		}
		var typed *ApplyError
		if !errors.As(applyErr, &typed) || typed.Stage != ApplyStageReady {
			t.Fatalf("candidate failure=%v; want actual readiness-stage failure", applyErr)
		}
		f.assertRuntimeOwnedReady(t)
		got, err := os.ReadFile(f.settings)
		if err != nil || string(got) != string(f.baseline) {
			t.Fatalf("settings.js not restored byte-for-byte: read=%v equal=%v", err, string(got) == string(f.baseline))
		}
		stored, err := f.config.Get()
		if err != nil {
			t.Fatalf("read restored structured configuration: %v", err)
		}
		if stored.Https != nil || stored.UIPort != f.baselineConfig.UIPort {
			t.Fatalf("structured config not restored: https=%+v uiPort=%d want nil/%d", stored.Https, stored.UIPort, f.baselineConfig.UIPort)
		}
		if _, err := os.Stat(missingKey); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("apply mutated TLS key path outside its write-set: %v", err)
		}
		if _, err := os.Stat(missingCert); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("apply mutated TLS certificate path outside its write-set: %v", err)
		}
		f.assertAudit(t, "apply.failure", "ready")
		f.assertAuditMeta(t, "apply.failure", "apply_adapter", AdapterNameNativeBinary)
		f.assertAuditMeta(t, "apply.failure", "apply_operation", AdapterNameNativeBinary)
		f.assertAudit(t, "apply.rollback.started", "ready")
		f.assertAudit(t, "apply.rollback.succeeded", "ready")
		f.assertAuditMeta(t, "apply.rollback.succeeded", "apply_rollback_adapter", AdapterNameNativeBinary)
		f.assertAuditMeta(t, "apply.rollback.succeeded", "apply_rollback_operation", AdapterNameNativeBinary)
		f.assertNoAudit(t, "apply.success")
	})

	t.Run("caller_cancellation_recovers_independently", func(t *testing.T) {
		f := newRealAcceptanceFixture(t, binary)
		entered, gate := make(chan struct{}), newReleaseGate()
		wrapped := &cancelDuringRealReady{ApplyAdapter: f.adapter, entered: entered, release: gate.done}
		f.coordinator = f.runtimeCoordinator(wrapped)
		candidate := f.baselineConfig
		candidate.UIPort++
		ctx, cancel := context.WithCancel(context.Background())
		finished := make(chan error, 1)
		workDone := make(chan struct{})
		go func() {
			defer close(workDone)
			_, err := f.coordinator.Apply(ctx, f.request(&candidate))
			finished <- err
		}()
		registerApplyWorkCleanup(t, cancel, gate, workDone)
		select {
		case <-entered:
		case <-time.After(45 * time.Second):
			cancel()
			t.Fatalf("candidate never reached real readiness wrapper; owned process state=%+v", f.manager.Status())
		}
		candidatePID := f.ownedPID(t)
		cancel()
		select {
		case applyErr := <-finished:
			if applyErr == nil || !errors.Is(applyErr, context.Canceled) {
				t.Fatalf("cancelled apply error=%v; want caller cancellation", applyErr)
			}
		case <-time.After(45 * time.Second):
			t.Fatal("cancellation-independent recovery exceeded 45 seconds")
		}
		gate.Release()
		f.assertRuntimeOwnedReady(t)
		if f.ownedPID(t) == candidatePID {
			t.Fatalf("recovery did not replace candidate process lifetime (pid=%d)", candidatePID)
		}
		got, err := os.ReadFile(f.settings)
		if err != nil || string(got) != string(f.baseline) {
			t.Fatalf("cancelled apply did not restore settings bytes: read=%v", err)
		}
		f.assertStructuredBaseline(t)
		f.assertAudit(t, "apply.rollback.succeeded", "ready")
		f.assertNoAudit(t, "apply.success")
	})

	t.Run("same_path_apply_rejects_while_real_apply_is_pending", func(t *testing.T) {
		f := newRealAcceptanceFixture(t, binary)
		entered, gate := make(chan struct{}), newReleaseGate()
		wrapped := &barrierRealReady{ApplyAdapter: f.adapter, entered: entered, release: gate.done}
		f.coordinator = f.runtimeCoordinator(wrapped)
		candidate := f.baselineConfig
		candidate.UIPort++
		firstDone := make(chan error, 1)
		workDone := make(chan struct{})
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		go func() {
			defer close(workDone)
			_, err := f.coordinator.Apply(ctx, f.request(&candidate))
			firstDone <- err
		}()
		registerApplyWorkCleanup(t, cancel, gate, workDone)
		select {
		case <-entered:
		case <-time.After(45 * time.Second):
			gate.Release()
			t.Fatalf("first apply did not reach readiness barrier; owned process state=%+v", f.manager.Status())
		}
		pendingBytes, err := os.ReadFile(f.settings)
		if err != nil {
			t.Fatal(err)
		}
		pid := f.ownedPID(t)
		second := f.request(nil)
		second.Content = string(pendingBytes)
		_, secondErr := f.coordinator.Apply(context.Background(), second)
		if !errors.Is(secondErr, ErrApplyInFlight) {
			t.Fatalf("competing apply error=%v; want ErrApplyInFlight", secondErr)
		}
		afterBytes, err := os.ReadFile(f.settings)
		if err != nil || string(afterBytes) != string(pendingBytes) || f.ownedPID(t) != pid {
			t.Fatalf("rejected apply mutated pending bytes or process: read=%v same=%v pid=%d want=%d", err, string(afterBytes) == string(pendingBytes), f.ownedPID(t), pid)
		}
		f.assertAudit(t, "apply.failure", "start")
		gate.Release()
		select {
		case err := <-firstDone:
			if err != nil {
				t.Fatalf("first real apply after barrier release: %v", err)
			}
		case <-time.After(45 * time.Second):
			t.Fatal("first real apply did not finish after barrier release")
		}
		f.assertRuntimeOwnedReady(t)
		stored, err := f.config.Get()
		if err != nil || stored.UIPort != candidate.UIPort {
			t.Fatalf("first structured apply state=%+v err=%v want uiPort=%d", stored, err, candidate.UIPort)
		}
		f.assertAudit(t, "apply.success", "ready")
	})
}

type cancelDuringRealReady struct {
	ApplyAdapter
	entered chan struct{}
	release chan struct{}
	mu      sync.Mutex
	first   bool
}

type ownedRealApplyAdapter struct {
	ApplyAdapter
	manager *ProcessManager
	port    string
}

func (a *ownedRealApplyAdapter) Ready(ctx context.Context) error {
	readyCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	pid, err := waitOwnedLoopbackListener(readyCtx, a.manager, a.port)
	if err != nil {
		return err
	}
	if err := a.ApplyAdapter.Ready(readyCtx); err != nil {
		return err
	}
	afterPID, err := currentOwnedProcessPID(a.manager)
	if err != nil {
		return err
	}
	if afterPID != pid {
		return fmt.Errorf("owned runtime changed during readiness: PID %d became %d", pid, afterPID)
	}
	if err := proveOwnedListener(afterPID, a.port); err != nil {
		return fmt.Errorf("readiness endpoint lost owned listener: %w", err)
	}
	return nil
}

func waitOwnedLoopbackListener(ctx context.Context, manager *ProcessManager, port string) (int, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	var observedPID int
	for {
		if manager == nil {
			return 0, errors.New("ProcessManager is nil")
		}
		if manager.IsExternalMode() {
			return 0, errors.New("ProcessManager attached to external runtime")
		}
		status := manager.Status()
		if status.Status != "running" || status.PID <= 0 {
			return 0, fmt.Errorf("ProcessManager has no tracked running child: status=%q PID=%d", status.Status, status.PID)
		}
		if observedPID == 0 {
			observedPID = status.PID
		} else if observedPID != status.PID {
			return 0, fmt.Errorf("owned runtime changed while waiting for listener: PID %d became %d", observedPID, status.PID)
		}
		if err := proveOwnedListener(status.PID, port); err == nil {
			return status.PID, nil
		}
		select {
		case <-ctx.Done():
			return 0, fmt.Errorf("owned PID %d did not bind loopback port %s: %w", observedPID, port, ctx.Err())
		case <-ticker.C:
		}
	}
}

func currentOwnedProcessPID(manager *ProcessManager) (int, error) {
	if manager == nil {
		return 0, errors.New("ProcessManager is nil")
	}
	if manager.IsExternalMode() {
		return 0, errors.New("ProcessManager attached to external runtime")
	}
	status := manager.Status()
	if status.Status != "running" || status.PID <= 0 {
		return 0, fmt.Errorf("ProcessManager has no tracked running child: status=%q PID=%d", status.Status, status.PID)
	}
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", status.PID)); err != nil {
		return 0, fmt.Errorf("tracked child PID %d is not live: %w", status.PID, err)
	}
	return status.PID, nil
}

func (a *cancelDuringRealReady) Ready(ctx context.Context) error {
	a.mu.Lock()
	first := !a.first
	a.first = true
	a.mu.Unlock()
	if first {
		close(a.entered)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-a.release:
		}
	}
	return a.ApplyAdapter.Ready(ctx)
}

type barrierRealReady struct {
	ApplyAdapter
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (a *barrierRealReady) Ready(ctx context.Context) error {
	blocked := false
	a.once.Do(func() { blocked = true; close(a.entered) })
	if blocked {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-a.release:
		}
	}
	return a.ApplyAdapter.Ready(ctx)
}

func newRealAcceptanceFixture(t *testing.T, binary string) *realAcceptanceFixture {
	t.Helper()
	root := t.TempDir()
	userDir := filepath.Join(root, "userDir")
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		t.Fatalf("create isolated userDir: %v", err)
	}
	port := unusedLoopbackPort(t)
	// These overrides isolate ProcessManager from inherited live Node-RED
	// configuration. The fixture has no .env and no EnvService layer.
	t.Setenv("NODE_RED_PORT", port)
	t.Setenv("NODE_RED_USER_DIR", userDir)
	t.Setenv("NODE_RED_SETTINGS", filepath.Join(userDir, "settings.js"))
	config := NewIsolatedConfigService(userDir)
	baselineConfig, err := config.Get()
	if err != nil {
		t.Fatalf("load isolated baseline defaults: %v", err)
	}
	baselineConfig.UIPort = 1880
	baselineConfig.UIHost = "127.0.0.1"
	if err := config.Save(baselineConfig); err != nil {
		t.Fatalf("seed isolated baseline configuration: %v", err)
	}
	settings := filepath.Join(userDir, "settings.js")
	// #nosec G304,G703 -- test fixture: settings path is built under the fixture temp root, not user input.
	baseline, err := os.ReadFile(settings)
	if err != nil {
		t.Fatalf("read baseline settings: %v", err)
	}
	manager := NewProcessManager(binary, userDir)
	// A failed candidate must not trigger the manager's production
	// exponential auto-restart loop after the test has cleaned its temp root.
	// The apply transaction itself exercises the real restart/recovery path.
	manager.maxRestarts = 0
	if manager.IsExternalMode() {
		t.Fatal("new ProcessManager unexpectedly entered external mode")
	}
	if err := manager.Start(); err != nil {
		t.Fatalf("start real Node-RED child: %v", err)
	}
	f := &realAcceptanceFixture{root: root, userDir: userDir, settings: settings, port: port, binary: binary,
		config: config, manager: manager, baseline: baseline, baselineConfig: baselineConfig, recorder: &realAcceptanceAudit{}}
	t.Cleanup(func() {
		status := manager.Status()
		if manager.IsExternalMode() || status.PID <= 0 {
			t.Errorf("refusing cleanup without owned child identity: external=%v status=%+v", manager.IsExternalMode(), status)
			return
		}
		if err := manager.Stop(); err != nil {
			t.Errorf("stop owned Node-RED PID %d: %v", status.PID, err)
		}
		if after := manager.Status(); after.PID != 0 || after.Status == "running" {
			t.Errorf("owned Node-RED child remained after cleanup: %+v", after)
		} else {
			t.Logf("cleanup: stopped owned PID %d; manager status=%q PID=%d external=false", status.PID, after.Status, after.PID)
		}
	})
	selectedAdapter, err := SelectApplyAdapter(model.ConfigurationCapabilities{Adapter: "nodered-5"}, model.NodeRedEnvironment{
		Detected: true, Mode: model.InstallationModeNative, ManagedByNRCC: true, Running: true,
	}, ApplyAdapterDeps{ProcessManager: manager, ReadinessURLResolver: func() (string, error) {
		return fmt.Sprintf("http://127.0.0.1:%s/settings", port), nil
	}})
	if err != nil {
		t.Fatalf("select production managed-native adapter: %v", err)
	}
	f.adapter = &ownedRealApplyAdapter{ApplyAdapter: selectedAdapter, manager: manager, port: port}
	f.coordinator = f.runtimeCoordinator(f.adapter)
	f.assertRuntimeOwnedReady(t)
	t.Logf("isolated fixture root=%q userDir=%q initial owned PID=%d external=false", root, userDir, f.ownedPID(t))
	return f
}

func registerApplyWorkCleanup(t *testing.T, cancel context.CancelFunc, gate *releaseGate, done <-chan struct{}) {
	t.Helper()
	t.Cleanup(func() {
		if err := drainApplyWork(cancel, gate.Release, done, 60*time.Second); err != nil {
			t.Errorf("apply goroutine did not drain before owned-process cleanup: %v", err)
		}
	})
}

func (f *realAcceptanceFixture) runtimeCoordinator(adapter ApplyAdapter) *ApplyCoordinator {
	return NewRuntimeApplyCoordinator(NewApplyService(f.config, f.recorder.Log), func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
}

func (f *realAcceptanceFixture) request(structured *model.NodeRedConfig) ApplyRequest {
	doc, err := f.config.GetRawSettings()
	if err != nil {
		panic(fmt.Sprintf("read isolated settings revision: %v", err))
	}
	return ApplyRequest{Path: f.settings, Expected: doc.Revision, BackupDir: filepath.Join(f.userDir, "backups", "settings"),
		Actor: "real-acceptance", Capabilities: model.ConfigurationCapabilities{Adapter: "nodered-5"}, StructuredConfig: structured,
		Content: string(f.baseline)}
}

func (f *realAcceptanceFixture) ownedPID(t *testing.T) int {
	t.Helper()
	if f.manager.IsExternalMode() {
		t.Fatal("runtime authority unexpectedly attached to an external listener")
	}
	status := f.manager.Status()
	if status.Status != "running" || status.PID <= 0 {
		t.Fatalf("no tracked owned Node-RED process: %+v", status)
	}
	return status.PID
}

func (f *realAcceptanceFixture) assertRuntimeOwnedReady(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := f.adapter.Ready(ctx); err != nil {
		t.Fatalf("real Node-RED HTTP readiness failed: %v", err)
	}
	pid := f.ownedPID(t)
	if err := proveOwnedListener(pid, f.port); err != nil {
		t.Fatalf("ready endpoint is not owned by tracked PID %d: %v", pid, err)
	}
}

func (f *realAcceptanceFixture) assertStructuredBaseline(t *testing.T) {
	t.Helper()
	stored, err := f.config.Get()
	if err != nil || stored.UIPort != f.baselineConfig.UIPort || stored.Https != nil {
		t.Fatalf("structured baseline not restored: config=%+v err=%v", stored, err)
	}
}

func (f *realAcceptanceFixture) assertAudit(t *testing.T, action, stage string) {
	t.Helper()
	for _, event := range f.recorder.snapshot() {
		if event.Action == action && (stage == "" || event.MetaCopy["failure_stage"] == stage || event.MetaCopy["apply_stage"] == stage || event.MetaCopy["apply_rollback_stage"] == stage) {
			return
		}
	}
	t.Fatalf("audit event %q with stage %q absent: %+v", action, stage, f.recorder.snapshot())
}

func (f *realAcceptanceFixture) assertAuditMeta(t *testing.T, action, key, value string) {
	t.Helper()
	for _, event := range f.recorder.snapshot() {
		if event.Action == action && event.MetaCopy[key] == value {
			return
		}
	}
	t.Fatalf("audit event %q metadata %s=%q absent: %+v", action, key, value, f.recorder.snapshot())
}

func (f *realAcceptanceFixture) assertNoAudit(t *testing.T, action string) {
	t.Helper()
	for _, event := range f.recorder.snapshot() {
		if event.Action == action {
			t.Fatalf("unexpected audit %q: %+v", action, event)
		}
	}
}

func unusedLoopbackPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve ephemeral loopback port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release ephemeral port: %v", err)
	}
	return fmt.Sprint(port)
}
