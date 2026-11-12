package connection

import (
	"reflect"
	"testing"

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
