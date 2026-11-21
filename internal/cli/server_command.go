package cli

import (
	"github.com/spf13/cobra"

	"LightRDP/internal/server"
)

func newServerCommand(listen ListenFunc) *cobra.Command {
	var cfg server.ListenConfig

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the signaling server",
		Long: "Start the signaling server that hosts and joiners use to find each other.\n" +
			"It listens on 127.0.0.1 by default. A network address in --bind requires\n" +
			"--cert and --key, so the server only accepts encrypted (wss://) connections.",
		Example: "  lightrdp server\n" +
			"  lightrdp server --bind 192.168.0.10 --cert server.crt --key server.key",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := server.ValidateListenConfig(cfg.Bind, cfg.CertFile, cfg.KeyFile); err != nil {
				return &ExitError{Code: exitUsage, Message: err.Error(), Err: err}
			}
			if err := listen(cfg); err != nil {
				return newRuntimeError(err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&cfg.Bind, "bind", "127.0.0.1", "listen address (use 0.0.0.0 to listen on all interfaces; non-loopback addresses require --cert and --key)")
	flags.IntVar(&cfg.Port, "port", 8080, "port")
	flags.BoolVar(&cfg.Debug, "debug", false, "verbose logs")
	flags.StringVar(&cfg.CertFile, "cert", "", "TLS certificate file (PEM); serves wss:// together with --key")
	flags.StringVar(&cfg.KeyFile, "key", "", "TLS private key file (PEM); serves wss:// together with --cert")
	return cmd
}
