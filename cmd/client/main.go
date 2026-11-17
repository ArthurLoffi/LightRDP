package main

import (
	"context"
	"crypto/rand"
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
	flag.Parse()

	if *mode != "host" && *mode != "join" {
		fmt.Fprintln(os.Stderr, "usage: client -mode host|join [-server ws://IP:8080/ws] [-id ID] [-peer HOST_ID]")
		os.Exit(2)
	}
	if *mode == "join" && *hostID == "" {
		fmt.Fprintln(os.Stderr, "-peer is required with -mode join")
		os.Exit(2)
	}
	if *id == "" {
		*id = rand.Text()[:6]
	}

	if err := run(*serverURL, *mode, *id, *hostID); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(serverURL, mode, selfID, hostID string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := transport.Dial(ctx, serverURL)
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

	if mode == "host" {
		s.lines = stdinLines()
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
