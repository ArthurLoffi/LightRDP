package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"LightRDP/internal/transport"
)

func writeTestCertificate(t *testing.T) (certFile, keyFile string, cert *x509.Certificate) {
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
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:         true,

		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Expected certificate, got: %v", err)
	}
	cert, err = x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("Expected parsed certificate, got: %v", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("Expected key encoded, got: %v", err)
	}

	dir := t.TempDir()
	certFile = filepath.Join(dir, "server.crt")
	keyFile = filepath.Join(dir, "server.key")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("Expected certificate written, got: %v", err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatalf("Expected key written, got: %v", err)
	}
	return certFile, keyFile, cert
}

func TestValidateListenConfig(t *testing.T) {
	const together = "-cert and -key must be used together"
	const required = "-cert and -key are required when -bind is not a loopback address"

	cases := []struct {
		bind, cert, key, want string
	}{
		{"127.0.0.1", "", "", ""},
		{"127.0.0.2", "", "", ""},
		{"::1", "", "", ""},
		{"localhost", "", "", ""},
		{"0.0.0.0", "", "", required},
		{"::", "", "", required},
		{"192.168.0.10", "", "", required},
		{"", "", "", required},
		{"192.168.0.10", "server.crt", "server.key", ""},
		{"0.0.0.0", "server.crt", "server.key", ""},
		{"127.0.0.1", "server.crt", "server.key", ""},
		{"127.0.0.1", "server.crt", "", together},
		{"127.0.0.1", "", "server.key", together},
		{"192.168.0.10", "server.crt", "", together},
		{"192.168.0.10", "", "server.key", together},
	}

	for _, c := range cases {
		err := validateListenConfig(c.bind, c.cert, c.key)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != c.want {
			t.Fatalf("Expected validateListenConfig(%q, %q, %q) = %q, got: %q", c.bind, c.cert, c.key, c.want, got)
		}
	}
}

func startTestServer(t *testing.T, certFile, keyFile string) (string, <-chan error) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Expected listener, got: %v", err)
	}
	srv := newServer(ln.Addr().String(), slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := make(chan error, 1)
	go func() { result <- serve(srv, ln, certFile, keyFile) }()
	t.Cleanup(func() { srv.Close() })

	return ln.Addr().String(), result
}

func TestServeTLS(t *testing.T) {
	certFile, keyFile, cert := writeTestCertificate(t)
	addr, _ := startTestServer(t, certFile, keyFile)

	roots := x509.NewCertPool()
	roots.AddCert(cert)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, err := transport.Dial(ctx, "wss://"+addr+"/ws", &tls.Config{RootCAs: roots})
	if err != nil {
		t.Fatalf("Expected wss handshake, got: %v", err)
	}
	conn.Close()

	if _, err := transport.Dial(ctx, "ws://"+addr+"/ws", nil); err == nil {
		t.Fatal("Expected plain WebSocket refused by TLS server, got: nil")
	}
}

func TestServeWithoutTLS(t *testing.T) {
	addr, _ := startTestServer(t, "", "")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, err := transport.Dial(ctx, "ws://"+addr+"/ws", nil)
	if err != nil {
		t.Fatalf("Expected ws connection on loopback, got: %v", err)
	}
	conn.Close()
}

func TestServeTLSWithBadFiles(t *testing.T) {
	certFile, _, _ := writeTestCertificate(t)
	_, otherKey, _ := writeTestCertificate(t)

	cases := map[string][2]string{
		"missing certificate": {filepath.Join(t.TempDir(), "missing.crt"), otherKey},
		"key does not match":  {certFile, otherKey},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			_, result := startTestServer(t, files[0], files[1])
			select {
			case err := <-result:
				if err == nil || errors.Is(err, http.ErrServerClosed) {
					t.Fatalf("Expected TLS file error, got: %v", err)
				}
				if name == "key does not match" && !strings.Contains(err.Error(), "tls: ") {
					t.Fatalf("Expected 'tls: ...' error, got: %v", err)
				}
			case <-time.After(testTimeout):
				t.Fatal("Expected serve to fail before timeout")
			}
		})
	}
}
