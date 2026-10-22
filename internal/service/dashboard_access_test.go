package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestDashboardApplySharesConfigRuntimeGuard(t *testing.T) {
	f := newApplyCoordinatorFixture(t)
	packageJSON, err := json.Marshal(map[string]any{"dependencies": map[string]string{legacyDashboard: "3.6.0"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, "package.json"), packageJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := &dashboardGuardAdapter{fakeAdapter: &fakeAdapter{name: "runtime"}, entered: make(chan struct{}), release: make(chan struct{})}
	coordinator := NewRuntimeApplyCoordinator(f.apply, func(ApplyRequest) (ApplyAdapter, error) { return adapter, nil })
	candidate := "module.exports = {\n  uiPort: 1991,\n}\n"
	finished := make(chan error, 1)
	go func() {
		_, err := coordinator.Apply(context.Background(), ApplyRequest{Path: f.settingsPath, Content: candidate, Expected: f.liveRevision(t), BackupDir: filepath.Join(f.dir, "backups")})
		finished <- err
	}()
	select {
	case <-adapter.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("config apply did not reach runtime restart")
	}
	before, err := os.ReadFile(f.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	access := NewDashboardAccessService(NewDashboardService(f.dir), f.svc, coordinator)
	live, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(live.Content, "module.exports = {") {
		t.Fatalf("dashboard fixture content is not patchable: %q", live.Content)
	}
	_, err = access.Apply(context.Background(), model.DashboardAccessPolicy{Target: dashboardTargetLegacy, Recipe: dashboardRecipeBasic, Username: "operator", Secret: "secret"}, "operator", nil)
	if !errors.Is(err, ErrApplyInFlight) {
		t.Fatalf("dashboard apply error=%v; want ErrApplyInFlight", err)
	}
	after, err := os.ReadFile(f.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("dashboard apply wrote settings while config runtime transaction held the guard")
	}
	close(adapter.release)
	if err := <-finished; err != nil {
		t.Fatalf("config apply failed: %v", err)
	}
}

type dashboardGuardAdapter struct {
	*fakeAdapter
	entered chan struct{}
	release chan struct{}
}

func (a *dashboardGuardAdapter) Restart(ctx context.Context) error {
	close(a.entered)
	select {
	case <-a.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestDashboardMiddlewarePairPreservesSource(t *testing.T) {
	source := `module.exports = {
  uiPort: 1880,
  httpMiddleware: function(req, res, next) { next(); },
  externalModules: { palette: { allowInstall: true } },
}
`
	policy := model.DashboardAccessPolicy{Target: dashboardTargetFlowFuse, Recipe: dashboardRecipeBasic, Username: "operator", Secret: "never-log-this"}
	got, err := renderDashboardPolicy(source, policy)
	if err != nil {
		t.Fatalf("renderDashboardPolicy: %v", err)
	}
	for _, want := range []string{"dashboard:", "middleware:", "ioMiddleware:", "httpMiddleware: function(req, res, next)", "externalModules: { palette: { allowInstall: true } }"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered source missing %q", want)
		}
	}
	if strings.Contains(RedactSettingsContent(got), "never-log-this") {
		t.Error("redacted dashboard source exposes the credential")
	}
}

func TestDashboardPolicyRejectsUnsafeOrAmbiguousDiscovery(t *testing.T) {
	valid := model.DashboardAccessPolicy{Target: dashboardTargetFlowFuse, Recipe: dashboardRecipeProxySSO, Secret: "proxy-secret"}
	available := model.DashboardDiscovery{
		Packages: model.DiscoverySource{State: discoveryAvailable}, Flows: model.DiscoverySource{State: discoveryAvailable},
		FlowFuse: []model.DashboardPackage{{Name: flowFuseDashboard}}, UIBases: []model.DashboardPath{{NodeID: "ui", Path: "/dashboard", State: discoveryAvailable}},
	}
	if !dashboardPolicySupported(available, valid) {
		t.Fatal("available FlowFuse discovery rejected")
	}
	available.Flows.State = discoveryMalformed
	if dashboardPolicySupported(available, valid) {
		t.Fatal("ambiguous FlowFuse discovery accepted")
	}
	for _, policy := range []model.DashboardAccessPolicy{
		{Target: dashboardTargetFlowFuse, Recipe: "javascript", Secret: "secret"},
		{Target: dashboardTargetFlowFuse, Recipe: dashboardRecipeBasic, Username: "user", Secret: "line\nbreak"},
		{Target: dashboardTargetFlowFuse, Recipe: dashboardRecipeBasic, Secret: "secret"},
	} {
		if !errors.Is(validateDashboardPolicy(policy), ErrUnsafeDashboardPolicy) {
			t.Errorf("unsafe policy %+v was accepted", policy)
		}
	}
}

func TestDashboardPolicyLegacyRendersBcryptAuth(t *testing.T) {
	got, err := renderDashboardPolicy("module.exports = {\n}\n", model.DashboardAccessPolicy{
		Target: dashboardTargetLegacy, Recipe: dashboardRecipeBasic, Username: "operator", Secret: "legacy-secret",
	})
	if err != nil {
		t.Fatalf("renderDashboardPolicy: %v", err)
	}
	if !strings.Contains(got, "httpNodeAuth") || strings.Contains(got, "legacy-secret") {
		t.Errorf("legacy recipe did not render bcrypt-only auth: %s", RedactSettingsContent(got))
	}
}

func TestDashboardPolicyDoesNotReplaceExistingFlowFuseSource(t *testing.T) {
	source := "module.exports = {\n  dashboard: { middleware: existingMiddleware },\n}\n"
	_, err := renderDashboardPolicy(source, model.DashboardAccessPolicy{Target: dashboardTargetFlowFuse, Recipe: dashboardRecipeProxySSO, Secret: "secret"})
	if !errors.Is(err, ErrDashboardPolicyUnavailable) {
		t.Fatalf("renderDashboardPolicy error = %v, want discovery/source refusal", err)
	}
}
