package cli

import "github.com/spf13/cobra"

func newHostCommand(connect ConnectFunc) *cobra.Command {
	var flags clientFlags

	cmd := &cobra.Command{
		Use:   "host",
		Short: "Wait for connections from other computers",
		Long: "Connect to the signaling server and wait for connection requests.\n" +
			"Share your ID with the person who will join. Each request shows a safety\n" +
			"code, and nothing is shared before you accept it with y.",
		Example: "  lightrdp host\n" +
			"  lightrdp host --id HOST --server wss://192.168.0.10:8080/ws --ca server.crt",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return flags.connect(connect, "host", "")
		},
	}

	addClientFlags(cmd, &flags)
	return cmd
}
