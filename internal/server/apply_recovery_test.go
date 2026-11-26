package server

import (
	"net/http"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

func TestExternalRecoveryResolverAllowsManagedComposeTarget(t *testing.T) {
	status := model.HostStatus{NodeRed: model.NodeRedEnvironment{
		Detected: true, Mode: model.InstallationModeDocker, ManagedByNRCC: true,
		ContainerID: "compose-container", ReadinessURL: "http://127.0.0.1:1880/settings",
	}}
	resolver := serverApplyResolver(func() model.HostStatus { return status }, func() *service.ProcessManager { return nil }, nil, service.NewDockerService())
	adapter, err := resolver(service.ApplyRequest{Path: "/data/settings.js", Content: "module.exports = { uiPort: 1880 };", Capabilities: model.ConfigurationCapabilities{Adapter: "nodered-5"}})
	if err != nil {
		t.Fatalf("Compose adapter resolution: %v", err)
	}
	if adapter == nil {
		t.Fatal("resolver returned nil adapter for Compose-managed Docker target")
	}
}

func TestExternalRecoveryAuditRequestCanHaveNoClientMetadata(t *testing.T) {
	req := &http.Request{}
	if got := req.Header.Get("User-Agent"); got != "" {
		t.Fatalf("Header.Get = %q", got)
	}
	if got := middleware.ExtractIP(req); got != "" {
		t.Fatalf("ExtractIP = %q", got)
	}
}
