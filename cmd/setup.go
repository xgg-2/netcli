package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xgg-2/netcli/internal/cert"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Generate a local CA certificate for HTTPS interception",
	Long: `Generates a local CA certificate and private key stored in
~/.config/netcli/ if not already present. Run this once before using
the watch or run commands to intercept HTTPS traffic.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, created, err := cert.EnsureCA()
		if err != nil {
			return fmt.Errorf("failed to set up CA: %w", err)
		}

		if created {
			fmt.Println("CA certificate generated.")
		} else {
			fmt.Println("CA certificate already exists.")
		}

		fmt.Printf("Certificate location: %s\n\n", path)
		printTrustInstructions(path)
		return nil
	},
}

func printTrustInstructions(caPath string) {
	fmt.Println("Trust the CA certificate to inspect HTTPS traffic.")
	fmt.Println()

	fmt.Println("macOS:")
	fmt.Printf("  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", caPath)
	fmt.Println()

	fmt.Println("Linux (Debian/Ubuntu):")
	fmt.Printf("  sudo cp %s /usr/local/share/ca-certificates/netcli.crt\n", caPath)
	fmt.Println("  sudo update-ca-certificates")
	fmt.Println()

	fmt.Println("Linux (Fedora/RHEL):")
	fmt.Printf("  sudo cp %s /etc/pki/ca-trust/source/anchors/netcli.crt\n", caPath)
	fmt.Println("  sudo update-ca-trust extract")
	fmt.Println()

	fmt.Println("Windows (run in an elevated PowerShell session):")
	fmt.Printf("  Import-Certificate -FilePath \"%s\" -CertStoreLocation Cert:\\LocalMachine\\Root\n", caPath)
	fmt.Println()

	fmt.Println("Android:")
	fmt.Printf("  Copy %s to the device, then install via:\n", caPath)
	fmt.Println("  Settings > Security > Install a certificate > CA certificate")
	fmt.Println()

	fmt.Println("Firefox (any OS):")
	fmt.Println("  Firefox maintains its own trust store.")
	fmt.Printf("  Go to Preferences > Privacy & Security > Certificates > View Certificates > Import and select %s\n", caPath)
}
