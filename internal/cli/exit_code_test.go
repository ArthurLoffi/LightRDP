package cli

import (
	"errors"
	"fmt"
	"testing"

	"LightRDP/internal/server"
)

func TestExitCode(t *testing.T) {
	cases := map[string]struct {
		err  error
		want int
	}{
		"no error":      {nil, 0},
		"plain error":   {errors.New("boom"), 1},
		"runtime error": {newRuntimeError(errors.New("boom")), 1},
		"usage error":   {&ExitError{Code: exitUsage, Message: "Error: bad", Err: errors.New("bad")}, 2},
		"wrapped usage": {fmt.Errorf("wrapped: %w", &ExitError{Code: exitUsage, Err: errors.New("bad")}), 2},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ExitCode(c.err); got != c.want {
				t.Fatalf("Expected exit code %d, got: %d", c.want, got)
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	cases := map[string]struct {
		err  error
		want string
	}{
		"no error":       {nil, ""},
		"plain error":    {errors.New("boom"), "error: boom"},
		"runtime error":  {newRuntimeError(errors.New("boom")), "error: boom"},
		"server stopped": {newRuntimeError(&server.StoppedError{Err: errors.New("tls: bad key")}), ""},
		"custom message": {&ExitError{Code: exitUsage, Message: "custom", Err: errors.New("bad")}, "custom"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ErrorMessage(c.err); got != c.want {
				t.Fatalf("Expected message %q, got: %q", c.want, got)
			}
		})
	}
}

func TestServerStoppedErrorExitsWithOne(t *testing.T) {
	err := newRuntimeError(&server.StoppedError{Err: errors.New("tls: bad key")})

	if code := ExitCode(err); code != exitRuntime {
		t.Fatalf("Expected exit code %d, got: %d", exitRuntime, code)
	}
}
