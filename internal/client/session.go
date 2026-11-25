package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v5"

	"LightRDP/internal/peer"
	"LightRDP/internal/signaling"
	"LightRDP/internal/transport"
)

const (
	sendTimeout            = 5 * time.Second
	hostConfirmTimeout     = 60 * time.Second
	safetyHandshakeTimeout = 60 * time.Second
	hostAnswerGrace        = time.Second

	hostBusyMessage             = "host busy"
	refusedByHostMessage        = "refused by host"
	confirmTimeoutMessage       = "timed out waiting for host"
	safetyCodeNotConfirmedError = "safety code not confirmed"

	joinerQuestion = "Same code on the host? [y/N] "
)

type peerState struct {
	peer  *peer.Peer
	state webrtc.PeerConnectionState
}

// session links signaling (transport) to the WebRTC connection (peer).
// Supports one P2P session at a time.
type session struct {
	ctx    context.Context
	conn   *transport.Conn
	cfg    webrtc.Configuration
	selfID string

	joining  bool
	remoteID string
	peer     *peer.Peer
	chat     *chat

	connected  atomic.Bool
	states     chan peerState
	handshakes chan safetyHandshakeResult

	lines          <-chan stdinLine
	confirmTimeout time.Duration
	answerGrace    time.Duration
	question       string
	promptedAt     time.Time
	confirmed      bool
	confirmExpired <-chan time.Time

	handshakeTimeout time.Duration
	handshakeExpired <-chan time.Time
}

func newSession(ctx context.Context, conn *transport.Conn, cfg webrtc.Configuration, selfID string) *session {
	return &session{
		ctx:        ctx,
		conn:       conn,
		cfg:        cfg,
		selfID:     selfID,
		states:     make(chan peerState, 16),
		handshakes: make(chan safetyHandshakeResult, 16),

		confirmTimeout:   hostConfirmTimeout,
		answerGrace:      hostAnswerGrace,
		handshakeTimeout: safetyHandshakeTimeout,
	}
}

func (s *session) close() {
	if s.peer != nil {
		_ = s.peer.Close()
	}
}

func (s *session) register() error {
	return s.send(signaling.TypeRegister, "", nil)
}

// join is called by the joiner: prepares the peer and asks the host for a session.
func (s *session) join(hostID string) error {
	s.joining = true
	if err := s.startPeer(hostID); err != nil {
		return err
	}
	// The answerer receives the DataChannel created by the other side.
	s.peer.OnDataChannel(s.chat.attach)

	return s.send(signaling.TypeConnect, hostID, nil)
}

// loop handles server messages until the session ends.
func (s *session) loop() error {
	msgs := s.conn.Messages()
	lines := s.lines

	for {
		var answers <-chan stdinLine
		if !s.confirmed {
			answers = lines
		}

		select {
		case <-s.ctx.Done():
			return nil

		case ev := <-s.states:
			ended, err := s.onPeerState(ev)
			if err != nil || ended {
				return err
			}

		case result := <-s.handshakes:
			if err := s.onSafetyHandshake(result); err != nil {
				return err
			}

		case <-s.handshakeExpired:
			return handshakeFailed("timed out")

		case msg, ok := <-msgs:
			if !ok {
				// Without the server, an already open P2P connection keeps working.
				if s.connected.Load() {
					fmt.Println("Signaling server disconnected; P2P chat continues.")
					msgs = nil
					continue
				}
				return fmt.Errorf("signaling server disconnected: %w", s.conn.Err())
			}
			if err := s.handle(msg); err != nil {
				return err
			}

		case line, ok := <-answers:
			if !ok {
				lines = nil
				continue
			}
			if err := s.onAnswer(line); err != nil {
				return err
			}

		case <-s.confirmExpired:
			if err := s.onConfirmTimeout(); err != nil {
				return err
			}
		}
	}
}

