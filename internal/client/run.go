package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pion/webrtc/v5"

	"LightRDP/internal/config"
	"LightRDP/internal/transport"
)

var URLS = []string{"stun:stun.l.google.com:19302"}

func LoadTLSConfig(caFile string) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile == "" {
		return cfg, nil
	}

	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	if !roots.AppendCertsFromPEM(pem) {
		return nil, errors.New("no valid PEM certificate in " + caFile)
	}
	cfg.RootCAs = roots
	return cfg, nil
}

func Run(serverURL string, tlsConfig *tls.Config, mode, selfID, hostID string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := transport.Dial(ctx, serverURL, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	cfg := config.NewPeerConfig(config.PeerConfig{
		ICEServers: []webrtc.ICEServer{
			{URLs: URLS},
		},
	})

	s := newSession(ctx, conn, cfg, selfID)
	defer s.close()

	if err := s.register(); err != nil {
		return err
	}
	fmt.Printf("Connected to server as %q\n", selfID)

	s.lines = stdinLines()
	if mode == "host" {
		fmt.Println("Waiting for connections...")
	} else {
		if err := s.join(hostID); err != nil {
			return err
		}
		fmt.Printf("Request sent to %q\n", hostID)
		fmt.Println("Waiting for host confirmation...")
	}

	return s.loop()
}
