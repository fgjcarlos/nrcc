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

// serviceInstallCmd represents 'nrcc service install' subcommand
var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the systemd unit file only",
	Long: `Install only the systemd unit file for nrcc.

This is useful if you've already created directories and configuration separately.
It installs and enables the service without starting it.

Requires root privileges.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot(); err != nil {
			return err
		}

		ui.SectionHeader("Service Unit Installation")

		layout := model.DefaultInstallLayout()
		installer := service.NewInstallerService(layout)

		opts := model.InstallOpts{
			Layout:      layout,
			SkipPrompt:  true,
			NodeRedMode: model.NodeRedInstallModeSkip,
		}

		spinner := ui.StartSpinner("Installing service unit…")
		ctx := context.Background()
		err := installer.Install(ctx, opts)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Failed: %v", err))
			return err
		}
		spinner.Success("Service unit installed and enabled")

		pterm.Println()
		pterm.Success.Println("✓ Systemd service unit is now ready")
		pterm.Printfln("  Start with: sudo systemctl start nrcc")
		pterm.Printfln("  View logs: journalctl -u nrcc -f")

		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
}
