package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/ui"
	"github.com/mattn/go-isatty"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// nodeRedCmd is the parent command for all node-red operations
var nodeRedCmd = &cobra.Command{
	Use:   "node-red",
	Short: "Gestiona Node-RED",
	Long:  `Gestiona la instalación, inicio, parada y estado de Node-RED`,
}

// package-level variables for flags
var (
	nodeRedManage  bool
	nodeRedLines   int
	nodeRedMode    string
	nodeRedForce   bool
	nodeRedConfirm bool
)

// getOrCreatePM creates a ProcessManager for node-red commands
func getOrCreatePM() *service.ProcessManager {
	nodeRedCmd := os.Getenv("NODE_RED_CMD")
	if nodeRedCmd == "" {
		nodeRedCmd = "node-red"
	}
	return service.NewProcessManager(nodeRedCmd, svc.DataDir, svc.LogBuf)
}

// isManageEnabled checks if process management is enabled via flag or env var
func isManageEnabled(cmd *cobra.Command) bool {
	manage, _ := cmd.Flags().GetBool("manage")
	if manage || os.Getenv("NRCC_MANAGE_NODE_RED") == "true" {
		return true
	}
	return false
}

// isJSONMode checks if JSON output is requested
func isJSONMode(cmd *cobra.Command) bool {
	jsonFlag, _ := cmd.Flags().GetBool("json")
	return jsonFlag
}

// Start subcommand
var nrStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Inicia Node-RED",
	Long:  "Inicia el proceso de Node-RED mediante el gestor de procesos",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if manage mode is enabled
		if !isManageEnabled(cmd) {
			err := fmt.Errorf("process management not enabled. Set NRCC_MANAGE_NODE_RED=true or use --manage flag")
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		pm := getOrCreatePM()
		status := pm.Status()

		// Check if already running
		if status.Status == "running" {
			err := fmt.Errorf("Node-RED is already running (pid: %d)", status.PID)
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		// Start Node-RED
		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Iniciando Node-RED…")
			defer func() { _ = spinner.Stop() }()
			err := pm.Start()
			if err != nil {
				spinner.Fail(fmt.Sprintf("Error al iniciar Node-RED: %v", err))
				return err
			}
			spinner.Success("Node-RED iniciado")
		} else {
			err := pm.Start()
			if err != nil {
				return printError(err)
			}
			status = pm.Status()
			mode := "native"
			return printJSON(map[string]interface{}{
				"status": "started",
				"pid":    status.PID,
				"mode":   mode,
			})
		}

		return nil
	},
}

// Stop subcommand
var nrStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Detiene Node-RED",
	Long:  "Detiene el proceso de Node-RED de forma ordenada",
	RunE: func(cmd *cobra.Command, args []string) error {
		pm := getOrCreatePM()
		status := pm.Status()

		// Check if running
		if status.Status != "running" {
			err := fmt.Errorf("Node-RED is not running")
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		// Stop Node-RED
		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Deteniendo Node-RED…")
			defer func() { _ = spinner.Stop() }()
			err := pm.Stop()
			if err != nil {
				spinner.Fail(fmt.Sprintf("Error al detener Node-RED: %v", err))
				return err
			}
			spinner.Success("Node-RED detenido")
		} else {
			err := pm.Stop()
			if err != nil {
				return printError(err)
			}
			return printJSON(map[string]interface{}{
				"status": "stopped",
				"pid":    status.PID,
			})
		}

		return nil
	},
}

// Restart subcommand
var nrRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Reinicia Node-RED",
	Long:  "Detiene e inicia nuevamente el proceso de Node-RED",
	RunE: func(cmd *cobra.Command, args []string) error {
		pm := getOrCreatePM()
		status := pm.Status()

		// Try to stop (ignore if not running)
		if status.Status == "running" {
			if !isJSONMode(cmd) {
				spinner := ui.StartSpinner("Deteniendo Node-RED…")
				_ = pm.Stop()
				spinner.Success("Node-RED detenido")
			} else {
				_ = pm.Stop()
			}
		}

		// Start Node-RED
		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Iniciando Node-RED…")
					defer func() { _ = spinner.Stop() }()
					err := pm.Start()
			if err != nil {
				spinner.Fail(fmt.Sprintf("Error al iniciar Node-RED: %v", err))
				return err
			}
			spinner.Success("Node-RED reiniciado")
		} else {
			err := pm.Start()
			if err != nil {
				return printError(err)
			}
			status = pm.Status()
			return printJSON(map[string]interface{}{
				"status": "restarted",
				"pid":    status.PID,
				"mode":   "native",
			})
		}

		return nil
	},
}

