package cmd

import (
	"github.com/spf13/cobra"
)

// serviceCmd represents the 'nrcc service' command group
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage nrcc system service",
	Long: `Manage the nrcc systemd service.

Subcommands:
  service install   Install the systemd unit (service only, without creating directories)
  service remove    Remove the systemd unit (keeping data)
  service status    Show service status`,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}
