package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// cliServices holds all service instances for CLI commands
type cliServices struct {
	Host    *service.HostService
	Config  *service.ConfigService
	Env     *service.EnvService
	LogBuf  *service.LogBuffer
	DataDir string
}

// package-level service instance
var svc *cliServices
var mkdirAll = os.MkdirAll

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nrcc",
	Short: "Gestor profesional de Node-RED",
	Long: `NRCC es un gestor profesional para Node-RED que proporciona:
- Control total sobre instancias de Node-RED
- Gestión de configuración y entorno
- Monitoreo y logging integrado`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeServices(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add persistent flags
	var dataDir string
	rootCmd.PersistentFlags().StringVar(
		&dataDir,
		"data-dir",
		"",
		"Data directory (default: $DATA_DIR or ./data)",
	)
	rootCmd.PersistentFlags().Bool(
		"json",
		false,
		"Output JSON format",
	)
	rootCmd.PersistentFlags().Bool(
		"no-color",
		false,
		"Disable color output",
	)
}

func isReadOnlyDataDirCommand(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Name() == "doctor" {
			return true
		}
	}
	return false
}

// initializeServices resolves the dataDir and initializes all services
func initializeServices(cmd *cobra.Command) error {
	// Resolve dataDir: flag → DATA_DIR env → ./data
	dataDir, _ := cmd.Flags().GetString("data-dir")
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
	}

	// Resolve absolute path
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return fmt.Errorf("invalid data directory: %w", err)
	}

	// Create data directory if it doesn't exist. Read-only commands such as
	// `doctor` must not require write access to an existing production dataDir.
	if !isReadOnlyDataDirCommand(cmd) {
		if err := mkdirAll(absDataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	// Initialize services
	hostSvc := service.NewHostService(absDataDir)
	configSvc := service.NewConfigService(absDataDir)
	envSvc := service.NewEnvService(configSvc)
	logBuf := service.NewLogBuffer(1000)

	// Store in package-level variable
	svc = &cliServices{
		Host:    hostSvc,
		Config:  configSvc,
		Env:     envSvc,
		LogBuf:  logBuf,
		DataDir: absDataDir,
	}

	// Handle color preference (--no-color or NO_COLOR env var)
	noColor, _ := cmd.Flags().GetBool("no-color")
	if noColor || os.Getenv("NO_COLOR") != "" {
		pterm.DisableColor()
	}

	return nil
}

// printJSON outputs data in JSON format with standard envelope: {"ok":bool,"data":any,"error":string}
func printJSON(data interface{}) error {
	envelope := map[string]interface{}{
		"ok":    true,
		"data":  data,
		"error": nil,
	}
	jsonBytes, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}

// printError outputs an error in JSON format with envelope
func printError(err error) error {
	envelope := map[string]interface{}{
		"ok":    false,
		"data":  nil,
		"error": err.Error(),
	}
	jsonBytes, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}

// getServices returns the initialized services (for use by subcommands)
func getServices() *cliServices {
	return svc
}
