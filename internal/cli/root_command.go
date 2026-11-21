package cli

import (
	"crypto/tls"
	"errors"
	"io"

	"github.com/spf13/cobra"

	"LightRDP/internal/client"
	"LightRDP/internal/server"
)

type ListenFunc func(cfg server.ListenConfig) error

type ConnectFunc func(serverURL string, tlsConfig *tls.Config, mode, selfID, hostID string) error

type Runtime struct {
	Listen  ListenFunc
	Connect ConnectFunc
}

func DefaultRuntime() Runtime {
	return Runtime{Listen: server.Listen, Connect: client.Run}
}

func NewRootCommand(runtime Runtime) *cobra.Command {
	root := &cobra.Command{
		Use:   "lightrdp",
		Short: "Lightweight remote help desk tool",
		Long: "LightRDP connects two computers directly (P2P) for remote help.\n" +
			"Run a signaling server, start a host on the computer that receives help,\n" +
			"and join it from the other computer with the host ID.",
		Example: "  lightrdp server\n" +
			"  lightrdp host --id HOST\n" +
			"  lightrdp join HOST",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newServerCommand(runtime.Listen),
		newHostCommand(runtime.Connect),
		newJoinCommand(runtime.Connect),
		newVersionCommand(),
	)
	return root
}

func Execute(args []string, stdout, stderr io.Writer) error {
	return execute(NewRootCommand(DefaultRuntime()), args, stdout, stderr)
}

func execute(root *cobra.Command, args []string, stdout, stderr io.Writer) error {
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	cmd, err := root.ExecuteC()
	if err == nil {
		return nil
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return err
	}
	return newUsageError(cmd, err)
}
