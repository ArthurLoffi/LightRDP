package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pion/webrtc/v5"

	"LightRDP/internal/config"
	"LightRDP/internal/transport"
)

var URLS = []string{"stun:stun.l.google.com:19302"}

func main() {
	serverURL := flag.String("server", "ws://localhost:8080/ws", "signaling server URL")
	mode := flag.String("mode", "", "host (waits for connections) or join (connects to a host)")
	id := flag.String("id", "", "your ID (generated if empty)")
	hostID := flag.String("peer", "", "host ID (required with -mode join)")
	caFile := flag.String("ca", "", "PEM file with extra trusted certificates for wss:// (self-signed server certificate or private CA)")
	flag.Parse()

	if *mode != "host" && *mode != "join" {
		fmt.Fprintln(os.Stderr, "usage: client -mode host|join [-server wss://IP:8080/ws] [-ca FILE] [-id ID] [-peer HOST_ID]")
		os.Exit(2)
	}
	if *mode == "join" && *hostID == "" {
		fmt.Fprintln(os.Stderr, "-peer is required with -mode join")
		os.Exit(2)
	}
	if *id == "" {
		*id = rand.Text()[:6]
	}

	tlsConfig, err := loadTLSConfig(*caFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: -ca:", err)
		os.Exit(1)
	}

	if err := run(*serverURL, tlsConfig, *mode, *id, *hostID); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func loadTLSConfig(caFile string) (*tls.Config, error) {
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

func run(serverURL string, tlsConfig *tls.Config, mode, selfID, hostID string) error {
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
