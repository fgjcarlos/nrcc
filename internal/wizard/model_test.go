package wizard

import (
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestShouldRunWizardTTYGate(t *testing.T) {
	tests := []struct {
		name          string
		interactive   bool
		explicitFlags bool
		want          bool
	}{
		{name: "interactive no flags", interactive: true, want: true},
		{name: "interactive explicit flags", interactive: true, explicitFlags: true, want: false},
		{name: "non interactive no flags", interactive: false, want: false},
		{name: "non interactive explicit flags", interactive: false, explicitFlags: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRunWizard(model.HostStatus{Interactive: tt.interactive}, tt.explicitFlags)
			if got != tt.want {
				t.Fatalf("ShouldRunWizard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewStateNodeRedDefaults(t *testing.T) {
	missing := NewState(model.HostStatus{})
	if missing.NodeRedMode != model.NodeRedInstallModeNative {
		t.Fatalf("missing Node-RED default = %q, want native", missing.NodeRedMode)
	}

	detected := NewState(model.HostStatus{NodeRed: model.NodeRedEnvironment{Detected: true}})
	if detected.NodeRedMode != model.NodeRedInstallModeSkip {
		t.Fatalf("detected Node-RED default = %q, want skip", detected.NodeRedMode)
	}
}

func TestBuildPlanCarriesDetectedNodeRedAndHTTPSChoices(t *testing.T) {
	state := NewState(model.HostStatus{
		NodeRed: model.NodeRedEnvironment{
			Detected:     true,
			Executable:   "/usr/bin/node-red",
			UserDir:      "/home/nr/.node-red",
			SettingsPath: "/home/nr/.node-red/settings.js",
		},
	})
	state.WithPortless = true
	state.PortlessQuickSetup = true
	state.PortlessTrust = true

	plan := state.BuildPlan()
	if plan.NodeRedMode != model.NodeRedInstallModeSkip || !plan.NodeRedDetected {
		t.Fatalf("plan Node-RED = mode %q detected %v, want skip/detected", plan.NodeRedMode, plan.NodeRedDetected)
	}
	if plan.NodeRedCommand != "/usr/bin/node-red" || plan.NodeRedUserDir != "/home/nr/.node-red" || plan.NodeRedSettings != "/home/nr/.node-red/settings.js" {
		t.Fatalf("plan did not carry detected Node-RED details: %#v", plan)
	}
	if !plan.WithPortless || !plan.PortlessQuickSetup || !plan.PortlessTrust {
		t.Fatalf("plan did not carry HTTPS choices: %#v", plan)
	}
	if got := state.SuccessURL(); got != "https://nrcc.localhost" {
		t.Fatalf("SuccessURL() = %q, want https://nrcc.localhost", got)
	}
}
