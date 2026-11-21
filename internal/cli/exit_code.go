package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"LightRDP/internal/server"
)

const (
	exitOK      = 0
	exitRuntime = 1
	exitUsage   = 2
)

type ExitError struct {
	Code    int
	Message string
	Err     error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

func ExitCode(err error) int {
	if err == nil {
		return exitOK
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return exitRuntime
}

func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Message
	}
	return "error: " + err.Error()
}

func newUsageError(cmd *cobra.Command, err error) *ExitError {
	message := "Error: " + err.Error() + "\n" + strings.TrimRight(cmd.UsageString(), "\n")
	return &ExitError{Code: exitUsage, Message: message, Err: err}
}

func newRuntimeError(err error) *ExitError {
	var stopped *server.StoppedError
	if errors.As(err, &stopped) {
		return &ExitError{Code: exitRuntime, Err: err}
	}
	return &ExitError{Code: exitRuntime, Message: "error: " + err.Error(), Err: err}
}
