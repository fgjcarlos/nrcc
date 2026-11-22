package server

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
	defer func() { _ = listener.Close() }()
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
