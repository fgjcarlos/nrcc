package cmd

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// setupCmd runs the interactive setup wizard
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Ejecuta el asistente de configuración",
	Long:  `Abre un asistente interactivo para configurar Node-RED inicial`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if stdin is a TTY
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			err := fmt.Errorf("setup requires an interactive terminal")
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: setup requires an interactive terminal\n")
			return err
		}

		// Call HostService.BootstrapCLI
		if err := svc.Host.BootstrapCLI(); err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		// If JSON mode requested, return success (though setup is interactive)
		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"setup_complete": true,
			})
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
