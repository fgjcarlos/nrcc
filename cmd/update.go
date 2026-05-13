package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// updateCmd is the parent command for update operations
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Gestiona actualizaciones",
	Long:  `Comprueba e instala actualizaciones de NRCC`,
}

// checkCmd checks for updates (stub)
var updateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Comprueba actualizaciones disponibles",
	Long:  `Verifica si hay nuevas versiones disponibles`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"implemented": false,
				"message":     "Comprobación de actualizaciones no implementada aún",
			})
		}

		fmt.Println("Comprobación de actualizaciones no implementada aún (próximamente)")
		return nil
	},
}

// applyCmd applies updates (stub)
var updateApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Aplica las actualizaciones disponibles",
	Long:  `Descarga e instala las actualizaciones disponibles`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"implemented": false,
				"message":     "Aplicación de actualizaciones no implementada aún",
			})
		}

		fmt.Println("Aplicación de actualizaciones no implementada aún (próximamente)")
		return nil
	},
}

func init() {
	// Register update subcommands
	updateCmd.AddCommand(updateCheckCmd)
	updateCmd.AddCommand(updateApplyCmd)

	// Register update group to root
	rootCmd.AddCommand(updateCmd)
}
