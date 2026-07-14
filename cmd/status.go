package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// statusCmd represents the 'nrcc status' command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show nrcc and service status",
	Long: `Display the current status of nrcc and its system service.

Shows:
- Systemd service state (active, inactive, failed)
- Node-RED subprocess state
- Installation directories
- Configured port`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonFlag, _ := cmd.Flags().GetBool("json")

		layout := model.DefaultInstallLayout()
		installer := service.NewInstallerService(layout)

		// Get installation status
		status := installer.Status()

		if jsonFlag {
			return printStatusJSON(status)
		}

		return printStatusHuman(status)
	},
}

// printStatusHuman prints human-readable status
func printStatusHuman(status model.InstallStatus) error {
	pterm.Println()
	pterm.Bold.Println("nrcc Status")
	pterm.Println()

	// Service state
	serviceStateIcon := "❌"
	if status.ServiceState == "active" {
		serviceStateIcon = "✓"
	}
	pterm.Printfln("%s Service State: %s", serviceStateIcon, status.ServiceState)

	// Data directory
	dataDirIcon := "✓"
	if !status.DataDirExists {
		dataDirIcon = "❌"
	}
	pterm.Printfln("%s Data Directory: %s", dataDirIcon, "/var/lib/nrcc")

	// Env file
	envFileIcon := "✓"
	if !status.EnvFileExists {
		envFileIcon = "❌"
	}
	pterm.Printfln("%s Config File: %s", envFileIcon, "/etc/nrcc/nrcc.env")

	// Unit file
	unitFileIcon := "✓"
	if !status.UnitFileExists {
		unitFileIcon = "❌"
	}
	pterm.Printfln("%s Unit File: %s", unitFileIcon, "/etc/systemd/system/nrcc.service")

	pterm.Println()

	// Recommendations. The first branch combines ServiceState and
	// DataDirExists so it cannot be a plain tagged switch on
	// ServiceState; the linter's ifElseChain hint does not apply
	// cleanly. Keeping the if-else chain is clearer than a switch
	// with a guard clause here.
	//nolint:gocritic
	if status.ServiceState == "active" && status.DataDirExists {
		pterm.Success.Println("✓ nrcc is properly installed and running")
		pterm.Printfln("  🌐 Access at: http://localhost:3001")
	} else if status.ServiceState == "inactive" {
		pterm.Warning.Println("⚠ nrcc is installed but not running")
		pterm.Println("  Start with: sudo systemctl start nrcc")
	} else if status.ServiceState == "unknown" {
		pterm.Warning.Println("⚠ systemd not available or nrcc not installed")
		pterm.Println("  Install with: sudo nrcc install")
	}

	return nil
}

// printStatusJSON prints JSON status
func printStatusJSON(status model.InstallStatus) error {
	response := map[string]interface{}{
		"ok": true,
		"data": map[string]interface{}{
			"serviceState":   status.ServiceState,
			"dataDirExists":  status.DataDirExists,
			"envFileExists":  status.EnvFileExists,
			"unitFileExists": status.UnitFileExists,
			"dataDir":        "/var/lib/nrcc",
			"configFile":     "/etc/nrcc/nrcc.env",
			"port":           "3001",
		},
		"error": nil,
	}

	output, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(output))
	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
