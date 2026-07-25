package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var runPort int

var runCmd = &cobra.Command{
	Use:   "run [--port PORT] -- <command> [args...]",
	Short: "Run a command with its traffic routed through the local proxy",
	Long: `Launches the given command as a subprocess with HTTP_PROXY and HTTPS_PROXY
environment variables set to the local netcli proxy. This scopes traffic
inspection to that single process only, without changing system-wide proxy
settings.

The proxy must already be running in another terminal via 'netcli watch'.
Alternatively, combine this with watch to run both simultaneously.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("command is required after --")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyAddr := fmt.Sprintf("http://localhost:%d", runPort)

		child := exec.Command(args[0], args[1:]...)
		child.Env = append(os.Environ(),
			"HTTP_PROXY="+proxyAddr,
			"HTTPS_PROXY="+proxyAddr,
			"http_proxy="+proxyAddr,
			"https_proxy="+proxyAddr,
		)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		if err := child.Start(); err != nil {
			return fmt.Errorf("start command: %w", err)
		}

		done := make(chan error, 1)
		go func() {
			done <- child.Wait()
		}()

		select {
		case <-sig:
			_ = child.Process.Signal(syscall.SIGTERM)
			<-done
		case err := <-done:
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
		}
		return nil
	},
}

func init() {
	runCmd.Flags().IntVar(&runPort, "port", 8080, "proxy port to connect to")
}
