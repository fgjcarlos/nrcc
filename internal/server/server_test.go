package server

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

func TestServerApplyResolverUsesLateInjectedProcessManagerForOwnedChild(t *testing.T) {
	host := service.NewIsolatedHostService(t.TempDir())
	var pm *service.ProcessManager
	resolver := serverApplyResolver(host.Detect, func() *service.ProcessManager { return pm }, nil, nil)
	request := service.ApplyRequest{Capabilities: model.ConfigurationCapabilities{Adapter: "nodered-5"}}
	if _, err := resolver(request); err == nil {
		t.Fatal("resolver accepted external native runtime without authority")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	readyServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	readyServer.Listener = listener
	readyServer.Start()
	defer readyServer.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	candidate := model.DefaultNodeRedConfig()
	candidate.Port = port
	request.StructuredConfig = &candidate
	request.Path = filepath.Join(t.TempDir(), "settings.js")
	if err := os.WriteFile(request.Path, []byte(fmt.Sprintf("module.exports = { uiPort: %d };\n", port)), 0o600); err != nil {
		t.Fatal(err)
	}
	pm = &service.ProcessManager{}
	adapter, err := resolver(request)
	if err != nil {
		t.Fatalf("resolver after late process-manager injection: %v", err)
	}
	if adapter.Name() != service.AdapterNameNativeBinary {
		t.Fatalf("adapter=%q; want owned native child", adapter.Name())
	}
	if err := adapter.Ready(t.Context()); err != nil {
		t.Fatalf("readiness did not use candidate's actual port %d: %v", port, err)
	}
}

func TestServerApplyResolverDoesNotTreatExternalProcessManagerAsOwnedChild(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	userDir := t.TempDir()
	settingsDir := t.TempDir()
	t.Setenv("NODE_RED_PORT", strconv.Itoa(port))
	t.Setenv("NODE_RED_USER_DIR", userDir)
	t.Setenv("NODE_RED_SETTINGS", filepath.Join(settingsDir, "settings.js"))
	reachable, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("test-owned listener is not reachable before ProcessManager.Start: %v", err)
	}
	if err := reachable.Close(); err != nil {
		t.Fatal(err)
	}
	pm := service.NewProcessManager(filepath.Join(t.TempDir(), "missing-node-red"), userDir)
	if err := pm.Start(); err != nil {
		t.Fatalf("attach ProcessManager to test-owned listener (nonexistent test binary prevents spawning Node-RED): %v", err)
	}
	defer func() {
		if err := pm.Stop(); err != nil {
			t.Errorf("detach external ProcessManager: %v", err)
		}
	}()
	if !pm.IsExternalMode() {
		t.Fatal("test ProcessManager did not enter external mode")
	}
	status := model.HostStatus{NodeRed: model.NodeRedEnvironment{Detected: true, Mode: model.InstallationModeDocker, ContainerID: "detected-container", ReadinessURL: "http://127.0.0.1:1880/settings"}}
	resolver := serverApplyResolver(func() model.HostStatus { return status }, func() *service.ProcessManager { return pm }, nil, service.NewDockerService())
	adapter, err := resolver(service.ApplyRequest{Content: "module.exports = { uiPort: 1880 };", Capabilities: model.ConfigurationCapabilities{Adapter: "nodered-5"}})
	if err != nil {
		t.Fatalf("resolver should use the detected supported Docker target: %v", err)
	}
	if adapter.Name() != service.AdapterNameDockerCustom {
		t.Fatalf("adapter=%q; want detected Docker target, not external ProcessManager", adapter.Name())
	}
}

func TestServerApplyResolverRejectsUnknownOwnedChildReadinessPort(t *testing.T) {
	pm := &service.ProcessManager{}
	resolver := serverApplyResolver(func() model.HostStatus { return model.HostStatus{} }, func() *service.ProcessManager { return pm }, nil, nil)
	_, err := resolver(service.ApplyRequest{Capabilities: model.ConfigurationCapabilities{Adapter: "nodered-5"}, Content: "module.exports = { uiPort: process.env.PORT || 1880 };"})
	if err == nil {
		t.Fatal("resolver guessed a readiness port for an expression-valued uiPort")
	}
}

func TestServerApplyResolverRejectsUnavailableAuthority(t *testing.T) {
	resolver := serverApplyResolver(service.NewIsolatedHostService(t.TempDir()).Detect, func() *service.ProcessManager { return nil }, nil, nil)
	_, err := resolver(service.ApplyRequest{Capabilities: model.ConfigurationCapabilities{Adapter: "nodered-5"}})
	if err == nil {
		t.Fatal("resolver accepted runtime without restart authority")
	}
}
