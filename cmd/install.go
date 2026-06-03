package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/composedof2/nrcc/internal/ui"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// requireRoot checks if the current process has root privileges
func requireRoot() error {
	if os.Getuid() != 0 {
		return fmt.Errorf("this command requires root privileges. Run: sudo nrcc %s", os.Args[1])
	}
	return nil
}

// installCmd represents the 'nrcc install' command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install nrcc as a system service",
	Long: `Install nrcc as a managed system service using systemd.

This command:
- Creates system user and group (nrcc)
- Initializes /etc/nrcc and /var/lib/nrcc directories
- Generates a secure JWT secret
- Installs the systemd unit file
- Enables and starts the service
- Decides how to handle Node-RED before the service starts
- Optionally installs Portless and configures HTTPS .localhost aliases

If Node-RED is already installed, installation continues without prompting.
If Node-RED is missing, use --node-red skip|native|docker to run non-interactively,
or let nrcc ask in an interactive terminal.

Requires root privileges (run with sudo).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateInstallPortlessFlags(); err != nil {
			return err
		}

		// First check: require root
		if err := requireRoot(); err != nil {
			return err
		}

		ui.SectionHeader("Installation")

		// Create installer service
		layout := model.DefaultInstallLayout()
		installer := service.NewInstallerService(layout)

		// Perform installation
		opts := model.InstallOpts{
			Layout:             layout,
			SkipPrompt:         false,
			NodeRedMode:        model.NodeRedInstallMode(installNodeRedMode),
			WithPortless:       installWithPortless,
			PortlessQuickSetup: installPortlessQuickSetup,
			PortlessTrust:      installPortlessTrust,
		}

		spinner := ui.StartSpinner("Installing nrcc as system service…")
		ctx := context.Background()
		err := installer.Install(ctx, opts)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Installation failed: %v", err))
			return err
		}
		spinner.Success("Installation completed successfully")

		// Print success message
		pterm.Println()
		pterm.Success.Println("✓ nrcc installed and running")
		if installWithPortless && installPortlessQuickSetup {
			pterm.Printfln("🌐 Access nrcc at: https://nrcc.localhost")
			pterm.Printfln("🔐 Local HTTPS uses Portless trust. If your browser warns, run: sudo nrcc portless setup-trust")
		} else {
			pterm.Printfln("🌐 Access nrcc at: http://localhost:3001")
		}
		pterm.Printfln("📁 Data directory: %s", layout.DataDir)
		pterm.Printfln("⚙️  Config file: %s", layout.EnvFile)
		pterm.Println()
		pterm.Println("Next steps:")
		pterm.Printfln("  • View status: nrcc status")
		pterm.Printfln("  • View logs: journalctl -u nrcc -f")
		pterm.Printfln("  • Uninstall: sudo nrcc uninstall")

		return nil
	},
}

var installWithPortless bool
var installPortlessQuickSetup bool
var installPortlessTrust bool
var installNodeRedMode string

func validateInstallPortlessFlags() error {
	if !installWithPortless && (installPortlessQuickSetup || installPortlessTrust) {
		return fmt.Errorf("--portless-quick-setup and --portless-trust require --with-portless")
	}
	if installNodeRedMode == "" {
		return nil
	}
	mode, err := parseNodeRedInstallMode(installNodeRedMode)
	if err != nil {
		return err
	}
	installNodeRedMode = string(mode)
	return nil
}

func parseNodeRedInstallMode(value string) (model.NodeRedInstallMode, error) {
	normalized := model.NodeRedInstallMode(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case model.NodeRedInstallModeSkip, model.NodeRedInstallModeNative, model.NodeRedInstallModeDocker:
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid value for --node-red: %q (expected skip, native, or docker)", value)
	}
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&installWithPortless, "with-portless", false, "Install Portless CLI globally with npm")
	installCmd.Flags().StringVar(&installNodeRedMode, "node-red", "", "How to prepare Node-RED before starting the service: skip, native, or docker")
	installCmd.Flags().BoolVar(&installPortlessQuickSetup, "portless-quick-setup", false, "After --with-portless, configure default aliases: nrcc -> 3001 and node-red -> 1880")
	installCmd.Flags().BoolVar(&installPortlessTrust, "portless-trust", false, "After --with-portless, run portless trust to trust the local HTTPS CA")
}
