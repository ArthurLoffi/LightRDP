package main

import (
	"fmt"
	"os"

	"LightRDP/internal/cli"
)

func main() {
	err := cli.Execute(os.Args[1:], os.Stdout, os.Stderr)
	if message := cli.ErrorMessage(err); message != "" {
		fmt.Fprintln(os.Stderr, message)
	}
	os.Exit(cli.ExitCode(err))
}