// Status subcommand
var nrStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Muestra el estado de Node-RED",
	Long:  "Muestra información sobre el estado actual de Node-RED",
	RunE: func(cmd *cobra.Command, args []string) error {
		pm := getOrCreatePM()
		status := pm.Status()

		// If not running according to PM, check with pgrep
		isRunning := status.Status == "running"
		var externalPID int
		if !isRunning {
			// Try pgrep -x node-red
			pgrepCmd := exec.Command("pgrep", "-x", "node-red")
			if err := pgrepCmd.Run(); err == nil {
				isRunning = true
				// Try to get the PID from pgrep
				out, _ := exec.Command("pgrep", "-x", "node-red").Output()
				if str := strings.TrimSpace(string(out)); str != "" {
					if pid, err := strconv.Atoi(str); err == nil {
						externalPID = pid
					}
				}
			}
		}

		// Get version
		version := "unknown"
		if versionCmd := exec.Command("node-red", "--version"); versionCmd.Run() == nil {
			out, _ := exec.Command("node-red", "--version").Output()
			version = strings.TrimSpace(string(out))
		}

		statusStr := "stopped"
		if isRunning {
			statusStr = "running"
		}

		pid := status.PID
		if externalPID > 0 {
			pid = externalPID
		}

		uptime := "N/A"
		if isRunning && status.Uptime > 0 {
			uptime = fmt.Sprintf("%v", time.Duration(status.Uptime)*time.Second)
		}

		if isJSONMode(cmd) {
			data := map[string]interface{}{
				"status":  statusStr,
				"pid":     pid,
				"uptime":  uptime,
				"version": version,
				"mode":    "native",
			}
			if !isRunning {
				data["pid"] = nil
				data["uptime"] = nil
			}
			return printJSON(data)
		}

		// Human output: pterm table
		data := [][]string{
			{"Estado", statusStr},
			{"PID", fmt.Sprintf("%d", pid)},
			{"Tiempo en ejecución", uptime},
			{"Versión", version},
			{"Modo", "native"},
		}

		if !isRunning {
			data[1][1] = "N/A"
			data[2][1] = "N/A"
		}

		table, _ := pterm.DefaultTable.WithHasHeader(true).WithData(data).Srender()
		pterm.Println(table)

		return nil
	},
}

// Logs subcommand
var nrLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Muestra los registros de Node-RED",
	Long:  "Muestra las últimas líneas de registros del gestor de procesos",
	RunE: func(cmd *cobra.Command, args []string) error {
		lines, _ := cmd.Flags().GetInt("lines")
		if lines < 1 {
			err := fmt.Errorf("--lines must be at least 1")
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		// Check if manage mode is enabled
		if !isManageEnabled(cmd) {
			err := fmt.Errorf("process management not enabled; logs unavailable")
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		pm := getOrCreatePM()
		logLines := pm.GetLogs(lines)

		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"lines":          logLines,
				"count":          len(logLines),
				"total_buffered": len(logLines),
			})
		}

		// Human output: print lines with numbers
		if len(logLines) == 0 {
			pterm.Println(pterm.Sprintf("No hay registros disponibles"))
			return nil
		}

		for i, line := range logLines {
			pterm.Printf("%d: %s\n", i+1, line)
		}

		return nil
	},
}

// Version subcommand
var nrVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Muestra la versión de Node-RED",
	Long:  "Detecta y muestra la versión instalada de Node-RED",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Run node-red --version
		versionCmd := exec.Command("node-red", "--version")
		output, err := versionCmd.CombinedOutput()

		if err != nil {
			err = fmt.Errorf("Node-RED is not installed or not found on PATH")
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		version := strings.TrimSpace(string(output))

		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{
				"version": version,
			})
		}

		pterm.Printf("Node-RED version: %s\n", version)
		return nil
	},
}

