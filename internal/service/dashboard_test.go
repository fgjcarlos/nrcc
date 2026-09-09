package service

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDashboardFixture(t *testing.T, packageJSON, flowsJSON string) string {
	t.Helper()
	dir := t.TempDir()
	if packageJSON != "" {
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if flowsJSON != "" {
		if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(flowsJSON), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLegacyDashboardDetection(t *testing.T) {
	discovery := NewDashboardService(writeDashboardFixture(t, `{"dependencies":{"node-red-dashboard":"^3.6.6"}}`, `[]`)).Discover()
	if discovery.Legacy == nil || discovery.Legacy.Package.Name != legacyDashboard || discovery.Legacy.Package.Version != "^3.6.6" || discovery.Legacy.Path != "/ui" {
		t.Fatalf("legacy discovery = %#v", discovery.Legacy)
	}
	if discovery.Packages.State != discoveryAvailable || len(discovery.FlowFuse) != 0 {
		t.Fatalf("discovery = %#v", discovery)
	}
}

func TestFlowFuseUIBaseDiscovery(t *testing.T) {
	flows := `[
		{"id":"base-one","type":"ui-base","path":"/dashboard"},
		{"id":"base-two","type":"ui-base","path":"/operations"},
		{"id":"other","type":"inject"}
	]`
	discovery := NewDashboardService(writeDashboardFixture(t, `{"dependencies":{"@flowfuse/node-red-dashboard":"1.31.0"}}`, flows)).Discover()
	if len(discovery.FlowFuse) != 1 || discovery.FlowFuse[0].Name != flowFuseDashboard {
		t.Fatalf("FlowFuse packages = %#v", discovery.FlowFuse)
	}
	if len(discovery.UIBases) != 2 || discovery.UIBases[0].Path != "/dashboard" || discovery.UIBases[1].Path != "/operations" {
		t.Fatalf("ui-base paths = %#v", discovery.UIBases)
	}
}

func TestDashboardDiscoveryHonorsEmptyAndMalformedInputs(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		discovery := NewDashboardService(t.TempDir()).Discover()
		if discovery.Packages.State != discoveryAbsent || discovery.Flows.State != discoveryAbsent || discovery.Legacy != nil || len(discovery.UIBases) != 0 {
			t.Fatalf("discovery = %#v", discovery)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		discovery := NewDashboardService(writeDashboardFixture(t, `{`, `{`)).Discover()
		if discovery.Packages.State != discoveryMalformed || discovery.Flows.State != discoveryMalformed {
			t.Fatalf("discovery = %#v", discovery)
		}
	})
	t.Run("ui base without path", func(t *testing.T) {
		discovery := NewDashboardService(writeDashboardFixture(t, `{}`, `[{"id":"base","type":"ui-base"}]`)).Discover()
		if len(discovery.UIBases) != 1 || discovery.UIBases[0].State != discoveryUnknown || discovery.UIBases[0].Path != "" {
			t.Fatalf("ui-base paths = %#v", discovery.UIBases)
		}
	})
}
