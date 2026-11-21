package cli

import (
	"errors"
	"testing"

	"LightRDP/internal/server"
)

func TestServerDefaults(t *testing.T) {
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "server")

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	want := server.ListenConfig{Bind: "127.0.0.1", Port: 8080}
	if len(fake.listenCalls) != 1 || fake.listenCalls[0] != want {
		t.Fatalf("Expected one listen call with %+v, got: %+v", want, fake.listenCalls)
	}
}

func TestServerFlags(t *testing.T) {
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "server", "--bind", "192.168.0.10", "--port", "9090", "--debug", "--cert", "server.crt", "--key", "server.key")

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	want := server.ListenConfig{Bind: "192.168.0.10", Port: 9090, Debug: true, CertFile: "server.crt", KeyFile: "server.key"}
	if len(fake.listenCalls) != 1 || fake.listenCalls[0] != want {
		t.Fatalf("Expected one listen call with %+v, got: %+v", want, fake.listenCalls)
	}
}

func TestServerListenRules(t *testing.T) {
	const together = "--cert and --key must be used together"
	const required = "--cert and --key are required when --bind is not a loopback address"

	cases := map[string]struct {
		args []string
		want string
	}{
		"cert without key":         {[]string{"--cert", "server.crt"}, together},
		"key without cert":         {[]string{"--key", "server.key"}, together},
		"network bind without TLS": {[]string{"--bind", "192.168.0.10"}, required},
		"all interfaces no TLS":    {[]string{"--bind", "0.0.0.0"}, required},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			fake := &fakeRuntime{}
			res := runCommand(t, fake, append([]string{"server"}, c.args...)...)

			if code := ExitCode(res.err); code != exitUsage {
				t.Fatalf("Expected exit code %d, got: %d", exitUsage, code)
			}
			if message := ErrorMessage(res.err); message != c.want {
				t.Fatalf("Expected %q, got: %q", c.want, message)
			}
			if len(fake.listenCalls) != 0 {
				t.Fatalf("Expected no listen call, got: %+v", fake.listenCalls)
			}
		})
	}
}

func TestServerRejectsArguments(t *testing.T) {
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "server", "extra")

	expectUsageError(t, res, `unknown command "extra" for "lightrdp server"`)
	if len(fake.listenCalls) != 0 {
		t.Fatalf("Expected no listen call, got: %+v", fake.listenCalls)
	}
}

func TestServerStoppedPrintsNothingMore(t *testing.T) {
	fake := &fakeRuntime{listenErr: &server.StoppedError{Err: errors.New("tls: private key does not match public key")}}
	res := runCommand(t, fake, "server")

	if code := ExitCode(res.err); code != exitRuntime {
		t.Fatalf("Expected exit code %d, got: %d", exitRuntime, code)
	}
	if message := ErrorMessage(res.err); message != "" {
		t.Fatalf("Expected no extra message, got: %q", message)
	}
}
