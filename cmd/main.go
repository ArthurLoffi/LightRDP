package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pion/webrtc/v4"
)

func main() {
	// Configuração do servidor auxiliar de conexão entre os dois peers.
	// Para conseguirem se conectar mesmo estando atras de NAT ou firewall.
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"localhost:8080"},
			},
		},
	}

	// Criada o par de conexão de um lado.
	// É uma conexão WebRTC para comunicações peer-to-peer
	// com outro PeerConnection pelo lado do navegador.
	// Adiar para fechar essa conexão assim que a aplicação parar.
	peerCnn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer func() {
		if err := peerCnn.Close(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}()

	// Handler para controlar e exibir mudanças no estado da conexão.
	peerCnn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer connection state changed: %s\n", state.String())

		// Muda o estado para falhar se ficar 30 segundos
		// sem atividade ou algum outro tipo de falha.
		if state == webrtc.PeerConnectionStateFailed {
			fmt.Println("Peer connection failed")
			os.Exit(1)
		}

		if state == webrtc.PeerConnectionStateClosed {
			fmt.Println("Peer connection closed")
			os.Exit(0)
		}
	})
}
