package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xgg-2/netcli/internal/cert"
)

var certInfoCmd = &cobra.Command{
	Use:   "cert-info",
	Short: "Print CA certificate location and installation instructions",
	Long: `Prints the path to the local CA certificate and provides
step-by-step instructions for trusting it on each supported OS.
Also explains certificate pinning and why pinned applications cannot
be intercepted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := cert.CAPath()
		if err != nil {
			return fmt.Errorf("could not determine CA path: %w", err)
		}

		exists, err := cert.CAExists()
		if err != nil {
			return fmt.Errorf("could not check CA existence: %w", err)
		}

		if !exists {
			fmt.Println("CA certificate has not been generated yet.")
			fmt.Println("Run 'netcli setup' to generate it.")
			return nil
		}

		fmt.Printf("CA certificate: %s\n\n", path)
		printTrustInstructions(path)
		printPinningNote()
		return nil
	},
}

func printPinningNote() {
	fmt.Println("Certificate Pinning")
	fmt.Println("-------------------")
	fmt.Println("Some applications (mobile apps, some desktop clients) use certificate")
	fmt.Println("pinning, which means they accept only a specific certificate or public")
	fmt.Println("key rather than trusting the system CA store. These applications will")
	fmt.Println("refuse connections proxied through netcli and may show a TLS error.")
	fmt.Println()
	fmt.Println("There is no workaround for certificate pinning without modifying the")
	fmt.Println("application binary. This is intentional security behavior.")
}
