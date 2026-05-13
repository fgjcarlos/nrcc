package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// envCmd is the parent command for all environment variable operations
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Gestiona variables de entorno",
	Long:  `Gestiona las variables de entorno de Node-RED`,
}

// listCmd shows all environment variables
var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista las variables de entorno",
	Long:  `Muestra todas las variables de entorno configuradas`,
	RunE: func(cmd *cobra.Command, args []string) error {
		envVars, err := svc.Env.List()
		if err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		if len(envVars) == 0 {
			if isJSONMode(cmd) {
				return printJSON(map[string]interface{}{
					"vars":  []interface{}{},
					"count": 0,
				})
			}
			pterm.Info.Println("No hay variables de entorno configuradas")
			return nil
		}

		if isJSONMode(cmd) {
			// Build JSON response
			varsData := make([]map[string]interface{}, 0)
			for _, ev := range envVars {
				varData := map[string]interface{}{
					"key":       ev.Key,
					"value":     "***", // Always masked in JSON for security
					"encrypted": ev.Encrypted,
				}
				varsData = append(varsData, varData)
			}
			return printJSON(map[string]interface{}{
				"vars":  varsData,
				"count": len(varsData),
			})
		}

		// Human output: pterm table
		tableData := [][]string{
			{"Key", "Value", "Encrypted"},
		}
		for _, ev := range envVars {
			value := ev.Value
			if ev.Encrypted {
				value = "[encrypted]"
			}
			tableData = append(tableData, []string{
				ev.Key,
				value,
				fmt.Sprintf("%v", ev.Encrypted),
			})
		}

		table, _ := pterm.DefaultTable.WithHasHeader(true).WithData(tableData).Srender()
		pterm.Println(table)

		return nil
	},
}

// setCmd creates or updates an environment variable
var envSetCmd = &cobra.Command{
	Use:   "set <KEY=VALUE>",
	Short: "Configura una variable de entorno",
	Long:  `Crea o actualiza una variable de entorno. Formato: KEY=VALUE`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]

		// Parse KEY=VALUE format
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			errMsg := fmt.Errorf("invalid format. Use: KEY=VALUE")
			if isJSONMode(cmd) {
				return printError(errMsg)
			}
			fmt.Fprintf(os.Stderr, "Error: invalid format. Use: KEY=VALUE\n")
			return errMsg
		}

		key := parts[0]
		value := parts[1]

		// Validate key matches [A-Z_][A-Z0-9_]*
		keyRegex := regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
		if !keyRegex.MatchString(key) {
			errMsg := fmt.Errorf("invalid key name: %s. Keys must match [A-Z_][A-Z0-9_]*", key)
			if isJSONMode(cmd) {
				return printError(errMsg)
			}
			fmt.Fprintf(os.Stderr, "Error: invalid key name: %s. Keys must match [A-Z_][A-Z0-9_]*\n", key)
			return errMsg
		}

		// Check if key already exists
		existingVars, _ := svc.Env.List()
		action := "created"
		for _, ev := range existingVars {
			if ev.Key == key {
				action = "updated"
				break
			}
		}

		// Set environment variable
		if err := svc.Env.Set(key, value, "string", "", false); err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"key":    key,
				"action": action,
			})
		}

		pterm.Success.Printf("Variable configurada: %s\n", key)
		return nil
	},
}

// deleteCmd removes an environment variable
var envDeleteCmd = &cobra.Command{
	Use:   "delete <KEY>",
	Short: "Elimina una variable de entorno",
	Long:  `Elimina una variable de entorno existente`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// Check if key exists
		existingVars, _ := svc.Env.List()
		found := false
		for _, ev := range existingVars {
			if ev.Key == key {
				found = true
				break
			}
		}

		if !found {
			errMsg := fmt.Errorf("variable not found: %s", key)
			if isJSONMode(cmd) {
				return printError(errMsg)
			}
			fmt.Fprintf(os.Stderr, "Error: variable not found: %s\n", key)
			return errMsg
		}

		// Delete environment variable
		if err := svc.Env.Delete(key); err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"key":     key,
				"deleted": true,
			})
		}

		pterm.Success.Printf("Variable eliminada: %s\n", key)
		return nil
	},
}

// editCmd opens the .env file in the user's preferred editor
var envEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edita las variables de entorno",
	Long:  `Abre el archivo .env en tu editor preferido`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if stdin is a TTY
		if !isatty.IsTerminal(os.Stdin.Fd()) {
			errMsg := fmt.Errorf("env edit requires an interactive terminal")
			if isJSONMode(cmd) {
				return printError(errMsg)
			}
			fmt.Fprintf(os.Stderr, "Error: env edit requires an interactive terminal\n")
			return errMsg
		}

		envPath := os.ExpandEnv("$HOME/.nrcc/.env")
		if svc != nil && svc.DataDir != "" {
			envPath = svc.DataDir + "/.env"
		}

		// Create empty .env if missing
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			if err := os.WriteFile(envPath, []byte(""), 0644); err != nil {
				if isJSONMode(cmd) {
					return printError(fmt.Errorf("failed to create .env file: %w", err))
				}
				fmt.Fprintf(os.Stderr, "Error: failed to create .env file: %v\n", err)
				return err
			}
		}

		// Resolve editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			// Try nano first
			if _, err := exec.LookPath("nano"); err == nil {
				editor = "nano"
			} else if _, err := exec.LookPath("vi"); err == nil {
				editor = "vi"
			} else {
				errMsg := fmt.Errorf("no editor found (set EDITOR environment variable)")
				if isJSONMode(cmd) {
					return printError(errMsg)
				}
				fmt.Fprintf(os.Stderr, "Error: no editor found (set EDITOR environment variable)\n")
				return errMsg
			}
		}

		// Open editor
		fmt.Printf("Abriendo %s en %s...\n", envPath, editor)
		editCmd := exec.Command(editor, envPath)
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr

		if err := editCmd.Run(); err != nil {
			if isJSONMode(cmd) {
				return printError(fmt.Errorf("editor exited with error: %w", err))
			}
			return nil // Editor errors are not fatal
		}

		return nil
	},
}

func init() {
	// Register env subcommands
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envDeleteCmd)
	envCmd.AddCommand(envEditCmd)

	// Register env group to root
	rootCmd.AddCommand(envCmd)
}
