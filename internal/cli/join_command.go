package cli

import "github.com/spf13/cobra"

func newJoinCommand(connect ConnectFunc) *cobra.Command {
	var flags clientFlags

	cmd := &cobra.Command{
		Use:   "join HOST_ID",
		Short: "Connect to a host by its ID",
		Long: "Connect to the signaling server and send a connection request to the host\n" +
			"with the given ID. Compare the safety code with the person at the host\n" +
			"before confirming it with y.",
		Example: "  lightrdp join HOST\n" +
			"  lightrdp join HOST --server wss://192.168.0.10:8080/ws --ca server.crt",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flags.connect(connect, "join", args[0])
		},
	}

	addClientFlags(cmd, &flags)
	return cmd
}
