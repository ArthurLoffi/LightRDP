package connection

import (
	"reflect"
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

func TestNewPeerConnection(t *testing.T) {
	peer, err := NewPeerConnection(config, "DataChannel")
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}
	defer peer.Close()

	if !reflect.DeepEqual(peer.config, config) || peer.connectionType != "DataChannel" || peer.PeerCnn == nil {
		t.Fatalf("Expected same items passed and a peer connection, got: config - %v, connection type - %s, peer cnn - %v", peer.config, peer.connectionType, peer.PeerCnn)
	}

	if peer.PeerCnn.ConnectionState() != webrtc.PeerConnectionStateNew {
		t.Fatalf("Expected connection state 'new', got: %v", peer.PeerCnn.ConnectionState())
	}
}

func TestNewPeerConnectionWithWrongURL(t *testing.T) {
	wrongConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"abc:1234"},
			},
		},
	}

	peer, err := NewPeerConnection(wrongConfig, "DataChannel")
	if err == nil {
		t.Fatalf("Expected error to connection with wrong url, got state: %v", peer.PeerCnn.ConnectionState())
	}
}

func TestPeerConnectionClose(t *testing.T) {
	peer, err := NewPeerConnection(config, "DataChannel")
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}

	peer.Close()

	if peer.PeerCnn.ConnectionState() != webrtc.PeerConnectionStateClosed {
		t.Fatalf("Expected connection closed, got: %v", peer.PeerCnn.ConnectionState())
	}

	_, err = peer.PeerCnn.CreateOffer(&webrtc.OfferOptions{ICERestart: false})
	if err == nil {
		t.Fatal("Expected error to create a offer with connection closed, got: created offer")
	}
}

func TestNewPeerHandler(t *testing.T) {
	peer, err := NewPeerConnection(config, "DataChannel")
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}
	defer peer.Close()

	remote, err := NewPeerConnection(config, "DataChannel")
	if err != nil {
		t.Fatalf("Expected new remote peer connection, got: %v", err)
	}
	defer remote.Close()

	finished := make(chan webrtc.PeerConnectionState, 1)
	peer.PeerHandler(func(state webrtc.PeerConnectionState) {
		finished <- state
	})

	connectPeers(t, peer, remote)
	waitForState(t, peer, webrtc.PeerConnectionStateConnected)

	select {
	case state := <-finished:
		t.Fatalf("Expected onFinish not to be called before failure, got: %v", state)
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
		t.Fatalf("Expected onFinish to be called, got state: %v", peer.PeerCnn.ConnectionState())
	}
}

func TestNewPeerHandlerWithNilOnFinish(t *testing.T) {
	peer, err := NewPeerConnection(config, "DataChannel")
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}
	defer peer.Close()

	remote, err := NewPeerConnection(config, "DataChannel")
	if err != nil {
		t.Fatalf("Expected new remote peer connection, got: %v", err)
	}
	defer remote.Close()

	peer.PeerHandler(nil)

	connectPeers(t, peer, remote)
	waitForState(t, peer, webrtc.PeerConnectionStateConnected)
}

func TestNewPeerHandlerOnLocalClose(t *testing.T) {
	peer, err := NewPeerConnection(config, "DataChannel")
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}

	remote, err := NewPeerConnection(config, "DataChannel")
	if err != nil {
		t.Fatalf("Expected new remote peer connection, got: %v", err)
	}
	defer remote.Close()

	finished := make(chan webrtc.PeerConnectionState, 1)
	peer.PeerHandler(func(state webrtc.PeerConnectionState) {
		finished <- state
	})

	connectPeers(t, peer, remote)
	waitForState(t, peer, webrtc.PeerConnectionStateConnected)

	// A Pion não dispara OnConnectionStateChange quando o próprio peer chama Close().
	peer.Close()

	select {
	case state := <-finished:
		t.Fatalf("Expected onFinish not to be called on local close, got: %v", state)
	case <-time.After(1 * time.Second):
	}
}

// Faz o papel do servidor de sinalização entre os dois peers.
func connectPeers(t *testing.T, offerer, answerer *Peer) {
	t.Helper()

	offerer.PeerCnn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			_ = answerer.PeerCnn.AddICECandidate(c.ToJSON())
		}
	})
	answerer.PeerCnn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			_ = offerer.PeerCnn.AddICECandidate(c.ToJSON())
		}
	})

	if _, err := offerer.PeerCnn.CreateDataChannel("test", nil); err != nil {
		t.Fatalf("Expected data channel, got: %v", err)
	}

	offer, err := offerer.PeerCnn.CreateOffer(nil)
	if err != nil {
		t.Fatalf("Expected offer, got: %v", err)
	}
	if err := offerer.PeerCnn.SetLocalDescription(offer); err != nil {
		t.Fatalf("Expected offerer local description, got: %v", err)
	}
	if err := answerer.PeerCnn.SetRemoteDescription(offer); err != nil {
		t.Fatalf("Expected answerer remote description, got: %v", err)
	}

	answer, err := answerer.PeerCnn.CreateAnswer(nil)
	if err != nil {
		t.Fatalf("Expected answer, got: %v", err)
	}
	if err := answerer.PeerCnn.SetLocalDescription(answer); err != nil {
		t.Fatalf("Expected answerer local description, got: %v", err)
	}
	if err := offerer.PeerCnn.SetRemoteDescription(answer); err != nil {
		t.Fatalf("Expected offerer remote description, got: %v", err)
	}
}

// Espera o estado consultando em loop: registrar outro OnConnectionStateChange
// substituiria o handler do PeerHandler.
func waitForState(t *testing.T, peer *Peer, want webrtc.PeerConnectionState) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for peer.PeerCnn.ConnectionState() != want {
		if time.Now().After(deadline) {
			t.Fatalf("Expected connection state '%v', got: %v", want, peer.PeerCnn.ConnectionState())
		}
		time.Sleep(20 * time.Millisecond)
	}
}