package cli

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var generatedIDFormat = regexp.MustCompile(`^[A-Z2-7]{6}$`)

func writeCertificateFile(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Expected key, got: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "LightRDP"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("192.168.0.10")},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Expected certificate, got: %v", err)
	}

	path := filepath.Join(t.TempDir(), "server.crt")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("Expected certificate written, got: %v", err)
	}
	return path
}

func expectVerifyingTLSConfig(t *testing.T, cfg *tls.Config) {
	t.Helper()

	if cfg == nil || cfg.InsecureSkipVerify || cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("Expected verifying TLS config, got: %+v", cfg)
	}
}

func TestHostDefaults(t *testing.T) {
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "host")

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	if len(fake.connectCalls) != 1 {
		t.Fatalf("Expected one connect call, got: %+v", fake.connectCalls)
	}
	call := fake.connectCalls[0]
	if call.serverURL != "ws://localhost:8080/ws" || call.mode != "host" || call.hostID != "" {
		t.Fatalf("Expected host defaults, got: %+v", call)
	}
	if !generatedIDFormat.MatchString(call.selfID) {
		t.Fatalf("Expected generated 6-character ID, got: %q", call.selfID)
	}
	if call.tlsConfig.RootCAs != nil {
		t.Fatalf("Expected system roots, got: %+v", call.tlsConfig.RootCAs)
	}
	expectVerifyingTLSConfig(t, call.tlsConfig)
}

func TestHostFlags(t *testing.T) {
	caFile := writeCertificateFile(t)
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "host", "--id", "HOST", "--server", "wss://192.168.0.10:8080/ws", "--ca", caFile)

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	call := fake.connectCalls[0]
	if call.serverURL != "wss://192.168.0.10:8080/ws" || call.mode != "host" || call.selfID != "HOST" || call.hostID != "" {
		t.Fatalf("Expected host flags, got: %+v", call)
	}
	if call.tlsConfig.RootCAs == nil {
		t.Fatal("Expected extra trusted certificates, got: nil RootCAs")
	}
	expectVerifyingTLSConfig(t, call.tlsConfig)
}

func TestHostRejectsArguments(t *testing.T) {
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "host", "HOST")

	expectUsageError(t, res, `unknown command "HOST" for "lightrdp host"`)
	if len(fake.connectCalls) != 0 {
		t.Fatalf("Expected no connect call, got: %+v", fake.connectCalls)
	}
}

func TestJoinDefaults(t *testing.T) {
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "join", "HOST")

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	if len(fake.connectCalls) != 1 {
		t.Fatalf("Expected one connect call, got: %+v", fake.connectCalls)
	}
	call := fake.connectCalls[0]
	if call.serverURL != "ws://localhost:8080/ws" || call.mode != "join" || call.hostID != "HOST" {
		t.Fatalf("Expected join defaults with host HOST, got: %+v", call)
	}
	if !generatedIDFormat.MatchString(call.selfID) {
		t.Fatalf("Expected generated 6-character ID, got: %q", call.selfID)
	}
	expectVerifyingTLSConfig(t, call.tlsConfig)
}

func TestJoinFlags(t *testing.T) {
	caFile := writeCertificateFile(t)
	fake := &fakeRuntime{}
	res := runCommand(t, fake, "join", "HOST", "--id", "HELPER", "--server", "wss://192.168.0.10:8080/ws", "--ca", caFile)

	if res.err != nil {
		t.Fatalf("Expected no error, got: %v", res.err)
	}
	call := fake.connectCalls[0]
	if call.serverURL != "wss://192.168.0.10:8080/ws" || call.mode != "join" || call.selfID != "HELPER" || call.hostID != "HOST" {
		t.Fatalf("Expected join flags, got: %+v", call)
	}
	if call.tlsConfig.RootCAs == nil {
		t.Fatal("Expected extra trusted certificates, got: nil RootCAs")
	}
	expectVerifyingTLSConfig(t, call.tlsConfig)
}

func TestJoinArgumentCount(t *testing.T) {
	cases := map[string][]string{
		"accepts 1 arg(s), received 0": {"join"},
		"accepts 1 arg(s), received 2": {"join", "HOST", "OTHER"},
	}
	for want, args := range cases {
		t.Run(want, func(t *testing.T) {
			fake := &fakeRuntime{}
			res := runCommand(t, fake, args...)

			expectUsageError(t, res, want)
			if !strings.Contains(ErrorMessage(res.err), "lightrdp join HOST_ID [flags]") {
				t.Fatalf("Expected join usage, got: %q", ErrorMessage(res.err))
			}
			if len(fake.connectCalls) != 0 {
				t.Fatalf("Expected no connect call, got: %+v", fake.connectCalls)
			}
		})
	}
}

func TestInvalidCAFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.crt")
	invalid := filepath.Join(t.TempDir(), "server.key")
	if err := os.WriteFile(invalid, []byte("not a certificate"), 0o600); err != nil {
		t.Fatalf("Expected file written, got: %v", err)
	}

	cases := map[string]struct {
		args []string
		want string
	}{
		"host missing file": {[]string{"host", "--ca", missing}, "error: --ca: open " + missing + ": no such file or directory"},
		"join invalid PEM":  {[]string{"join", "HOST", "--ca", invalid}, "error: --ca: no valid PEM certificate in " + invalid},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			fake := &fakeRuntime{}
			res := runCommand(t, fake, c.args...)

			if code := ExitCode(res.err); code != exitRuntime {
				t.Fatalf("Expected exit code %d, got: %d", exitRuntime, code)
			}
			if message := ErrorMessage(res.err); message != c.want {
				t.Fatalf("Expected %q, got: %q", c.want, message)
			}
			if len(fake.connectCalls) != 0 {
				t.Fatalf("Expected no connect call, got: %+v", fake.connectCalls)
			}
		})
	}
}

func TestNoFlagSkipsVerification(t *testing.T) {
	for _, command := range []string{"host", "join"} {
		cmd, _, err := NewRootCommand((&fakeRuntime{}).runtime()).Find([]string{command})
		if err != nil {
			t.Fatalf("Expected %s command, got: %v", command, err)
		}
		for _, name := range []string{"insecure", "skip-verify", "insecure-skip-verify", "no-verify"} {
			if cmd.Flags().Lookup(name) != nil {
				t.Fatalf("Expected no --%s flag on %s", name, command)
			}
		}
	}
}
