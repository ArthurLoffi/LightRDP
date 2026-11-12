package config

import "github.com/pion/webrtc/v5"

type PeerConfig struct {
	ICEServers []webrtc.ICEServer
}

func NewPeerConfig(peerConfig PeerConfig) webrtc.Configuration {
	return webrtc.Configuration{
		ICEServers: peerConfig.ICEServers,
	}
}
