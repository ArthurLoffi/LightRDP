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
	serverURL := flag.String("server", "ws://localhost:8080/ws", "URL do servidor de sinalização")
	mode := flag.String("mode", "", "host (aguarda conexões) ou join (conecta a um host)")
	id := flag.String("id", "", "seu ID (gerado se vazio)")
	hostID := flag.String("peer", "", "ID do host (obrigatório em -mode join)")
	flag.Parse()

	if *mode != "host" && *mode != "join" {
		fmt.Fprintln(os.Stderr, "uso: client -mode host|join [-server ws://IP:8080/ws] [-id ID] [-peer ID_DO_HOST]")
		os.Exit(2)
	}
	if *mode == "join" && *hostID == "" {
		fmt.Fprintln(os.Stderr, "-peer é obrigatório em -mode join")
		os.Exit(2)
	}
	if *id == "" {
		*id = rand.Text()[:6]
	}

	if err := run(*serverURL, *mode, *id, *hostID); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
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
	fmt.Printf("Conectado ao servidor como %q\n", selfID)

	if mode == "host" {
		fmt.Println("Aguardando conexões...")
	} else {
		if err := s.join(hostID); err != nil {
			return err
		}
		fmt.Printf("Pedido enviado para %q\n", hostID)
	}

	return s.loop()
}
