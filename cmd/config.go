package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// configCmd is the parent command for all config operations
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gestiona la configuración",
	Long:  `Gestiona los parámetros de configuración de Node-RED`,
}

// listCmd shows all configuration values
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista la configuración actual",
	Long:  `Muestra todos los parámetros de configuración en formato tabla`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := svc.Config.Get()
		if err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		if isJSONMode(cmd) {
			// Build JSON response
			settingsData := []map[string]interface{}{
				{"key": "flowFile", "value": cfg.FlowFile, "type": "string"},
				{"key": "credentialSecret", "value": "***", "type": "secret"},
				{"key": "httpAdminRoot", "value": cfg.HTTPAdminRoot, "type": "string"},
				{"key": "httpNodeRoot", "value": cfg.HTTPNodeRoot, "type": "string"},
				{"key": "port", "value": cfg.Port, "type": "number"},
				{"key": "uiPort", "value": cfg.UIPort, "type": "number"},
				{"key": "adminAuthEnabled", "value": cfg.AdminAuth != nil, "type": "boolean"},
			}
			return printJSON(map[string]interface{}{
				"settings": settingsData,
			})
		}

		// Human output: pterm table
		tableData := [][]string{
			{"Key", "Value", "Type"},
		}

		tableData = append(tableData, []string{
			"flowFile", cfg.FlowFile, "string",
		})
		tableData = append(tableData, []string{
			"credentialSecret", "***", "secret",
		})
		tableData = append(tableData, []string{
			"httpAdminRoot", cfg.HTTPAdminRoot, "string",
		})
		tableData = append(tableData, []string{
			"httpNodeRoot", cfg.HTTPNodeRoot, "string",
		})
		tableData = append(tableData, []string{
			"port", fmt.Sprintf("%d", cfg.Port), "number",
		})
		tableData = append(tableData, []string{
			"uiPort", fmt.Sprintf("%d", cfg.UIPort), "number",
		})
		tableData = append(tableData, []string{
			"adminAuthEnabled", fmt.Sprintf("%v", cfg.AdminAuth != nil), "boolean",
		})

		table, _ := pterm.DefaultTable.WithHasHeader(true).WithData(tableData).Srender()
		pterm.Println(table)

		return nil
	},
}

// getCmd retrieves a specific configuration value
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Obtiene un parámetro de configuración",
	Long:  `Muestra el valor de un parámetro específico de configuración`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		cfg, err := svc.Config.Get()
		if err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		var value interface{}
		var found bool

		// Map key to config field
		switch key {
		case "flowFile":
			value = cfg.FlowFile
			found = true
		case "credentialSecret":
			value = "***"
			found = true
		case "httpAdminRoot":
			value = cfg.HTTPAdminRoot
			found = true
		case "httpNodeRoot":
			value = cfg.HTTPNodeRoot
			found = true
		case "port":
			value = cfg.Port
			found = true
		case "uiPort":
			value = cfg.UIPort
			found = true
		case "adminAuthEnabled":
			value = cfg.AdminAuth != nil
			found = true
		}

		if !found {
			err := fmt.Errorf("key not found: %s", key)
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: key not found: %s\n", key)
			return err
		}

		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"key":   key,
				"value": value,
			})
		}

		fmt.Printf("%s: %v\n", key, value)
		return nil
	},
}

// setCmd updates a configuration value
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Actualiza un parámetro de configuración",
	Long:  `Modifica el valor de un parámetro de configuración`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		cfg, err := svc.Config.Get()
		if err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		var previousValue interface{}

		// Update config field
		switch key {
		case "flowFile":
			previousValue = cfg.FlowFile
			cfg.FlowFile = value
		case "httpAdminRoot":
			previousValue = cfg.HTTPAdminRoot
			cfg.HTTPAdminRoot = value
		case "httpNodeRoot":
			previousValue = cfg.HTTPNodeRoot
			cfg.HTTPNodeRoot = value
		case "port":
			var port int
			_, err := fmt.Sscanf(value, "%d", &port)
			if err != nil {
				errMsg := fmt.Errorf("invalid port value: %s", value)
				if isJSONMode(cmd) {
					return printError(errMsg)
				}
				fmt.Fprintf(os.Stderr, "Error: invalid port value: %s\n", value)
				return errMsg
			}
			previousValue = cfg.Port
			cfg.Port = port
		case "uiPort":
			var port int
			_, err := fmt.Sscanf(value, "%d", &port)
			if err != nil {
				errMsg := fmt.Errorf("invalid uiPort value: %s", value)
				if isJSONMode(cmd) {
					return printError(errMsg)
				}
				fmt.Fprintf(os.Stderr, "Error: invalid uiPort value: %s\n", value)
				return errMsg
			}
			previousValue = cfg.UIPort
			cfg.UIPort = port
		default:
			errMsg := fmt.Errorf("unknown configuration key: %s", key)
			if isJSONMode(cmd) {
				return printError(errMsg)
			}
			fmt.Fprintf(os.Stderr, "Error: unknown configuration key: %s\n", key)
			return errMsg
		}

		// Save config
		if err := svc.Config.Save(cfg); err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"key":            key,
				"value":          value,
				"previous_value": previousValue,
			})
		}

		pterm.Success.Printf("Configuración actualizada: %s = %v\n", key, value)
		pterm.Info.Println("Reinicia Node-RED para aplicar los cambios")
		return nil
	},
}

// backupCmd creates a timestamped backup of settings
var configBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Crea una copia de seguridad de la configuración",
	Long:  `Crea una copia de seguridad con timestamp de la configuración actual`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read current settings from file
		rawSettings, err := svc.Config.GetRawSettings()
		if err != nil {
			if isJSONMode(cmd) {
				return printError(fmt.Errorf("failed to read settings: %w", err))
			}
			fmt.Fprintf(os.Stderr, "Error: failed to read settings: %v\n", err)
			return err
		}

		if rawSettings.Path == "" {
			errMsg := fmt.Errorf("settings.js not found in %s. Run: nrcc setup", svc.DataDir)
			if isJSONMode(cmd) {
				return printError(errMsg)
			}
			fmt.Fprintf(os.Stderr, "Error: settings.js not found in %s. Run: nrcc setup\n", svc.DataDir)
			return errMsg
		}

		// Create backup directory
		backupDir := filepath.Join(svc.DataDir, "backups")
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			if isJSONMode(cmd) {
				return printError(fmt.Errorf("failed to create backup directory: %w", err))
			}
			fmt.Fprintf(os.Stderr, "Error: failed to create backup directory: %v\n", err)
			return err
		}

		// Create backup file
		timestamp := time.Now().Format("20060102_150405")
		backupPath := filepath.Join(backupDir, fmt.Sprintf("settings_%s.js", timestamp))

		if err := os.WriteFile(backupPath, []byte(rawSettings.Content), 0644); err != nil {
			if isJSONMode(cmd) {
				return printError(fmt.Errorf("failed to write backup: %w", err))
			}
			fmt.Fprintf(os.Stderr, "Error: failed to write backup: %v\n", err)
			return err
		}

		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"backup_path": backupPath,
				"timestamp":   timestamp,
			})
		}

		pterm.Success.Printf("Copia de seguridad creada: %s\n", backupPath)
		return nil
	},
}

func init() {
	// Register config subcommands
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configBackupCmd)

	// Register config group to root
	rootCmd.AddCommand(configCmd)
}
