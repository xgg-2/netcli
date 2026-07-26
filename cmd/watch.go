package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/xgg-2/netcli/internal/cert"
	"github.com/xgg-2/netcli/internal/proxy"
	"github.com/xgg-2/netcli/internal/sysproxy"
	"github.com/xgg-2/netcli/internal/tui"
	"github.com/xgg-2/netcli/internal/types"
)

var (
	watchPort        int
	watchBind        string
	watchFilter      string
	watchSave        string
	watchSystemProxy bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Start the proxy and open the live traffic TUI",
	Long: `Starts a local HTTP/HTTPS proxy and opens an interactive terminal UI
showing all traffic routed through the proxy. Configure applications or
browsers to use localhost:<port> as their HTTP and HTTPS proxy.

By default the proxy binds to 127.0.0.1 (loopback only). Use --bind 0.0.0.0
to expose it on all network interfaces, for example when proxying traffic
from another device on the same network.

On Windows, pass --system-proxy to have netcli configure the system proxy
automatically on startup and restore it on exit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		exists, err := cert.CAExists()
		if err != nil {
			return fmt.Errorf("checking CA: %w", err)
		}
		if !exists {
			return fmt.Errorf("CA certificate not found; run 'netcli setup' first")
		}

		configDir, err := cert.ConfigDir()
		if err != nil {
			return fmt.Errorf("config dir: %w", err)
		}

		if watchSystemProxy {
			if err := sysproxy.Enable(watchPort); err != nil {
				return fmt.Errorf("system proxy: %w", err)
			}
			defer func() { _ = sysproxy.Disable() }()

			_ = sysproxy.RegisterCtrlHandler(func() { _ = sysproxy.Disable() })

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				_ = sysproxy.Disable()
				os.Exit(0)
			}()
		}

		entryChan := make(chan *types.RequestEntry, 256)

		addr := fmt.Sprintf("%s:%d", watchBind, watchPort)
		go func() {
			if err := proxy.Start(&proxy.Config{
				Addr:      addr,
				CaRootDir: configDir,
				Filter:    watchFilter,
				EntryChan: entryChan,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "proxy error: %v\n", err)
				os.Exit(1)
			}
		}()

		m, err := tui.NewModel(entryChan, watchSave)
		if err != nil {
			return fmt.Errorf("init TUI: %w", err)
		}

		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		return nil
	},
}

func init() {
	watchCmd.Flags().IntVar(&watchPort, "port", 8080, "proxy listen port")
	watchCmd.Flags().StringVar(&watchBind, "bind", "127.0.0.1", "IP address to bind the proxy listener to (use 0.0.0.0 to expose on all interfaces)")
	watchCmd.Flags().StringVar(&watchFilter, "filter", "", "show only requests matching this domain or path substring")
	watchCmd.Flags().StringVar(&watchSave, "save", "", "append captured requests to this JSONL file")
	watchCmd.Flags().BoolVar(&watchSystemProxy, "system-proxy", false, "configure the Windows system proxy automatically (Windows only)")
}