func (s *session) handle(msg signaling.Message) error {
	switch msg.Type {
	case signaling.TypeConnect:
		return s.onConnect(msg.From)

	case signaling.TypeOffer:
		if !s.fromRemote(msg) {
			return nil
		}
		var offer webrtc.SessionDescription
		if err := msg.Decode(&offer); err != nil {
			return s.onInvalidRemoteMessage(err)
		}
		answer, err := s.peer.HandleOffer(offer)
		if err != nil {
			return s.onInvalidRemoteMessage(err)
		}
		return s.send(signaling.TypeAnswer, s.remoteID, answer)

	case signaling.TypeAnswer:
		if !s.fromRemote(msg) {
			return nil
		}
		var answer webrtc.SessionDescription
		if err := msg.Decode(&answer); err != nil {
			return s.onInvalidRemoteMessage(err)
		}
		return s.onInvalidRemoteMessage(s.peer.HandleAnswer(answer))

	case signaling.TypeCandidate:
		if !s.fromRemote(msg) {
			return nil
		}
		var c webrtc.ICECandidateInit
		if err := msg.Decode(&c); err != nil {
			return s.onInvalidRemoteMessage(err)
		}
		return s.onInvalidRemoteMessage(s.peer.AddICECandidate(c))

	case signaling.TypeReject:
		if !s.fromRemote(msg) || s.chatOpen() {
			return nil
		}
		var e signaling.ErrorPayload
		_ = msg.Decode(&e)
		refused := errors.New("connection refused: " + sanitize(e.Message))
		s.interruptPrompt()
		if s.joining {
			return refused
		}
		s.dropAttempt()
		fmt.Fprintln(os.Stderr, "error:", refused)
		return nil

	case signaling.TypeError:
		// Errors are only accepted from the server.
		if msg.From != "" {
			return nil
		}
		var e signaling.ErrorPayload
		_ = msg.Decode(&e)
		// Before the P2P connection, a signaling error ends the session.
		if !s.connected.Load() {
			return errors.New("server: " + sanitize(e.Message))
		}
		fmt.Println("server notice:", sanitize(e.Message))
		return nil

	default:
		fmt.Printf("unknown message: %q\n", msg.Type)
		return nil
	}
}

// onConnect is called on the host when someone asks for a session.
func (s *session) onConnect(from string) error {
	if s.peer != nil {
		return s.send(signaling.TypeReject, from, signaling.ErrorPayload{Message: hostBusyMessage})
	}

	fmt.Printf("Connection request from %q\n", sanitize(from))
	s.confirmExpired = time.After(s.confirmTimeout)

	if err := s.startPeer(from); err != nil {
		return err
	}

	// The offerer creates the DataChannel.
	dc, err := s.peer.CreateDataChannel("chat")
	if err != nil {
		return err
	}
	s.chat.attach(dc)

	offer, err := s.peer.CreateOffer()
	if err != nil {
		return err
	}
	return s.send(signaling.TypeOffer, s.remoteID, offer)
}

func (s *session) onPeerState(ev peerState) (bool, error) {
	if ev.peer != s.peer {
		return false, nil
	}

	switch ev.state {
	case webrtc.PeerConnectionStateConnected:
		s.connected.Store(true)
		if s.joining && !s.confirmed && s.question == "" {
			s.handshakeExpired = time.After(s.handshakeTimeout)
		}
		return false, nil
	// The state changes to failed after 30 seconds
	// without activity or on any other failure.
	case webrtc.PeerConnectionStateFailed:
		return s.endAttempt(errors.New("peer connection failed"))
	case webrtc.PeerConnectionStateClosed:
		return s.endAttempt(nil)
	}
	return false, nil
}

func (s *session) endAttempt(err error) (bool, error) {
	s.interruptPrompt()
	if s.joining || s.chatOpen() {
		return true, err
	}
	s.dropAttempt()
	fmt.Println("Waiting for connections...")
	return false, nil
}

func (s *session) onInvalidRemoteMessage(err error) error {
	if err == nil || s.joining || s.chatOpen() {
		return err
	}
	s.interruptPrompt()
	from := s.dropAttempt()
	fmt.Fprintf(os.Stderr, "Request from %q dropped: %s\n", sanitize(from), sanitize(err.Error()))
	fmt.Println("Waiting for connections...")
	return nil
}

func (s *session) onSafetyHandshake(result safetyHandshakeResult) error {
	if result.chat != s.chat {
		return nil
	}
	s.handshakeExpired = nil
	if result.err != nil {
		return s.onInvalidRemoteMessage(result.err)
	}
	return s.askSafetyCode(result.joinerNonce, result.hostNonce)
}

