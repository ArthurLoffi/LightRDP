package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "v0.0.1"

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "Print the LightRDP version",
		Long:    "Print the LightRDP version.",
		Example: "  lightrdp version",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "lightrdp "+Version)
			return err
		},
	}
}
