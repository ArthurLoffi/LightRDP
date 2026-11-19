package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
)

func newTLSTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		_, _, _ = conn.Read(r.Context())
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestDialRefusesInsecureURL(t *testing.T) {
	for _, url := range []string{
		"ws://192.168.0.10:8080/ws",
		"ws://0.0.0.0:8080/ws",
		"ws://[::]:8080/ws",
		"ws://example.com/ws",
		"WS://192.168.0.10:8080/ws",
		"http://192.168.0.10:8080/ws",
	} {
		t.Run(url, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			conn, err := Dial(ctx, url, nil)
			if err == nil {
				conn.Close()
				t.Fatal("Expected insecure URL error, got: nil")
			}
			want := "transport: insecure server URL " + url + ": use wss://"
			if err.Error() != want {
				t.Fatalf("Expected %q, got: %q", want, err.Error())
			}
		})
	}
}

func TestDialAcceptsLoopbackWebSocket(t *testing.T) {
	ts := newTestServer(t)
	port := ts.url[strings.LastIndex(ts.url, ":"):]

	for _, url := range []string{ts.url, "ws://localhost" + port, "ws://LOCALHOST" + port} {
		t.Run(url, func(t *testing.T) {
			conn := dialTestServer(t, url)
			ts.serverConn(t)
			conn.Close()
		})
	}
}

func TestIsLoopbackHost(t *testing.T) {
	cases := map[string]bool{
		"localhost":    true,
		"LocalHost":    true,
		"127.0.0.1":    true,
		"127.1.2.3":    true,
		"::1":          true,
		"0.0.0.0":      false,
		"::":           false,
		"192.168.0.10": false,
		"example.com":  false,
		"":             false,
	}
	for host, want := range cases {
		if got := IsLoopbackHost(host); got != want {
			t.Fatalf("Expected IsLoopbackHost(%q) = %v, got: %v", host, want, got)
		}
	}
}

func TestDialSecureWithTrustedCertificate(t *testing.T) {
	srv := newTLSTestServer(t)
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	url := "wss" + strings.TrimPrefix(srv.URL, "https")
	conn, err := Dial(ctx, url, &tls.Config{RootCAs: roots})
	if err != nil {
		t.Fatalf("Expected wss connection with trusted certificate, got: %v", err)
	}
	conn.Close()
}

func TestDialSecureWithUntrustedCertificate(t *testing.T) {
	srv := newTLSTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	url := "wss" + strings.TrimPrefix(srv.URL, "https")
	for name, cfg := range map[string]*tls.Config{"system roots": nil, "empty pool": {RootCAs: x509.NewCertPool()}} {
		t.Run(name, func(t *testing.T) {
			conn, err := Dial(ctx, url, cfg)
			if err == nil {
				conn.Close()
				t.Fatal("Expected TLS error, got: nil")
			}
			var unknownAuthority x509.UnknownAuthorityError
			if !errors.As(err, &unknownAuthority) {
				t.Fatalf("Expected x509 unknown authority error, got: %v", err)
			}
			if !strings.HasPrefix(err.Error(), "transport: dial "+url+": ") {
				t.Fatalf("Expected error naming %s, got: %v", url, err)
			}
		})
	}
}
