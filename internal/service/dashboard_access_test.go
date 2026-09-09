package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

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
