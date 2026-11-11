package connection

import (
	"fmt"
	"log"
	"os"

	"github.com/pion/webrtc/v5"
)

// Configuração para inicialização de uma conexão Peer
type Peer struct {
	config webrtc.Configuration
	connectionType string
	PeerCnn *webrtc.PeerConnection
}

func NewPeerConnection(config webrtc.Configuration, connectionType string) (*Peer, error) {
	// Criada o par de conexão de um lado.
	// É uma conexão WebRTC para comunicações peer-to-peer
	// com outro PeerConnection pelo lado do navegador.
	// Quem chama deve fechar a conexão com Close() quando a aplicação parar.
	peerCnn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	peer := Peer{
		config: config,
		connectionType: connectionType,
		PeerCnn: peerCnn,
	}

	return &peer, nil
}

// Fecha a conexão peer.
func (p *Peer) Close() {
	if err := p.PeerCnn.Close(); err != nil {
		log.Printf("Error closing peer connection: %v", err)
	}
}

func (p *Peer) PeerHandler() {
	// Handler para controlar e exibir mudanças no estado da conexão.
	p.PeerCnn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
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
