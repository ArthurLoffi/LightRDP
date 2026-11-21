package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/fs"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeCertificateFile(t *testing.T) (string, *x509.Certificate) {
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
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("Expected parsed certificate, got: %v", err)
	}

	path := filepath.Join(t.TempDir(), "server.crt")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("Expected certificate written, got: %v", err)
	}
	return path, cert
}

func TestLoadTLSConfigWithoutCA(t *testing.T) {
	cfg, err := LoadTLSConfig("")
	if err != nil {
		t.Fatalf("Expected config, got: %v", err)
	}
	if cfg.RootCAs != nil || cfg.InsecureSkipVerify || cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("Expected system roots with verification, got: %+v", cfg)
	}
}

func TestLoadTLSConfigWithValidCA(t *testing.T) {
	path, cert := writeCertificateFile(t)

	cfg, err := LoadTLSConfig(path)
	if err != nil {
		t.Fatalf("Expected config, got: %v", err)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("Expected verification on, got: InsecureSkipVerify")
	}
	if _, err := cert.Verify(x509.VerifyOptions{Roots: cfg.RootCAs, DNSName: "192.168.0.10"}); err != nil {
		t.Fatalf("Expected certificate trusted by the config, got: %v", err)
	}
}

func TestLoadTLSConfigWithMissingCA(t *testing.T) {
	_, err := LoadTLSConfig(filepath.Join(t.TempDir(), "missing.crt"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Expected file not found error, got: %v", err)
	}
}

func TestLoadTLSConfigWithInvalidPEM(t *testing.T) {
	keyPEM := "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg\n-----END PRIVATE KEY-----\n"
	for name, content := range map[string]string{
		"text":     "not a certificate",
		"key file": keyPEM,
		"empty":    "",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.crt")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("Expected file written, got: %v", err)
			}

			_, err := LoadTLSConfig(path)
			if err == nil || err.Error() != "no valid PEM certificate in "+path {
				t.Fatalf("Expected 'no valid PEM certificate in %s', got: %v", path, err)
			}
		})
	}
}
