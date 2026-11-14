package peer

import (
	"testing"
	"time"

	"github.com/pion/webrtc/v5"
)

var config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}

func TestNew(t *testing.T) {
	peer, err := New(config)
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}
	defer peer.Close()

	if peer.pc == nil {
		t.Fatal("Expected a peer connection, got: nil")
	}

	if peer.pc.ConnectionState() != webrtc.PeerConnectionStateNew {
		t.Fatalf("Expected connection state 'new', got: %v", peer.pc.ConnectionState())
	}
}

func TestNewWithWrongURL(t *testing.T) {
	wrongConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"abc:1234"},
			},
		},
	}

	peer, err := New(wrongConfig)
	if err == nil {
		t.Fatalf("Expected error to connection with wrong url, got state: %v", peer.pc.ConnectionState())
	}
}

func TestClose(t *testing.T) {
	peer, err := New(config)
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}

	if err := peer.Close(); err != nil {
		t.Fatalf("Expected close without error, got: %v", err)
	}

	if peer.pc.ConnectionState() != webrtc.PeerConnectionStateClosed {
		t.Fatalf("Expected connection closed, got: %v", peer.pc.ConnectionState())
	}

	if _, err = peer.CreateOffer(); err == nil {
		t.Fatal("Expected error to create a offer with connection closed, got: created offer")
	}
}

func TestOnConnectionStateChangeOnRemoteFailure(t *testing.T) {
	peer, err := New(config)
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}
	defer peer.Close()

	remote, err := New(config)
	if err != nil {
		t.Fatalf("Expected new remote peer connection, got: %v", err)
	}
	defer remote.Close()

	finished := watchFinish(peer)

	connectPeers(t, peer, remote)
	waitForState(t, peer, webrtc.PeerConnectionStateConnected)

	select {
	case state := <-finished:
		t.Fatalf("Expected no finish state before failure, got: %v", state)
	default:
	}

	// O outro lado some: disconnected -> failed (~30s, timeout padrão do ICE).
	remote.Close()

	select {
	case state := <-finished:
		if state != webrtc.PeerConnectionStateFailed {
			t.Fatalf("Expected connection state 'failed', got: %v", state)
		}
	case <-time.After(45 * time.Second):
		t.Fatalf("Expected failed state, got state: %v", peer.pc.ConnectionState())
	}
}

func TestOnConnectionStateChangeWithNilHandler(t *testing.T) {
	peer, err := New(config)
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}
	defer peer.Close()

	remote, err := New(config)
	if err != nil {
		t.Fatalf("Expected new remote peer connection, got: %v", err)
	}
	defer remote.Close()

	peer.OnConnectionStateChange(nil)

	connectPeers(t, peer, remote)
	waitForState(t, peer, webrtc.PeerConnectionStateConnected)
}

func TestOnConnectionStateChangeOnLocalClose(t *testing.T) {
	peer, err := New(config)
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}

	remote, err := New(config)
	if err != nil {
		t.Fatalf("Expected new remote peer connection, got: %v", err)
	}
	defer remote.Close()

	finished := watchFinish(peer)

	connectPeers(t, peer, remote)
	waitForState(t, peer, webrtc.PeerConnectionStateConnected)

	// A Pion não dispara OnConnectionStateChange quando o próprio peer chama Close().
	peer.Close()

	select {
	case state := <-finished:
		t.Fatalf("Expected no finish state on local close, got: %v", state)
	case <-time.After(1 * time.Second):
	}
}

func TestAddICECandidateBeforeRemoteDescription(t *testing.T) {
	peer, err := New(config)
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}
	defer peer.Close()

	candidate := webrtc.ICECandidateInit{Candidate: "candidate:1 1 udp 2130706431 192.168.0.10 50000 typ host"}
	if err := peer.AddICECandidate(candidate); err != nil {
		t.Fatalf("Expected candidate to be queued, got: %v", err)
	}

	peer.mu.Lock()
	queued := len(peer.pending)
	peer.mu.Unlock()

	if queued != 1 {
		t.Fatalf("Expected 1 queued candidate, got: %d", queued)
	}
}

// Encaminha os estados failed/closed para um canal.
func watchFinish(peer *Peer) <-chan webrtc.PeerConnectionState {
	finished := make(chan webrtc.PeerConnectionState, 1)
	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			finished <- state
		}
	})
	return finished
}

// Faz o papel do servidor de sinalização entre os dois peers.
// Os candidatos podem chegar antes da descrição remota e ficam na fila do Peer.
func connectPeers(t *testing.T, offerer, answerer *Peer) {
	t.Helper()

	offerer.OnICECandidate(func(c webrtc.ICECandidateInit) {
		_ = answerer.AddICECandidate(c)
	})
	answerer.OnICECandidate(func(c webrtc.ICECandidateInit) {
		_ = offerer.AddICECandidate(c)
	})

	if _, err := offerer.CreateDataChannel("test"); err != nil {
		t.Fatalf("Expected data channel, got: %v", err)
	}

	offer, err := offerer.CreateOffer()
	if err != nil {
		t.Fatalf("Expected offer, got: %v", err)
	}

	answer, err := answerer.HandleOffer(offer)
	if err != nil {
		t.Fatalf("Expected answer, got: %v", err)
	}

	if err := offerer.HandleAnswer(answer); err != nil {
		t.Fatalf("Expected offerer remote description, got: %v", err)
	}
}

// Espera o estado consultando em loop: registrar outro OnConnectionStateChange
// substituiria o handler já registrado.
func waitForState(t *testing.T, peer *Peer, want webrtc.PeerConnectionState) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for peer.pc.ConnectionState() != want {
		if time.Now().After(deadline) {
			t.Fatalf("Expected connection state '%v', got: %v", want, peer.pc.ConnectionState())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
