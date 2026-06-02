package wizard

import "github.com/composedof2/nrcc/internal/model"

// Step identifies a guided-install wizard stage. The model is intentionally
// pure so the TTY renderer can be swapped without changing install semantics.
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

// Model holds wizard state without terminal/UI dependencies.
type Model struct {
	Status model.HostStatus
	Plan   model.InstallPlan
	Step   Step
}

// New creates a wizard model from a pre-scan result. Existing Node-RED
// installations are adopted by default and missing Node-RED starts with a
// skip-safe plan until the user explicitly chooses native or docker.
func New(status model.HostStatus) Model {
	plan := model.InstallPlan{}
	if status.NodeRed.Detected {
		plan.NodeRedMode = model.NodeRedInstallModeSkip
		plan.NodeRedDetected = true
		plan.NodeRedUserDir = status.NodeRed.UserDir
		plan.NodeRedSettings = status.NodeRed.SettingsPath
		if status.NodeRed.Mode == model.InstallationModeNative {
			plan.NodeRedCommand = status.NodeRed.Executable
		}
	}
	return Model{Status: status, Plan: plan, Step: StepPreScan}
}

// ChooseNodeRedMode records the user's Node-RED path.
func (m Model) ChooseNodeRedMode(mode model.NodeRedInstallMode) Model {
	m.Plan.NodeRedMode = mode
	m.Step = StepHTTPS
	return m
}

// ConfigureHTTPS records optional in-flow HTTPS/Portless decisions.
func (m Model) ConfigureHTTPS(enabled, quickSetup, trust bool) Model {
	m.Plan.WithPortless = enabled
	m.Plan.PortlessQuickSetup = enabled && quickSetup
	m.Plan.PortlessTrust = enabled && trust
	m.Step = StepPublicAccess
	return m
}

// Advance moves to the next stable wizard stage.
func (m Model) Advance() Model {
	switch m.Step {
	case StepPreScan:
		m.Step = StepNodeRedMode
	case StepNodeRedMode:
		m.Step = StepHTTPS
	case StepHTTPS:
		m.Step = StepPublicAccess
	case StepPublicAccess:
		m.Step = StepSummary
	case StepSummary:
		m.Step = StepExecute
	case StepExecute:
		m.Step = StepSuccess
	}
	return m
}

// SuccessURL is the user-facing URL shown at the end of the wizard.
func (m Model) SuccessURL() string {
	if m.Plan.WithPortless && m.Plan.PortlessQuickSetup {
		return "https://nrcc.localhost"
	}
	return "http://localhost:3001"
}
