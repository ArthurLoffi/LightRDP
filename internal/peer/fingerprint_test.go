package peer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"regexp"
	"testing"

	"github.com/pion/webrtc/v5"
)

var fingerprintFormat = regexp.MustCompile(`^([0-9a-f]{2}:){31}[0-9a-f]{2}$`)

func TestDTLSFingerprintsMatchBetweenPeers(t *testing.T) {
	offerer := newTestPeer(t)
	answerer := newTestPeer(t)

	connectPeers(t, offerer, answerer)
	waitForState(t, offerer, webrtc.PeerConnectionStateConnected)
	waitForState(t, answerer, webrtc.PeerConnectionStateConnected)

	a, err := offerer.DTLSFingerprints()
	if err != nil {
		t.Fatalf("Expected offerer fingerprints, got: %v", err)
	}
	b, err := answerer.DTLSFingerprints()
	if err != nil {
		t.Fatalf("Expected answerer fingerprints, got: %v", err)
	}

	if a.Local != b.Remote || a.Remote != b.Local {
		t.Fatalf("Expected crossed fingerprints to match, got: offerer %+v, answerer %+v", a, b)
	}
	if a.Local == a.Remote {
		t.Fatalf("Expected different certificates on each side, got: %q", a.Local)
	}
	for _, fp := range []string{a.Local, a.Remote} {
		if !fingerprintFormat.MatchString(fp) {
			t.Fatalf("Expected sha-256 fingerprint format, got: %q", fp)
		}
	}
}

func TestDTLSFingerprintsBeforeConnect(t *testing.T) {
	p := newTestPeer(t)

	if _, err := p.DTLSFingerprints(); err == nil || err.Error() != "peer: remote DTLS certificate not available" {
		t.Fatalf("Expected 'peer: remote DTLS certificate not available', got: %v", err)
	}
}

func TestDTLSFingerprintsUseHandshakeCertificate(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Expected key, got: %v", err)
	}
	cert, err := webrtc.GenerateCertificate(key)
	if err != nil {
		t.Fatalf("Expected certificate, got: %v", err)
	}
	certFingerprints, err := cert.GetFingerprints()
	if err != nil {
		t.Fatalf("Expected certificate fingerprints, got: %v", err)
	}

	offerer, err := New(webrtc.Configuration{Certificates: []webrtc.Certificate{*cert}})
	if err != nil {
		t.Fatalf("Expected offerer, got: %v", err)
	}
	t.Cleanup(func() { offerer.Close() })
	answerer := newTestPeer(t)

	connectPeers(t, offerer, answerer)
	waitForState(t, offerer, webrtc.PeerConnectionStateConnected)
	waitForState(t, answerer, webrtc.PeerConnectionStateConnected)

	local, err := offerer.DTLSFingerprints()
	if err != nil {
		t.Fatalf("Expected offerer fingerprints, got: %v", err)
	}
	remote, err := answerer.DTLSFingerprints()
	if err != nil {
		t.Fatalf("Expected answerer fingerprints, got: %v", err)
	}
	if local.Local != certFingerprints[0].Value || remote.Remote != certFingerprints[0].Value {
		t.Fatalf("Expected %q on both sides, got: local %q, remote %q", certFingerprints[0].Value, local.Local, remote.Remote)
	}
}
