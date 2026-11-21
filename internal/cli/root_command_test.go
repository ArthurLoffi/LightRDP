package cli

import (
	"bytes"
	"crypto/tls"
	"errors"
	"strings"
	"testing"

	"LightRDP/internal/server"
)

type connectCall struct {
	serverURL string
	tlsConfig *tls.Config
	mode      string
	selfID    string
	hostID    string
}

type fakeRuntime struct {
	listenCalls  []server.ListenConfig
	connectCalls []connectCall
	listenErr    error
	connectErr   error
}

func (f *fakeRuntime) runtime() Runtime {
	return Runtime{
		Listen: func(cfg server.ListenConfig) error {
			f.listenCalls = append(f.listenCalls, cfg)
			return f.listenErr
		},
		Connect: func(serverURL string, tlsConfig *tls.Config, mode, selfID, hostID string) error {
			f.connectCalls = append(f.connectCalls, connectCall{serverURL, tlsConfig, mode, selfID, hostID})
			return f.connectErr
		},
	}
}

type result struct {
	err    error
	stdout string
	stderr string
}

func runCommand(t *testing.T, fake *fakeRuntime, args ...string) result {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := execute(NewRootCommand(fake.runtime()), append([]string{}, args...), &stdout, &stderr)
	return result{err: err, stdout: stdout.String(), stderr: stderr.String()}
}

func expectUsageError(t *testing.T, res result, want string) {
	t.Helper()

	if code := ExitCode(res.err); code != exitUsage {
		t.Fatalf("Expected exit code %d, got: %d (err: %v)", exitUsage, code, res.err)
	}
	message := ErrorMessage(res.err)
	if !strings.HasPrefix(message, "Error: "+want) {
		t.Fatalf("Expected message starting with %q, got: %q", "Error: "+want, message)
	}
	if !strings.Contains(message, "Usage:") {
		t.Fatalf("Expected usage in message, got: %q", message)
	}
}

func TestRootWithoutCommandPrintsHelp(t *testing.T) {
	res := runCommand(t, &fakeRuntime{})

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	for _, want := range []string{"Usage:", "server", "host", "join", "version", "completion"} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("Expected root help to contain %q, got: %q", want, res.stdout)
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	res := runCommand(t, &fakeRuntime{}, "hots")

	expectUsageError(t, res, `unknown command "hots" for "lightrdp"`)
}

func TestUnknownFlag(t *testing.T) {
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "server", "--address", "127.0.0.1")

	expectUsageError(t, res, "unknown flag: --address")
	if len(fake.listenCalls) != 0 {
		t.Fatalf("Expected no listen call, got: %+v", fake.listenCalls)
	}
}

func TestOldSingleDashFlags(t *testing.T) {
	cases := map[string][]string{
		"unknown shorthand flag: 'm' in -mode": {"host", "-mode", "host"},
		"unknown shorthand flag: 'b' in -bind": {"server", "-bind", "127.0.0.1"},
		"unknown shorthand flag: 'p' in -peer": {"join", "-peer", "HOST"},
	}
	for want, args := range cases {
		t.Run(want, func(t *testing.T) {
			fake := &fakeRuntime{}
			res := runCommand(t, fake, args...)

			expectUsageError(t, res, want)
			if len(fake.listenCalls) != 0 || len(fake.connectCalls) != 0 {
				t.Fatalf("Expected no runtime call, got: %+v %+v", fake.listenCalls, fake.connectCalls)
			}
		})
	}
}

func TestHelpOfEveryCommand(t *testing.T) {
	cases := map[string][]string{
		"":        {"lightrdp [command]", "Examples:", "lightrdp join HOST"},
		"server":  {"lightrdp server [flags]", "Examples:", "--bind string", `(default "127.0.0.1")`, "--port int", "(default 8080)", "--cert string", "--key string", "--debug"},
		"host":    {"lightrdp host [flags]", "Examples:", "--server string", `(default "ws://localhost:8080/ws")`, "--ca string", "--id string"},
		"join":    {"lightrdp join HOST_ID [flags]", "Examples:", "lightrdp join HOST", "--server string", "--ca string", "--id string"},
		"version": {"lightrdp version [flags]", "Examples:"},
	}
	for command, wants := range cases {
		for _, helpFlag := range []string{"--help", "-h"} {
			t.Run(command+" "+helpFlag, func(t *testing.T) {
				args := []string{helpFlag}
				if command != "" {
					args = []string{command, helpFlag}
				}
				fake := &fakeRuntime{}
				res := runCommand(t, fake, args...)

				if res.err != nil {
					t.Fatalf("Expected no error, got: %v", res.err)
				}
				for _, want := range wants {
					if !strings.Contains(res.stdout, want) {
						t.Fatalf("Expected help to contain %q, got: %q", want, res.stdout)
					}
				}
				if len(fake.listenCalls) != 0 || len(fake.connectCalls) != 0 {
					t.Fatalf("Expected no runtime call, got: %+v %+v", fake.listenCalls, fake.connectCalls)
				}
			})
		}
	}
}

func TestCompletionBash(t *testing.T) {
	res := runCommand(t, &fakeRuntime{}, "completion", "bash")

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	if !strings.Contains(res.stdout, "lightrdp") || len(res.stdout) < 100 {
		t.Fatalf("Expected bash completion script, got: %q", res.stdout)
	}
}

func TestRuntimeErrorExitsWithOne(t *testing.T) {
	fake := &fakeRuntime{connectErr: errors.New("server: target not found: HOST")}
	res := runCommand(t, fake, "join", "HOST")

	if code := ExitCode(res.err); code != exitRuntime {
		t.Fatalf("Expected exit code %d, got: %d", exitRuntime, code)
	}
	if message := ErrorMessage(res.err); message != "error: server: target not found: HOST" {
		t.Fatalf("Expected 'error: server: target not found: HOST', got: %q", message)
	}
}
