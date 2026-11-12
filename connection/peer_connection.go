package connection

import (
	"fmt"
	"log"

	"github.com/pion/webrtc/v5"
)

// Configuração para inicialização de uma conexão Peer
type Peer struct {
	config         webrtc.Configuration
	connectionType string
	PeerCnn        *webrtc.PeerConnection
}

// Criada o par de conexão de um lado.
// É uma conexão WebRTC para comunicações peer-to-peer
// com outro PeerConnection pelo lado do navegador.
// Quem chama deve fechar a conexão com Close() quando a aplicação parar.
// Adicionar api para realização de testes com timeout curto
func NewPeerConnection(config webrtc.Configuration, connectionType string) (*Peer, error) {
	return newPeerConnection(webrtc.NewAPI(), config, connectionType)
}

func newPeerConnection(api *webrtc.API, config webrtc.Configuration, connectionType string) (*Peer, error) {
	peerCnn, err := api.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	return &Peer{
		config:         config,
		connectionType: connectionType,
		PeerCnn:        peerCnn,
	}, nil
}

// Fecha a conexão peer.
func (p *Peer) Close() {
	if err := p.PeerCnn.Close(); err != nil {
		log.Printf("Error closing peer connection: %v", err)
	}
}

// Handler para controlar e exibir mudanças no estado da conexão.
func (p *Peer) PeerHandler(onFinish func(webrtc.PeerConnectionState)) {
	p.PeerCnn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer connection state changed: %s\n", state.String())

		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
		if onFinish != nil {
			onFinish(state)
		}
	}
})
}
