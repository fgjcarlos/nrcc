package wizard

import (
	"testing"

	"github.com/composedof2/nrcc/internal/model"
)

func TestNewAdoptsExistingNativeNodeRed(t *testing.T) {
	status := model.HostStatus{
		NodeRed: model.NodeRedEnvironment{
			Detected:     true,
			Mode:         model.InstallationModeNative,
			Executable:   "/usr/local/bin/node-red",
			UserDir:      "/home/nr/.node-red",
			SettingsPath: "/home/nr/.node-red/settings.js",
		},
	}

	got := New(status)
	if got.Plan.NodeRedMode != model.NodeRedInstallModeSkip {
		t.Fatalf("NodeRedMode = %q, want skip for adopt-existing flow", got.Plan.NodeRedMode)
	}
	if !got.Plan.NodeRedDetected {
		t.Fatal("NodeRedDetected = false, want true")
	}
	if got.Plan.NodeRedCommand != "/usr/local/bin/node-red" {
		t.Fatalf("NodeRedCommand = %q", got.Plan.NodeRedCommand)
	}
	if got.Plan.NodeRedUserDir != "/home/nr/.node-red" || got.Plan.NodeRedSettings != "/home/nr/.node-red/settings.js" {
		t.Fatalf("adopted paths not preserved: %#v", got.Plan)
	}
}

func TestConfigureHTTPSKeepsSkipWorkingHTTPInstall(t *testing.T) {
	m := New(model.HostStatus{}).ChooseNodeRedMode(model.NodeRedInstallModeSkip).ConfigureHTTPS(false, true, true)
	if m.Plan.WithPortless || m.Plan.PortlessQuickSetup || m.Plan.PortlessTrust {
		t.Fatalf("disabled HTTPS should clear Portless options: %#v", m.Plan)
	}
	if got := m.SuccessURL(); got != "http://localhost:3001" {
		t.Fatalf("SuccessURL = %q", got)
	}
}

func TestConfigureHTTPSSuccessURL(t *testing.T) {
	m := New(model.HostStatus{}).ConfigureHTTPS(true, true, true)
	if got := m.SuccessURL(); got != "https://nrcc.localhost" {
		t.Fatalf("SuccessURL = %q", got)
	}
}

func TestAdvanceWizardStages(t *testing.T) {
	m := New(model.HostStatus{})
	want := []Step{StepNodeRedMode, StepHTTPS, StepPublicAccess, StepSummary, StepExecute, StepSuccess}
	for _, step := range want {
		m = m.Advance()
		if m.Step != step {
			t.Fatalf("Step = %q, want %q", m.Step, step)
		}
	}
}
