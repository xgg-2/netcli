package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "netcli",
	Short: "Terminal-based HTTP/HTTPS traffic inspector",
	Long: `netcli is a CLI tool for inspecting HTTP and HTTPS traffic from any
application or browser on the system. It operates as a MITM proxy and
presents captured traffic in an interactive terminal UI.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(certInfoCmd)
	rootCmd.AddCommand(restoreProxyCmd)
}
