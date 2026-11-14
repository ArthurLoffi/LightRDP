// Package peer encapsula a PeerConnection do pion. Não sabe nada de
// sinalização: quem chama decide como enviar SDP e candidatos ICE.
package peer

import (
	"fmt"
	"sync"

	"github.com/pion/webrtc/v5"
)

// Configuração para inicialização de uma conexão Peer
type Peer struct {
	pc *webrtc.PeerConnection

	// Candidatos remotos que chegam antes da descrição remota ficam na fila.
	mu        sync.Mutex
	remoteSet bool
	pending   []webrtc.ICECandidateInit
}

func New(config webrtc.Configuration) (*Peer, error) {
	// Criada o par de conexão de um lado.
	// É uma conexão WebRTC para comunicações peer-to-peer
	// com outro PeerConnection pelo lado do navegador.
	// Quem chama deve fechar a conexão com Close() quando a aplicação parar.
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("peer: new connection: %w", err)
	}

	return &Peer{pc: pc}, nil
}

// Fecha a conexão peer.
func (p *Peer) Close() error {
	return p.pc.Close()
}

// OnICECandidate chama fn para cada candidato local (trickle ICE).
// O candidato nil, que marca o fim da coleta, é ignorado.
func (p *Peer) OnICECandidate(fn func(webrtc.ICECandidateInit)) {
	p.pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			fn(c.ToJSON())
		}
	})
}

func (p *Peer) OnConnectionStateChange(fn func(webrtc.PeerConnectionState)) {
	p.pc.OnConnectionStateChange(fn)
}

// OnDataChannel é usado por quem responde: o canal é criado pelo outro lado.
func (p *Peer) OnDataChannel(fn func(*webrtc.DataChannel)) {
	p.pc.OnDataChannel(fn)
}

// CreateDataChannel é usado por quem faz a offer.
func (p *Peer) CreateDataChannel(label string) (*webrtc.DataChannel, error) {
	dc, err := p.pc.CreateDataChannel(label, nil)
	if err != nil {
		return nil, fmt.Errorf("peer: create data channel: %w", err)
	}
	return dc, nil
}

// CreateOffer cria a offer e a define como descrição local.
func (p *Peer) CreateOffer() (webrtc.SessionDescription, error) {
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("peer: create offer: %w", err)
	}
	if err := p.pc.SetLocalDescription(offer); err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("peer: set local offer: %w", err)
	}
	return offer, nil
}

// HandleOffer aplica a offer remota e devolve a answer local.
func (p *Peer) HandleOffer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if err := p.setRemote(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("peer: create answer: %w", err)
	}
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("peer: set local answer: %w", err)
	}
	return answer, nil
}

// HandleAnswer aplica a answer remota.
func (p *Peer) HandleAnswer(answer webrtc.SessionDescription) error {
	return p.setRemote(answer)
}

// AddICECandidate adiciona um candidato remoto, ou o enfileira se a
// descrição remota ainda não chegou.
func (p *Peer) AddICECandidate(c webrtc.ICECandidateInit) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.remoteSet {
		p.pending = append(p.pending, c)
		return nil
	}
	if err := p.pc.AddICECandidate(c); err != nil {
		return fmt.Errorf("peer: add candidate: %w", err)
	}
	return nil
}

func (p *Peer) setRemote(desc webrtc.SessionDescription) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.pc.SetRemoteDescription(desc); err != nil {
		return fmt.Errorf("peer: set remote %s: %w", desc.Type, err)
	}
	p.remoteSet = true

	for _, c := range p.pending {
		if err := p.pc.AddICECandidate(c); err != nil {
			return fmt.Errorf("peer: add queued candidate: %w", err)
		}
	}
	p.pending = nil

	return nil
}
