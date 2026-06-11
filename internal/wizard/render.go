package wizard

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/fgjcarlos/nrcc/internal/model"
)

// Run renders the staged TUI and returns the shared install plan. The rendering
// stays thin; durable install decisions live in State/BuildPlan for table tests.
func Run(ctx context.Context, status model.HostStatus) (model.InstallPlan, State, error) {
	state := NewState(status)

	if !status.NodeRed.Detected {
		state.CurrentStep = StepNodeRedMode
		mode := string(state.NodeRedMode)
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("How should nrcc prepare Node-RED?").
				Description("Detected host first; choose how to make Node-RED available before nrcc starts.").
				Options(
					huh.NewOption("Install native Node-RED with npm", string(model.NodeRedInstallModeNative)),
					huh.NewOption("Install Docker and run Node-RED in Docker", string(model.NodeRedInstallModeDocker)),
					huh.NewOption("Skip Node-RED setup", string(model.NodeRedInstallModeSkip)),
				).
				Value(&mode),
		))
		if err := form.WithTheme(huh.ThemeCharm()).WithAccessible(true).RunWithContext(ctx); err != nil {
			return model.InstallPlan{}, state, err
		}
		state.NodeRedMode = model.NodeRedInstallMode(mode)
	} else {
		state.NodeRedMode = model.NodeRedInstallModeSkip
	}

	state.CurrentStep = StepHTTPS
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Configure local HTTPS with Portless?").
			Description("nrcc remains HTTP internally; Portless provides https://nrcc.localhost and https://node-red.localhost.").
			Affirmative("Yes, install and configure Portless").
			Negative("No, keep http://localhost:3001").
			Value(&state.WithPortless),
	))
	if err := form.WithTheme(huh.ThemeCharm()).WithAccessible(true).RunWithContext(ctx); err != nil {
		return model.InstallPlan{}, state, err
	}
	if state.WithPortless {
		state.PortlessQuickSetup = true
		form := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title("Trust the Portless local CA now?").
				Description("Choose yes to run the same trust setup as `nrcc portless setup-trust` during install.").
				Affirmative("Trust now").
				Negative("Show command later").
				Value(&state.PortlessTrust),
		))
		if err := form.WithTheme(huh.ThemeCharm()).WithAccessible(true).RunWithContext(ctx); err != nil {
			return model.InstallPlan{}, state, err
		}
	}

	state.CurrentStep = StepPublicAccess
	state.PublicAccessNotice = true
	if err := runNotice(ctx, "Public access", "Tailscale Funnel/public DNS is not changed by nrcc. Use Portless locally first, then expose deliberately after install."); err != nil {
		return model.InstallPlan{}, state, err
	}

	state.CurrentStep = StepSummary
	state.Confirmed = true
	confirm := true
	summary := []string{
		fmt.Sprintf("Node-RED mode: %s", state.NodeRedMode),
		fmt.Sprintf("HTTPS URL after install: %s", state.SuccessURL()),
	}
	if state.PortlessTrust {
		summary = append(summary, "Portless trust: run during install")
	}
	form = huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Review install plan").
			Description(strings.Join(summary, "\n")).
			Affirmative("Install nrcc").
			Negative("Abort").
			Value(&confirm),
	))
	if err := form.WithTheme(huh.ThemeCharm()).WithAccessible(true).RunWithContext(ctx); err != nil {
		return model.InstallPlan{}, state, err
	}
	if !confirm {
		return model.InstallPlan{}, state, fmt.Errorf("install aborted by user")
	}

	state.CurrentStep = StepExecute
	return state.BuildPlan(), state, nil
}

func runNotice(ctx context.Context, title, description string) error {
	m := noticeModel{title: title, description: description}
	_, err := tea.NewProgram(m, tea.WithoutRenderer(), tea.WithInput(nil)).Run()
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

type noticeModel struct {
	title       string
	description string
}

func (m noticeModel) Init() tea.Cmd { return tea.Quit }

func (m noticeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, tea.Quit }

func (m noticeModel) View() string { return m.title + "\n" + m.description + "\n" }
