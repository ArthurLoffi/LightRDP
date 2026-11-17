package peer

import (
	"strings"
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

	// The other side goes away: disconnected -> failed (~30s, default ICE timeout).
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

	// Pion does not fire OnConnectionStateChange when the peer itself calls Close().
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

func TestCreateDataChannelReachesRemote(t *testing.T) {
	offerer := newTestPeer(t)
	answerer := newTestPeer(t)

	labels := make(chan string, 2)
	answerer.OnDataChannel(func(dc *webrtc.DataChannel) {
		labels <- dc.Label()
	})

	dc, err := offerer.CreateDataChannel("chat")
	if err != nil {
		t.Fatalf("Expected data channel, got: %v", err)
	}
	if dc.Label() != "chat" {
		t.Fatalf("Expected label 'chat', got: %q", dc.Label())
	}

	connectPeers(t, offerer, answerer)

	deadline := time.After(10 * time.Second)
	for {
		select {
		case label := <-labels:
			if label == "chat" {
				return
			}
		case <-deadline:
			t.Fatal("Expected remote data channel 'chat' before timeout")
		}
	}
}

func TestOnICECandidateAfterCreateOffer(t *testing.T) {
	peer := newTestPeer(t)

	candidates := make(chan webrtc.ICECandidateInit, 16)
	peer.OnICECandidate(func(c webrtc.ICECandidateInit) {
		candidates <- c
	})

	if _, err := peer.CreateDataChannel("test"); err != nil {
		t.Fatalf("Expected data channel, got: %v", err)
	}
	gathered := webrtc.GatheringCompletePromise(peer.pc)
	if _, err := peer.CreateOffer(); err != nil {
		t.Fatalf("Expected offer, got: %v", err)
	}

	select {
	case <-gathered:
	case <-time.After(10 * time.Second):
		t.Fatal("Expected ICE gathering to complete before timeout")
	}

	if len(candidates) == 0 {
		t.Fatal("Expected at least one local candidate, got: none")
	}
	for len(candidates) > 0 {
		if c := <-candidates; c.Candidate == "" {
			t.Fatalf("Expected non-empty candidate, got: %+v", c)
		}
	}
}

func TestHandleOfferWithInvalidSDP(t *testing.T) {
	peer := newTestPeer(t)

	_, err := peer.HandleOffer(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: "invalid"})
	if err == nil {
		t.Fatal("Expected error for invalid offer, got: nil")
	}
	if !strings.HasPrefix(err.Error(), "peer: set remote offer") {
		t.Fatalf("Expected error to start with 'peer: set remote offer', got: %v", err)
	}
}

func TestHandleAnswerWithoutLocalOffer(t *testing.T) {
	offerer := newTestPeer(t)
	answerer := newTestPeer(t)

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

	other := newTestPeer(t)
	err = other.HandleAnswer(answer)
	if err == nil {
		t.Fatal("Expected error for answer without local offer, got: nil")
	}
	if !strings.HasPrefix(err.Error(), "peer: set remote answer") {
		t.Fatalf("Expected error to start with 'peer: set remote answer', got: %v", err)
	}
}

func TestQueuedCandidatesAreApplied(t *testing.T) {
	offerer := newTestPeer(t)
	answerer := newTestPeer(t)

	if _, err := offerer.CreateDataChannel("test"); err != nil {
		t.Fatalf("Expected data channel, got: %v", err)
	}
	offer, offererCandidates := offerWithCandidates(t, offerer)

	for _, c := range offererCandidates {
		if err := answerer.AddICECandidate(c); err != nil {
			t.Fatalf("Expected candidate queued, got: %v", err)
		}
	}
	assertPending(t, answerer, len(offererCandidates))

	answerCandidates := collectCandidates(answerer)
	answerGathered := webrtc.GatheringCompletePromise(answerer.pc)
	answer, err := answerer.HandleOffer(offer)
	if err != nil {
		t.Fatalf("Expected answer, got: %v", err)
	}
	assertPending(t, answerer, 0)

	waitGathered(t, answerGathered)
	for _, c := range drainCandidates(answerCandidates) {
		if err := offerer.AddICECandidate(c); err != nil {
			t.Fatalf("Expected candidate queued, got: %v", err)
		}
	}

	if err := offerer.HandleAnswer(answer); err != nil {
		t.Fatalf("Expected answer applied, got: %v", err)
	}
	assertPending(t, offerer, 0)

	waitForState(t, offerer, webrtc.PeerConnectionStateConnected)
	waitForState(t, answerer, webrtc.PeerConnectionStateConnected)
}

func TestCreateOfferOnClosedPeer(t *testing.T) {
	peer := newTestPeer(t)
	if err := peer.Close(); err != nil {
		t.Fatalf("Expected close without error, got: %v", err)
	}

	_, err := peer.CreateOffer()
	if err == nil {
		t.Fatal("Expected error on closed peer, got: nil")
	}
	if !strings.HasPrefix(err.Error(), "peer: create offer") {
		t.Fatalf("Expected error to start with 'peer: create offer', got: %v", err)
	}
}

func newTestPeer(t *testing.T) *Peer {
	t.Helper()

	peer, err := New(webrtc.Configuration{})
	if err != nil {
		t.Fatalf("Expected new peer connection, got: %v", err)
	}
	t.Cleanup(func() { peer.Close() })

	return peer
}

func collectCandidates(peer *Peer) chan webrtc.ICECandidateInit {
	candidates := make(chan webrtc.ICECandidateInit, 64)
	peer.OnICECandidate(func(c webrtc.ICECandidateInit) {
		candidates <- c
	})
	return candidates
}

func drainCandidates(candidates chan webrtc.ICECandidateInit) []webrtc.ICECandidateInit {
	var all []webrtc.ICECandidateInit
	for len(candidates) > 0 {
		all = append(all, <-candidates)
	}
	return all
}

func waitGathered(t *testing.T, gathered <-chan struct{}) {
	t.Helper()

	select {
	case <-gathered:
	case <-time.After(10 * time.Second):
		t.Fatal("Expected ICE gathering to complete before timeout")
	}
}

func offerWithCandidates(t *testing.T, peer *Peer) (webrtc.SessionDescription, []webrtc.ICECandidateInit) {
	t.Helper()

	candidates := collectCandidates(peer)
	gathered := webrtc.GatheringCompletePromise(peer.pc)

	offer, err := peer.CreateOffer()
	if err != nil {
		t.Fatalf("Expected offer, got: %v", err)
	}
	waitGathered(t, gathered)

	all := drainCandidates(candidates)
	if len(all) == 0 {
		t.Fatal("Expected local candidates, got: none")
	}
	return offer, all
}

func assertPending(t *testing.T, peer *Peer, want int) {
	t.Helper()

	peer.mu.Lock()
	got := len(peer.pending)
	peer.mu.Unlock()

	if got != want {
		t.Fatalf("Expected %d queued candidates, got: %d", want, got)
	}
}

// Forwards the failed/closed states to a channel.
func watchFinish(peer *Peer) <-chan webrtc.PeerConnectionState {
	finished := make(chan webrtc.PeerConnectionState, 1)
	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			finished <- state
		}
	})
	return finished
}

// Acts as the signaling server between the two peers.
// Candidates may arrive before the remote description and are queued by the Peer.
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

// Waits for the state by polling: registering another OnConnectionStateChange
// would replace the handler already registered.
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
