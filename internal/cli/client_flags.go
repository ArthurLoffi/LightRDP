package cli

import (
	"crypto/rand"

	"github.com/spf13/cobra"

	"LightRDP/internal/client"
)

type clientFlags struct {
	serverURL string
	caFile    string
	id        string
}

func addClientFlags(cmd *cobra.Command, f *clientFlags) {
	flags := cmd.Flags()
	flags.StringVar(&f.serverURL, "server", "ws://localhost:8080/ws", "signaling server URL")
	flags.StringVar(&f.caFile, "ca", "", "PEM file with extra trusted certificates for wss:// (self-signed server certificate or private CA)")
	flags.StringVar(&f.id, "id", "", "your ID (generated if empty)")
}

func (f *clientFlags) connect(run ConnectFunc, mode, hostID string) error {
	selfID := f.id
	if selfID == "" {
		selfID = rand.Text()[:6]
	}

	tlsConfig, err := client.LoadTLSConfig(f.caFile)
	if err != nil {
		return &ExitError{Code: exitRuntime, Message: "error: --ca: " + err.Error(), Err: err}
	}

	if err := run(f.serverURL, tlsConfig, mode, selfID, hostID); err != nil {
		return newRuntimeError(err)
	}
	return nil
}
