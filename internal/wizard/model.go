package wizard

import "github.com/fgjcarlos/nrcc/internal/model"

// Step names the guided install stages. It is intentionally UI-agnostic so the
// wizard decisions can be tested without a terminal or Bubble Tea runtime.
type Step string

const (
	StepPreScan      Step = "pre-scan"
	StepNodeRedMode  Step = "node-red-mode"
	StepHTTPS        Step = "https"
	StepPublicAccess Step = "public-access"
	StepSummary      Step = "summary"
	StepExecute      Step = "execute"
	StepSuccess      Step = "success"
)

// State is the pure decision model for the install wizard.
type State struct {
	Status             model.HostStatus
	NodeRedMode        model.NodeRedInstallMode
	WithPortless       bool
	PortlessQuickSetup bool
	PortlessTrust      bool
	PublicAccessNotice bool
	Confirmed          bool
	CurrentStep        Step
}

// NewState creates a detect-first wizard state from host status.
func NewState(status model.HostStatus) State {
	state := State{Status: status, CurrentStep: StepPreScan}
	if status.NodeRed.Detected {
		state.NodeRedMode = model.NodeRedInstallModeSkip
	} else {
		state.NodeRedMode = model.NodeRedInstallModeNative
	}
	return state
}

// NodeRedOptions returns the selectable Node-RED modes for the current host.
func (s State) NodeRedOptions() []model.NodeRedInstallMode {
	if s.Status.NodeRed.Detected {
		return []model.NodeRedInstallMode{model.NodeRedInstallModeSkip}
	}
	return []model.NodeRedInstallMode{
		model.NodeRedInstallModeNative,
		model.NodeRedInstallModeDocker,
		model.NodeRedInstallModeSkip,
	}
}

// BuildPlan converts wizard choices into the shared installer plan.
func (s State) BuildPlan() model.InstallPlan {
	plan := model.InstallPlan{
		SkipPrompt:         true,
		NodeRedMode:        s.NodeRedMode,
		NodeRedDetected:    s.Status.NodeRed.Detected,
		NodeRedCommand:     s.Status.NodeRed.Executable,
		NodeRedUserDir:     s.Status.NodeRed.UserDir,
		NodeRedSettings:    s.Status.NodeRed.SettingsPath,
		WithPortless:       s.WithPortless,
		PortlessQuickSetup: s.PortlessQuickSetup,
		PortlessTrust:      s.PortlessTrust,
	}
	if plan.NodeRedCommand == "" && s.Status.NodeRedBinary.Installed {
		plan.NodeRedCommand = s.Status.NodeRedBinary.Command
	}
	if plan.NodeRedSettings == "" {
		plan.NodeRedSettings = s.Status.Settings.Path
	}
	return plan
}

// SuccessURL is the URL shown at the end of the install flow.
func (s State) SuccessURL() string {
	if s.WithPortless && s.PortlessQuickSetup {
		return "https://nrcc.localhost"
	}
	return "http://localhost:3001"
}

// ShouldRunWizard enforces the TTY gate: only an interactive terminal with no
// explicit install flags may enter the TUI. Non-TTY and flag-driven installs use
// the existing non-interactive path.
func ShouldRunWizard(status model.HostStatus, explicitInstallFlags bool) bool {
	return status.Interactive && !explicitInstallFlags
}