func (s *session) askSafetyCode(joinerNonce, hostNonce []byte) error {
	if s.confirmed || s.question != "" {
		return nil
	}

	fingerprints, err := s.peer.DTLSFingerprints()
	if err != nil {
		return err
	}

	question := joinerQuestion
	hostFingerprint, joinerFingerprint := fingerprints.Remote, fingerprints.Local
	if !s.joining {
		question = fmt.Sprintf("Accept %q with this code? [y/N] ", sanitize(s.remoteID))
		hostFingerprint, joinerFingerprint = fingerprints.Local, fingerprints.Remote
	}
	code := safetyCode(hostFingerprint, joinerFingerprint, joinerNonce, hostNonce)

	fmt.Printf("Safety code: %s\n", code)
	s.prompt(question)
	return nil
}

func (s *session) interruptPrompt() {
	if s.question != "" {
		fmt.Println()
		s.question = ""
	}
}

func (s *session) prompt(question string) {
	s.question = question
	fmt.Print(question)
	s.promptedAt = time.Now()
}

func (s *session) onAnswer(line stdinLine) error {
	if s.question == "" || line.readAt.Before(s.promptedAt) {
		return nil
	}
	if line.readAt.Before(s.promptedAt.Add(s.answerGrace)) {
		fmt.Print("Answer ignored: too fast. " + s.question)
		return nil
	}
	if !isAcceptAnswer(line.text) {
		return s.refuse()
	}

	s.question = ""
	s.confirmed = true
	s.confirmExpired = nil
	s.chat.confirmLocal()
	return nil
}

func (s *session) refuse() error {
	if s.joining {
		s.question = ""
		if err := s.send(signaling.TypeReject, s.remoteID, signaling.ErrorPayload{Message: safetyCodeNotConfirmedError}); err != nil {
			return err
		}
		return errors.New(safetyCodeNotConfirmedError)
	}

	from := s.dropAttempt()
	fmt.Printf("Request from %q refused.\n", sanitize(from))
	return s.send(signaling.TypeReject, from, signaling.ErrorPayload{Message: refusedByHostMessage})
}

func (s *session) onConfirmTimeout() error {
	if s.peer == nil || s.confirmed {
		s.confirmExpired = nil
		return nil
	}
	s.interruptPrompt()
	from := s.dropAttempt()
	fmt.Printf("Request from %q refused: timed out.\n", sanitize(from))
	return s.send(signaling.TypeReject, from, signaling.ErrorPayload{Message: confirmTimeoutMessage})
}

func (s *session) dropAttempt() string {
	from := s.remoteID
	if s.peer != nil {
		_ = s.peer.Close()
	}
	s.peer = nil
	s.chat = nil
	s.remoteID = ""
	s.question = ""
	s.confirmed = false
	s.confirmExpired = nil
	s.handshakeExpired = nil
	s.connected.Store(false)
	return from
}

func (s *session) chatOpen() bool {
	return s.chat != nil && s.chat.isOpen()
}

func isAcceptAnswer(line string) bool {
	answer := strings.TrimSpace(line)
	return answer == "y" || answer == "Y"
}

func (s *session) startPeer(remoteID string) error {
	p, err := peer.New(s.cfg)
	if err != nil {
		return err
	}

	s.remoteID = remoteID
	s.peer = p
	s.chat = &chat{
		handshake: safetyHandshake{joining: s.joining},
		report:    s.reportSafetyHandshake,
	}

	// Trickle ICE: each local candidate goes to the other side as soon as it appears.
	p.OnICECandidate(func(c webrtc.ICECandidateInit) {
		if err := s.send(signaling.TypeCandidate, remoteID, c); err != nil {
			fmt.Println("failed to send candidate:", err)
		}
	})

	// Handler that tracks and prints connection state changes.
	p.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer connection state changed: %s\n", state.String())

		select {
		case s.states <- peerState{peer: p, state: state}:
		case <-s.ctx.Done():
		}
	})

	return nil
}

func (s *session) reportSafetyHandshake(result safetyHandshakeResult) {
	select {
	case s.handshakes <- result:
	case <-s.ctx.Done():
	}
}

// fromRemote drops messages from anyone other than the current peer.
func (s *session) fromRemote(msg signaling.Message) bool {
	return s.peer != nil && msg.From == s.remoteID
}

func (s *session) send(t signaling.Type, to string, payload any) error {
	msg, err := signaling.New(t, s.selfID, to, payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(s.ctx, sendTimeout)
	defer cancel()

	return s.conn.Send(ctx, msg)
}
