package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// doctorCmd runs system diagnostics
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Ejecuta diagnóstico del sistema",
	Long:  `Realiza un diagnóstico completo del sistema para verificar dependencias y configuración`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Call HostService.PrintDoctorReport
		status := svc.Host.PrintDoctorReport()

		if isJSONMode(cmd) {
			// Build JSON response from check results
			checksData := make([]map[string]interface{}, 0)

			// Add checks based on status
			if status.NodeJS.Installed {
				checksData = append(checksData, map[string]interface{}{
					"name":    "node",
					"status":  "ok",
					"message": fmt.Sprintf("version: %s", status.NodeJS.Version),
				})
			} else {
				checksData = append(checksData, map[string]interface{}{
					"name":    "node",
					"status":  "error",
					"message": "not found",
				})
			}

			if status.NPM.Installed {
				checksData = append(checksData, map[string]interface{}{
					"name":    "npm",
					"status":  "ok",
					"message": fmt.Sprintf("version: %s", status.NPM.Version),
				})
			} else {
				checksData = append(checksData, map[string]interface{}{
					"name":    "npm",
					"status":  "error",
					"message": "not found",
				})
			}

			if status.NodeRedBinary.Installed {
				checksData = append(checksData, map[string]interface{}{
					"name":    "node-red",
					"status":  "ok",
					"message": fmt.Sprintf("version: %s", status.NodeRedBinary.Version),
				})
			} else {
				checksData = append(checksData, map[string]interface{}{
					"name":    "node-red",
					"status":  "warn",
					"message": "not found (can be installed)",
				})
			}

			if status.Portless.Installed {
				checksData = append(checksData, map[string]interface{}{
					"name":    "portless",
					"status":  "ok",
					"message": fmt.Sprintf("version: %s", status.Portless.Version),
				})
			} else {
				checksData = append(checksData, map[string]interface{}{
					"name":    "portless",
					"status":  "warn",
					"message": "not found (optional)",
				})
			}

			if status.Docker.Installed {
				checksData = append(checksData, map[string]interface{}{
					"name":    "docker",
					"status":  "ok",
					"message": fmt.Sprintf("version: %s", status.Docker.Version),
				})
			}

			// Determine overall status
			overallStatus := "ok"
			for _, check := range checksData {
				if checkStatus, ok := check["status"].(string); ok && checkStatus == "error" {
					overallStatus = "error"
					break
				}
			}

			return printJSON(map[string]interface{}{
				"checks":   checksData,
				"overall":  overallStatus,
				"platform": status.Platform,
			})
		}

		// Human output already printed by PrintDoctorReport
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
