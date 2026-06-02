package wizard

import (
	"fmt"

	"github.com/pterm/pterm"
)

// RenderSummary is the current thin TTY renderer for the pure wizard model.
// The model deliberately stays independent from pterm/Bubble Tea/huh so the
// guided install can migrate renderers without changing install decisions.
func RenderSummary(m Model) {
	pterm.Info.Println("Install summary")
	pterm.Printfln("  Node-RED mode: %s", m.Plan.NodeRedMode)
	if m.Plan.NodeRedDetected {
		pterm.Printfln("  Adopt existing Node-RED: yes")
	}
	if m.Plan.NodeRedCommand != "" {
		pterm.Printfln("  Node-RED command: %s", m.Plan.NodeRedCommand)
	}
	if m.Plan.NodeRedSettings != "" {
		pterm.Printfln("  settings.js: %s", m.Plan.NodeRedSettings)
	}
	pterm.Printfln("  HTTPS: %s", yesNo(m.Plan.WithPortless))
	pterm.Printfln("  URL after install: %s", m.SuccessURL())
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// PublicAccessGuidance returns non-mutating guidance for Tailscale Funnel.
func PublicAccessGuidance() string {
	return fmt.Sprintf("Tailscale Funnel is informational only: verify tailscaled, MagicDNS, and HTTPS certs in the Tailscale admin console before exposing %s.", "nrcc")
}
