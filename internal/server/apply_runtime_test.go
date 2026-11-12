package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

func TestManagedReadinessResolvesRestoredSettingsWithPinnedAdapter(t *testing.T) {
	oldTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer oldTarget.Close()
	candidateTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer candidateTarget.Close()
	oldPort := serverPort(t, oldTarget.Listener.Addr())
	candidatePort := serverPort(t, candidateTarget.Listener.Addr())
	path := filepath.Join(t.TempDir(), "settings.js")
	oldContent := fmt.Sprintf("module.exports = {\n  uiPort: %d,\n}\n", oldPort)
	candidateContent := fmt.Sprintf("module.exports = {\n  uiPort: %d,\n}\n", candidatePort)
	if err := os.WriteFile(path, []byte(oldContent), 0o600); err != nil {
		t.Fatal(err)
	}
	pm := &service.ProcessManager{}
	resolverCalls := 0
	resolver := serverApplyResolver(func() model.HostStatus { return model.HostStatus{} }, func() *service.ProcessManager { return pm }, nil, nil)
	adapter, err := resolver(service.ApplyRequest{Path: path, Content: candidateContent, Capabilities: model.ConfigurationCapabilities{Adapter: "nodered-5"}})
	if err != nil {
		t.Fatal(err)
	}
	resolverCalls++
	if err := os.WriteFile(path, []byte(candidateContent), 0o600); err != nil {
		t.Fatal(err)
	}
	pinned := &noOpRestartAdapter{ApplyAdapter: adapter}
	if err := pinned.Restart(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := pinned.Ready(context.Background()); err == nil {
		t.Fatal("candidate endpoint should fail readiness")
	}
	if err := os.WriteFile(path, []byte(oldContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := pinned.Restart(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := pinned.Ready(context.Background()); err != nil {
		t.Fatalf("restored readiness did not follow active settings port %d: %v", oldPort, err)
	}
	if resolverCalls != 1 {
		t.Fatalf("authority resolution count=%d, want one pinned adapter", resolverCalls)
	}
	if pinned.restartCalls != 2 {
		t.Fatalf("restart calls=%d, want candidate and recovery restart", pinned.restartCalls)
	}
}

type noOpRestartAdapter struct {
	service.ApplyAdapter
	restartCalls int
}

func (a *noOpRestartAdapter) Restart(context.Context) error { a.restartCalls++; return nil }

func TestHardenedNodeRedFixtureProcess(t *testing.T) {
	if os.Getenv("NRCC_TEST_HARDENED_NODE_RED_CHILD") != "1" {
		return
	}
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, 4, 0, 0); errno != 0 {
		fmt.Fprintf(os.Stderr, "PR_SET_DUMPABLE failed: %v\n", errno)
		os.Exit(2)
	}
	port := os.Getenv("NODE_RED_PORT")
	server := &http.Server{
		Addr: "127.0.0.1:" + port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/settings" {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "hardened HTTP fixture failed: %v\n", err)
		os.Exit(3)
	}
}

func TestManagedReadinessUsesLaunchPortWithoutProcAccess(t *testing.T) {
	osPort := reservePort(t)
	dotenvPort := reservePort(t)
	dataDir := t.TempDir()
	settingsPath := filepath.Join(dataDir, "settings.js")
	settings := fmt.Sprintf("module.exports = { uiPort: %d };\n", dotenvPort)
	if err := os.WriteFile(settingsPath, []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, ".env"), []byte(fmt.Sprintf("NODE_RED_PORT=%d\n", dotenvPort)), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODE_RED_PORT", strconv.Itoa(osPort))
	t.Setenv("NODE_RED_USER_DIR", dataDir)
	t.Setenv("NODE_RED_SETTINGS", settingsPath)
	t.Setenv("NRCC_TEST_HARDENED_NODE_RED_CHILD", "1")
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test binary: %v", err)
	}
	t.Setenv("NRCC_TEST_HARDENED_NODE_RED_BINARY", binary)
	launcher := filepath.Join(t.TempDir(), "hardened-node-red")
	launcherScript := "#!/bin/sh\nexec \"$NRCC_TEST_HARDENED_NODE_RED_BINARY\" -test.run '^TestHardenedNodeRedFixtureProcess$'\n"
	// #nosec G306 -- test fixture: the launcher must be executable by the shell to spawn the hardened child.
	if err := os.WriteFile(launcher, []byte(launcherScript), 0o700); err != nil {
		t.Fatalf("write launcher fixture: %v", err)
	}
	pm := service.NewProcessManager(launcher, dataDir)
	if err := pm.Start(); err != nil {
		t.Fatalf("start hardened Node-RED child: %v", err)
	}
	t.Cleanup(func() {
		if err := pm.Stop(context.Background()); err != nil {
			t.Errorf("stop hardened Node-RED child: %v", err)
		}
	})

	pid := pm.Status().PID
	if pid <= 0 {
		t.Fatalf("managed child has no PID: %+v", pm.Status())
	}
	readyURL := fmt.Sprintf("http://127.0.0.1:%d/settings", dotenvPort)
	client := &http.Client{Timeout: 250 * time.Millisecond}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		response, requestErr := client.Get(readyURL) // #nosec G107 -- test URL uses reserved loopback and a fixture port.
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		select {
		case <-ticker.C:
		case <-timeout.C:
			t.Fatalf("hardened Node-RED fixture did not become ready at %s: %v", readyURL, requestErr)
		}
	}
	if _, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid)); !errors.Is(err, syscall.EACCES) && !errors.Is(err, syscall.EPERM) {
		t.Fatalf("expected hardened child environment to be unreadable, got error %v", err)
	}

	for _, source := range []string{
		fmt.Sprintf("module.exports = { uiPort: %d };\n", dotenvPort),
		fmt.Sprintf("module.exports = { uiPort: %d };\n", osPort),
		"module.exports = { editorTheme: {} };\n",
	} {
		if err := os.WriteFile(settingsPath, []byte(source), 0o600); err != nil {
			t.Fatalf("write readiness source: %v", err)
		}
		readinessURL, err := managedSettingsReadinessURL(settingsPath, pm)
		if err != nil {
			t.Fatalf("managedSettingsReadinessURL: %v", err)
		}
		if readinessURL != readyURL {
			t.Fatalf("managedSettingsReadinessURL = %q, want launch port URL %q", readinessURL, readyURL)
		}
	}
}

func reservePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release loopback port: %v", err)
	}
	return port
}

func TestSettingsReadinessURLDoesNotGuessWhenUiPortIsOmitted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.js")
	if err := os.WriteFile(path, []byte("module.exports = { editorTheme: {} };\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := settingsReadinessURL(path); !errors.Is(err, errSettingsPortMissing) {
		t.Fatalf("settingsReadinessURL error = %v, want missing static uiPort without runtime authority", err)
	}
}

func TestSettingsReadinessURLDoesNotGuessDynamicOrAmbiguousPorts(t *testing.T) {
	for _, content := range []string{
		"module.exports = { uiPort: process.env.PORT || 1880 };",
		"module.exports = { uiPort: 1880, uiPort: 1881 };",
		"module.exports = { ['ui' + 'Port']: 1881 };",
		"module.exports = { [process.env.PORT_KEY]: 1881 };",
	} {
		t.Run(content, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.js")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("NODE_RED_PORT", "")
			if _, err := settingsReadinessURL(path); err == nil {
				t.Fatal("settingsReadinessURL accepted dynamic or ambiguous uiPort")
			}
		})
	}
}

func TestApplyCandidatePortRequiresUnambiguousStaticLiteral(t *testing.T) {
	for _, input := range []string{
		"// uiPort: 1880\nmodule.exports = { uiPort: process.env.PORT || 1880 };",
		`module.exports = { note: "uiPort: 1880" };`,
		"module.exports = { uiPort: 1880, ...other, uiPort: 1881 };",
		"module.exports = { uiPort: 1880 + offset };",
		"module.exports = { uiPort: 1880 }; module.exports = { uiPort: 1881 };",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := applyCandidatePort(service.ApplyRequest{Content: input}); err == nil {
				t.Fatalf("accepted ambiguous/dynamic candidate %q", input)
			}
		})
	}
	cfg := model.DefaultNodeRedConfig()
	cfg.Port = 65536
	if _, err := applyCandidatePort(service.ApplyRequest{StructuredConfig: &cfg}); err == nil {
		t.Fatal("accepted structured port above 65535")
	}
}

func TestApplyCandidatePortReportsSyntaxErrorForInvalidRawSettings(t *testing.T) {
	// The raw apply API must report a syntax error to the operator; the
	// acceptance suite asserts that phrase on the validation failure.
	_, err := applyCandidatePort(service.ApplyRequest{Content: "module.exports = { uiPort: 1880, broken: };"})
	if err == nil {
		t.Fatal("accepted invalid raw settings")
	}
	if !strings.Contains(err.Error(), "syntax error") {
		t.Fatalf("error %q does not report a syntax error", err)
	}
}

func serverPort(t *testing.T, address net.Addr) int {
	t.Helper()
	_, raw, err := net.SplitHostPort(address.String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatal(err)
	}
	return port
}
