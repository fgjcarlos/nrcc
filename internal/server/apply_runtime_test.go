package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

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
