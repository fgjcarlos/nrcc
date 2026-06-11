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

// serviceRemoveCmd represents 'nrcc service remove' subcommand
var serviceRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove the systemd unit file",
	Long: `Remove the systemd unit file for nrcc service.

This stops and disables the service, then removes the unit file.
Data directories and configuration are preserved.

Requires root privileges.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot(); err != nil {
			return err
		}

		ui.SectionHeader("Service Unit Removal")

		layout := model.DefaultInstallLayout()
		installer := service.NewInstallerService(layout)

		opts := model.UninstallOpts{
			Layout:     layout,
			KeepData:   true, // Always keep data for service remove
			Purge:      false,
			SkipPrompt: true,
		}

		spinner := ui.StartSpinner("Removing service unit…")
		ctx := context.Background()
		err := installer.Uninstall(ctx, opts)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Failed: %v", err))
			return err
		}
		spinner.Success("Service unit removed")

		pterm.Println()
		pterm.Success.Println("✓ Systemd service unit removed")
		pterm.Printfln("  Data preserved in: %s", layout.DataDir)
		pterm.Printfln("  Config preserved in: %s", layout.ConfigDir)

		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceRemoveCmd)
}
