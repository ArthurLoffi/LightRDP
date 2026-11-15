package config

import (
	"reflect"
	"testing"

	"github.com/pion/webrtc/v5"
)

func TestNewPeerConfigCopiesICEServers(t *testing.T) {
	servers := []webrtc.ICEServer{
		{URLs: []string{"stun:stun.example.com:3478"}},
		{URLs: []string{"turn:turn.example.com:3478"}, Username: "user", Credential: "pass"},
	}

	cfg := NewPeerConfig(PeerConfig{ICEServers: servers})

	if !reflect.DeepEqual(cfg.ICEServers, servers) {
		t.Fatalf("Expected ICE servers %+v, got: %+v", servers, cfg.ICEServers)
	}
}

func TestNewPeerConfigWithoutICEServers(t *testing.T) {
	cfg := NewPeerConfig(PeerConfig{})

	if len(cfg.ICEServers) != 0 {
		t.Fatalf("Expected no ICE servers, got: %+v", cfg.ICEServers)
	}
}
