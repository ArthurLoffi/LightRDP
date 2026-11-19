// Package peer wraps the pion PeerConnection. It knows nothing about
// signaling: the caller decides how to send SDP and ICE candidates.
package peer

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/pion/webrtc/v5"
)

// Setup for starting a Peer connection
type Peer struct {
	pc *webrtc.PeerConnection

	// Remote candidates that arrive before the remote description are queued.
	mu        sync.Mutex
	remoteSet bool
	pending   []webrtc.ICECandidateInit
}

func New(config webrtc.Configuration) (*Peer, error) {
	// Creates one side of the connection pair.
	// It is a WebRTC connection for peer-to-peer communication
	// with another PeerConnection on the browser side.
	// The caller must close the connection with Close() when the application stops.
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("peer: new connection: %w", err)
	}

	return &Peer{pc: pc}, nil
}

// Closes the peer connection.
func (p *Peer) Close() error {
	return p.pc.Close()
}

// OnICECandidate calls fn for each local candidate (trickle ICE).
// The nil candidate, which marks the end of gathering, is ignored.
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

// OnDataChannel is used by the answerer: the channel is created by the other side.
func (p *Peer) OnDataChannel(fn func(*webrtc.DataChannel)) {
	p.pc.OnDataChannel(fn)
}

// CreateDataChannel is used by the offerer.
func (p *Peer) CreateDataChannel(label string) (*webrtc.DataChannel, error) {
	dc, err := p.pc.CreateDataChannel(label, nil)
	if err != nil {
		return nil, fmt.Errorf("peer: create data channel: %w", err)
	}
	return dc, nil
}

// CreateOffer creates the offer and sets it as the local description.
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

// HandleOffer applies the remote offer and returns the local answer.
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

// HandleAnswer applies the remote answer.
func (p *Peer) HandleAnswer(answer webrtc.SessionDescription) error {
	return p.setRemote(answer)
}

// AddICECandidate adds a remote candidate, or queues it if the
// remote description has not arrived yet.
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

type DTLSFingerprints struct {
	Local  string
	Remote string
}

func (p *Peer) DTLSFingerprints() (DTLSFingerprints, error) {
	dtls := p.pc.SCTP().Transport()

	remoteCertificate := dtls.GetRemoteCertificate()
	if len(remoteCertificate) == 0 {
		return DTLSFingerprints{}, errors.New("peer: remote DTLS certificate not available")
	}

	params, err := dtls.GetLocalParameters()
	if err != nil {
		return DTLSFingerprints{}, fmt.Errorf("peer: local DTLS parameters: %w", err)
	}
	for _, fp := range params.Fingerprints {
		if strings.EqualFold(fp.Algorithm, "sha-256") {
			return DTLSFingerprints{
				Local:  strings.ToLower(fp.Value),
				Remote: sha256Fingerprint(remoteCertificate),
			}, nil
		}
	}
	return DTLSFingerprints{}, errors.New("peer: no sha-256 local DTLS fingerprint")
}

func sha256Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, ":")
}