// Detect subcommand
var nrDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detecta la instalación de Node-RED",
	Long:  "Detecta si Node-RED está instalado y en qué modo",
	RunE: func(cmd *cobra.Command, args []string) error {
		status := svc.Host.Detect()

		path := "N/A"
		if status.NodeRedBinary.Installed {
			path = status.NodeRedBinary.Command
		}

		mode := "not-found"
		if status.NodeRed.Detected {
			switch status.NodeRed.Mode {
			case "native":
				mode = "native"
			case "docker":
				mode = "docker"
			}
		}

		version := "N/A"
		if status.NodeRedBinary.Installed {
			version = status.NodeRedBinary.Version
		}

		npmPrefix := "N/A"
		if prefix := os.Getenv("npm_config_prefix"); prefix != "" {
			npmPrefix = prefix
		}

		if isJSONMode(cmd) {
			data := map[string]interface{}{
				"path":       path,
				"mode":       mode,
				"version":    version,
				"npm_prefix": npmPrefix,
			}
			return printJSON(data)
		}

		// Human output: pterm table
		data := [][]string{
			{"Detectado", fmt.Sprintf("%v", status.NodeRed.Detected)},
			{"Ruta", path},
			{"Modo", mode},
			{"Versión", version},
			{"Prefijo npm", npmPrefix},
		}

		table, _ := pterm.DefaultTable.WithHasHeader(true).WithData(data).Srender()
		pterm.Println(table)

		return nil
	},
}

// Install subcommand
var nrInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Instala Node-RED",
	Long:  "Instala Node-RED en el sistema (nativo o Docker)",
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, _ := cmd.Flags().GetString("mode")

		// If mode is empty and TTY, prompt
		if mode == "" {
			if isatty.IsTerminal(os.Stdin.Fd()) {
				options := []string{"native", "docker"}
				selected, _ := pterm.DefaultInteractiveSelect.
					WithOptions(options).
					WithDefaultText("Selecciona modo de instalación").
					Show()
				mode = selected
			} else {
				err := fmt.Errorf("--mode is required in non-interactive mode")
				if isJSONMode(cmd) {
					return printError(err)
				}
				return err
			}
		}

		// Check if already installed
		status := svc.Host.Detect()
		if status.NodeRedBinary.Installed {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				if isatty.IsTerminal(os.Stdin.Fd()) {
					result, _ := pterm.DefaultInteractiveConfirm.
						WithDefaultText(fmt.Sprintf("Node-RED v%s ya está instalado. ¿Continuar?", status.NodeRedBinary.Version)).
						Show()
					if !result {
						if isJSONMode(cmd) {
							return printJSON(map[string]interface{}{
								"installed": false,
								"message":   "Installation cancelled",
							})
						}
						pterm.Println(pterm.Sprintf("Instalación cancelada"))
						return nil
					}
				} else {
					err := fmt.Errorf("Node-RED already installed. Use --force to override")
					if isJSONMode(cmd) {
						return printError(err)
					}
					return err
				}
			}
		}

		// Perform installation
		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Instalando Node-RED…")
			defer func() { _ = spinner.Stop() }()

			var err error
			switch mode {
			case "native":
				err = svc.Host.InstallNodeRedNative()
			case "docker":
				err = fmt.Errorf("docker mode not yet implemented")
			default:
				err = fmt.Errorf("invalid mode: %s", mode)
			}

			if err != nil {
				spinner.Fail(fmt.Sprintf("Error: %v", err))
				return err
			}

			newStatus := svc.Host.Detect()
			spinner.Success(fmt.Sprintf("Node-RED %s instalado", newStatus.NodeRedBinary.Version))
		} else {
			var err error
			switch mode {
			case "native":
				err = svc.Host.InstallNodeRedNative()
			case "docker":
				err = fmt.Errorf("docker mode not yet implemented")
			default:
				err = fmt.Errorf("invalid mode: %s", mode)
			}

			if err != nil {
				return printError(err)
			}

			newStatus := svc.Host.Detect()
			return printJSON(map[string]interface{}{
				"installed": true,
				"version":   newStatus.NodeRedBinary.Version,
				"mode":      mode,
			})
		}

		return nil
	},
}

