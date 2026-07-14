package cmd

import (
	"fmt"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/ui"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	portlessExposeName   string
	portlessExposePort   int
	portlessExposeForce  bool
	portlessForce        bool
	portlessCleanAliases bool
	portlessUninstallYes bool
	portlessTrustYes     bool
)

var portlessCmd = &cobra.Command{
	Use:   "portless",
	Short: "Gestiona la integracion con Portless",
	Long: `Gestiona Portless para exponer nrcc o Node-RED con URLs nombradas.

Portless provee HTTPS local mediante dominios .localhost, modo LAN con mDNS
y exposicion por Tailscale/Funnel cuando el CLI de Tailscale esta configurado.

Getting Started:
  nrcc portless install
  nrcc portless quick-setup
  nrcc portless setup-trust

Subcommands:
  install       Instala Portless con npm
  status        Muestra version, comando y aliases registrados
  expose        Registra un alias Portless para un puerto local
  quick-setup   Registra nrcc/node-red y arranca el proxy HTTPS local
  setup-trust   Ejecuta portless trust para eliminar avisos HTTPS
  uninstall     Desinstala Portless y opcionalmente borra aliases`,
}

var portlessInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Instala Portless con npm",
	Long:  "Instala el CLI de Portless globalmente usando npm install -g portless.",
	RunE: func(cmd *cobra.Command, args []string) error {
		status := svc.Host.Detect()
		if status.Portless.Installed {
			if isJSONMode(cmd) {
				return printJSON(map[string]interface{}{
					"installed": true,
					"version":   status.Portless.Version,
					"command":   status.Portless.Command,
				})
			}
			pterm.Success.Printfln("Portless ya esta instalado: %s", status.Portless.Version)
			return nil
		}

		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Instalando Portless...")
			defer func() { _ = spinner.Stop() }()
			if err := svc.Host.InstallPortless(); err != nil {
				spinner.Fail(fmt.Sprintf("Error: %v", err))
				return err
			}
			newStatus := svc.Host.Detect()
			spinner.Success(fmt.Sprintf("Portless %s instalado", newStatus.Portless.Version))
			return nil
		}

		if err := svc.Host.InstallPortless(); err != nil {
			return printError(err)
		}
		newStatus := svc.Host.Detect()
		return printJSON(map[string]interface{}{
			"installed": true,
			"version":   newStatus.Portless.Version,
			"command":   newStatus.Portless.Command,
		})
	},
}

var portlessStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Muestra si Portless esta disponible",
	RunE: func(cmd *cobra.Command, args []string) error {
		status := svc.Host.Detect()
		aliases := []interface{}{}
		if status.Portless.Installed {
			if readAliases, err := svc.Host.ReadPortlessAliases(); err == nil {
				readAliases = svc.Host.CheckPortlessAliasReachability(readAliases)
				for _, alias := range readAliases {
					aliases = append(aliases, alias)
				}
			}
		}
		if isJSONMode(cmd) {
			data := map[string]interface{}{
				"installed": status.Portless.Installed,
				"version":   status.Portless.Version,
				"command":   status.Portless.Command,
				"aliases":   aliases,
			}
			return printJSON(data)
		}

		if !status.Portless.Installed {
			pterm.Warning.Println("Portless no esta instalado. Ejecuta: nrcc portless install")
			return nil
		}
		pterm.Success.Printfln("Portless disponible: %s", status.Portless.Version)
		pterm.Printfln("Comando: %s", status.Portless.Command)

		if readAliases, err := svc.Host.ReadPortlessAliases(); err == nil {
			readAliases = svc.Host.CheckPortlessAliasReachability(readAliases)
			if len(readAliases) == 0 {
				pterm.Println()
				pterm.Info.Println("No aliases registered. Run: nrcc portless quick-setup")
				return nil
			}
			pterm.Println()
			pterm.Info.Println("Aliases registrados:")
			for _, alias := range readAliases {
				state := "not reachable"
				if alias.Reachable {
					state = "reachable"
				}
				pterm.Printf("  %s -> %s  (%s, %s)\n", alias.Name, alias.LocalAddress, alias.URL, state)
			}
		}

		return nil
	},
}

var portlessExposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Registra un alias Portless para un puerto local",
	Long: `Registra un alias estatico de Portless para un servicio local existente.

Ejemplos:
  nrcc portless expose --name nrcc --port 3001
  nrcc portless expose --name node-red --port 1880

Despues de registrar el alias, Portless enruta https://<name>.localhost al puerto indicado.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(portlessExposeName)
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if portlessExposePort == 0 {
			return fmt.Errorf("--port is required")
		}

		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Registrando alias Portless...")
			defer func() { _ = spinner.Stop() }()
			if err := svc.Host.ExposePortlessAlias(name, portlessExposePort, portlessExposeForce); err != nil {
				spinner.Fail(fmt.Sprintf("Error: %v", err))
				return err
			}
			spinner.Success(fmt.Sprintf("Alias registrado: https://%s.localhost", name))
			return nil
		}

		if err := svc.Host.ExposePortlessAlias(name, portlessExposePort, portlessExposeForce); err != nil {
			return printError(err)
		}
		return printJSON(map[string]interface{}{
			"name":  name,
			"port":  portlessExposePort,
			"url":   fmt.Sprintf("https://%s.localhost", name),
			"force": portlessExposeForce,
		})
	},
}

var portlessQuickSetupCmd = &cobra.Command{
	Use:   "quick-setup",
	Short: "Registra aliases HTTPS locales para nrcc y Node-RED",
	Long: `Registra los aliases Portless recomendados en una sola operacion:
  nrcc     -> 3001  (https://nrcc.localhost)
  node-red -> 1880  (https://node-red.localhost)

Tambien arranca el proxy local de Portless, requerido para que las URLs
https://*.localhost respondan.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		status := svc.Host.Detect()
		if !status.Portless.Installed {
			err := fmt.Errorf("portless is not installed. Run: nrcc portless install")
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Registrando aliases Portless...")
			defer func() { _ = spinner.Stop() }()
			if err := svc.Host.QuickSetupPortless(portlessForce); err != nil {
				spinner.Fail(fmt.Sprintf("Error: %v", err))
				return err
			}
			spinner.Success("Aliases Portless configurados y proxy iniciado")
			pterm.Printfln("https://nrcc.localhost -> :3001")
			pterm.Printfln("https://node-red.localhost -> :1880")
			pterm.Info.Println("Run 'nrcc portless setup-trust' if you see certificate errors")
			return nil
		}

		if err := svc.Host.QuickSetupPortless(portlessForce); err != nil {
			return printError(err)
		}
		return printJSON(map[string]interface{}{
			"aliases": []map[string]interface{}{
				{"name": "nrcc", "port": 3001, "url": "https://nrcc.localhost"},
				{"name": "node-red", "port": 1880, "url": "https://node-red.localhost"},
			},
			"force": portlessForce,
		})
	},
}

var portlessUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Desinstala Portless",
	Long:  "Desinstala el paquete npm global portless y, si se solicita, borra ~/.portless/.",
	RunE: func(cmd *cobra.Command, args []string) error {
		status := svc.Host.Detect()
		if !status.Portless.Installed {
			if isJSONMode(cmd) {
				return printJSON(map[string]interface{}{"installed": false, "message": "Portless is not installed"})
			}
			pterm.Info.Println("Portless is not installed")
			return nil
		}

		if !portlessUninstallYes && !isJSONMode(cmd) {
			confirmed, _ := pterm.DefaultInteractiveConfirm.Show("Uninstall Portless npm package?")
			if !confirmed {
				pterm.Info.Println("Portless uninstall cancelled")
				return nil
			}
		}

		cleanAliases := false
		if portlessCleanAliases {
			if portlessUninstallYes || isJSONMode(cmd) {
				cleanAliases = false
			} else {
				confirmed, _ := pterm.DefaultInteractiveConfirm.Show("Remove ~/.portless/ directory? Aliases will be lost")
				cleanAliases = confirmed
			}
		}

		if !isJSONMode(cmd) {
			spinner := ui.StartSpinner("Desinstalando Portless...")
			defer func() { _ = spinner.Stop() }()
			if err := svc.Host.UninstallPortless(cleanAliases); err != nil {
				spinner.Fail(fmt.Sprintf("Error: %v", err))
				return err
			}
			spinner.Success("Portless desinstalado")
			if portlessCleanAliases && !cleanAliases {
				pterm.Info.Println("Config directory left in place: ~/.portless/")
			}
			return nil
		}

		if err := svc.Host.UninstallPortless(cleanAliases); err != nil {
			return printError(err)
		}
		return printJSON(map[string]interface{}{"installed": false, "cleanAliases": cleanAliases})
	},
}

var portlessSetupTrustCmd = &cobra.Command{
	Use:   "setup-trust",
	Short: "Confia en la CA local de Portless",
	Long:  "Ejecuta portless trust para instalar la CA local que elimina errores HTTPS para *.localhost.",
	RunE: func(cmd *cobra.Command, args []string) error {
		status := svc.Host.Detect()
		if !status.Portless.Installed {
			err := fmt.Errorf("portless is not installed. Run: nrcc portless install")
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}

		if !isJSONMode(cmd) {
			pterm.Info.Println("portless trust installs a local CA certificate to eliminate HTTPS certificate errors for *.localhost.")
			if !portlessTrustYes {
				confirmed, _ := pterm.DefaultInteractiveConfirm.Show("Proceed with portless trust?")
				if !confirmed {
					pterm.Info.Println("Portless trust setup cancelled")
					return nil
				}
			}
		}

		if err := svc.Host.SetupPortlessTrust(); err != nil {
			if isJSONMode(cmd) {
				return printError(err)
			}
			return err
		}
		if isJSONMode(cmd) {
			return printJSON(map[string]interface{}{"trusted": true})
		}
		pterm.Success.Println("You can now access https://nrcc.localhost and https://node-red.localhost without certificate warnings")
		pterm.Info.Println("Your browser may need a restart before it notices the new trust setting")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(portlessCmd)
	portlessCmd.AddCommand(portlessInstallCmd, portlessStatusCmd, portlessExposeCmd, portlessQuickSetupCmd, portlessUninstallCmd, portlessSetupTrustCmd)
	portlessExposeCmd.Flags().StringVar(&portlessExposeName, "name", "", "Portless alias name, e.g. nrcc or node-red")
	portlessExposeCmd.Flags().IntVar(&portlessExposePort, "port", 0, "Local port to expose through Portless")
	portlessExposeCmd.Flags().BoolVar(&portlessExposeForce, "force", false, "Overwrite an existing Portless alias")
	portlessQuickSetupCmd.Flags().BoolVar(&portlessForce, "force", false, "Overwrite existing Portless aliases")
	portlessUninstallCmd.Flags().BoolVar(&portlessUninstallYes, "yes", false, "Skip primary confirmation prompt")
	portlessUninstallCmd.Flags().BoolVar(&portlessCleanAliases, "clean-aliases", false, "Offer to delete ~/.portless/ aliases and trust data")
	portlessSetupTrustCmd.Flags().BoolVar(&portlessTrustYes, "yes", false, "Skip confirmation prompt")
}
