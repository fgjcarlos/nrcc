package cmd

import (
	"context"
	"fmt"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/ui"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the 'nrcc uninstall' command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall nrcc system service",
	Long: `Remove nrcc from your system.

This command:
- Stops and disables the systemd service
- Removes the service unit file and binary
- Optionally removes configuration and data directories

Requires root privileges (run with sudo).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// First check: require root
		if err := requireRoot(); err != nil {
			return err
		}

		ui.SectionHeader("Uninstallation")

		// Parse flags
		keepData, _ := cmd.Flags().GetBool("keep-data")
		purge, _ := cmd.Flags().GetBool("purge")

		// Create installer service
		layout := model.DefaultInstallLayout()
		installer := service.NewInstallerService(layout)

		// Prepare uninstall options
		opts := model.UninstallOpts{
			Layout:    layout,
			KeepData:  keepData,
			Purge:     purge,
			SkipPrompt: keepData || purge, // Skip prompt if flags are explicit
		}

		// Perform uninstallation
		spinner := ui.StartSpinner("Uninstalling nrcc…")
		ctx := context.Background()
		err := installer.Uninstall(ctx, opts)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Uninstallation failed: %v", err))
			return err
		}
		spinner.Success("Uninstallation completed")

		// Print confirmation
		pterm.Println()
		if keepData {
			pterm.Success.Println("✓ nrcc service removed (data preserved in /var/lib/nrcc)")
		} else {
			pterm.Success.Println("✓ nrcc fully removed (data directory deleted)")
		}

		return nil
	},
}

func init() {
	uninstallCmd.Flags().Bool(
		"keep-data",
		false,
		"Keep /var/lib/nrcc data after uninstall",
	)
	uninstallCmd.Flags().Bool(
		"purge",
		false,
		"Remove all data and config without prompting",
	)
	rootCmd.AddCommand(uninstallCmd)
}