// Uninstall subcommand
var nrUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Desinstala Node-RED",
	Long:  "Desinstala Node-RED del sistema de forma permanente",
	RunE: func(cmd *cobra.Command, args []string) error {
		confirm, _ := cmd.Flags().GetBool("confirm")

		// Show warning banner
		if !isJSONMode(cmd) {
			pterm.DefaultBox.Println("⚠️  ADVERTENCIA: Esta operación eliminará Node-RED del sistema.\nNo se puede deshacer.")
		}

		// Check TTY for confirmation
		if !confirm {
			if isatty.IsTerminal(os.Stdin.Fd()) {
				result, _ := pterm.DefaultInteractiveConfirm.
					WithDefaultText("¿Confirmas la desinstalación? [y/N]").
					WithDefaultValue(false).
					Show()
				if !result {
					if isJSONMode(cmd) {
						return printJSON(map[string]interface{}{
							"uninstalled": false,
							"message":     "Uninstall cancelled",
						})
					}
					pterm.Println(pterm.Sprintf("Desinstalación cancelada"))
					return nil
				}
			} else {
				err := fmt.Errorf("--confirm flag is required for uninstall in non-interactive mode")
				if isJSONMode(cmd) {
					return printError(err)
				}
				return err
			}
		}

		// Check if running
		pm := getOrCreatePM()
		status := pm.Status()
		if status.Status == "running" {
			err := fmt.Errorf("Node-RED is currently running. Stop it first with: nrcc node-red stop")
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		// Get version before uninstall
		currentStatus := svc.Host.Detect()
		previousVersion := "unknown"
		if currentStatus.NodeRedBinary.Installed {
			previousVersion = currentStatus.NodeRedBinary.Version
		}

		// Uninstall
		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Desinstalando Node-RED…")
			defer func() { _ = spinner.Stop() }()
			err := svc.Host.UninstallNodeRedNative()
			if err != nil {
				spinner.Fail(fmt.Sprintf("Error: %v", err))
				return err
			}
			spinner.Success("Node-RED desinstalado correctamente")
		} else {
			err := svc.Host.UninstallNodeRedNative()
			if err != nil {
				return printError(err)
			}
			return printJSON(map[string]interface{}{
				"uninstalled":      true,
				"previous_version": previousVersion,
			})
		}

		return nil
	},
}

// Update subcommand
var nrUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Actualiza Node-RED",
	Long:  "Actualiza Node-RED a la última versión disponible",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Actualizando Node-RED…")
			defer func() { _ = spinner.Stop() }()
			oldVersion, newVersion, err := svc.Host.UpdateNodeRedNative()
			if err != nil {
				spinner.Fail(fmt.Sprintf("Error: %v", err))
				return err
			}

			if oldVersion == newVersion {
				spinner.Success(fmt.Sprintf("Node-RED ya está en la última versión (%s)", newVersion))
			} else {
				spinner.Success(fmt.Sprintf("Node-RED actualizado: %s → %s", oldVersion, newVersion))
			}
		} else {
			oldVersion, newVersion, err := svc.Host.UpdateNodeRedNative()
			if err != nil {
				return printError(err)
			}
			return printJSON(map[string]interface{}{
				"previous_version": oldVersion,
				"new_version":      newVersion,
				"updated":          oldVersion != newVersion,
			})
		}

		return nil
	},
}

func init() {
	// Register node-red group with root
	rootCmd.AddCommand(nodeRedCmd)

	// Add subcommands to node-red group
	nodeRedCmd.AddCommand(
		nrStartCmd,
		nrStopCmd,
		nrRestartCmd,
		nrStatusCmd,
		nrLogsCmd,
		nrVersionCmd,
		nrDetectCmd,
		nrInstallCmd,
		nrUninstallCmd,
		nrUpdateCmd,
	)

	// Flags for start
	nrStartCmd.Flags().BoolVar(&nodeRedManage, "manage", false, "Enable process management mode")

	// Flags for logs
	nrLogsCmd.Flags().IntVarP(&nodeRedLines, "lines", "n", 50, "Number of log lines to display")

	// Flags for install
	nrInstallCmd.Flags().StringVar(&nodeRedMode, "mode", "", "Installation mode (native or docker)")
	nrInstallCmd.Flags().BoolVar(&nodeRedForce, "force", false, "Skip already installed check")

	// Flags for uninstall
	nrUninstallCmd.Flags().BoolVar(&nodeRedConfirm, "confirm", false, "Confirm uninstallation")
}
