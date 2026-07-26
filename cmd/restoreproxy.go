package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xgg-2/netcli/internal/sysproxy"
)

var restoreProxyCmd = &cobra.Command{
	Use:   "restore-proxy",
	Short: "Disable the Windows system proxy (safety net if auto-restore failed)",
	Long: `Directly sets ProxyEnable to 0 in the Windows system proxy registry and
broadcasts the change to all running applications. Run this at any time if
netcli exited without restoring the proxy, for example after a power loss,
system freeze, or forced termination.

On non-Windows platforms this command is not applicable and exits cleanly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sysproxy.DisableDirect(); err != nil {
			return fmt.Errorf("restore proxy: %w", err)
		}
		return nil
	},
}
