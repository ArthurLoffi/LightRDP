package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v5"

	"LightRDP/internal/peer"
	"LightRDP/internal/signaling"
	"LightRDP/internal/transport"
)

const (
	sendTimeout        = 5 * time.Second
	hostConfirmTimeout = 60 * time.Second
	hostAnswerGrace    = time.Second

	hostBusyMessage       = "host busy"
	refusedByHostMessage  = "refused by host"
	confirmTimeoutMessage = "timed out waiting for host"
)

// session links signaling (transport) to the WebRTC connection (peer).
// Supports one P2P session at a time.
type session struct {
	ctx    context.Context
	conn   *transport.Conn
	cfg    webrtc.Configuration
	selfID string

	// Set before any pion callback fires.
	remoteID string
	peer     *peer.Peer

	connected atomic.Bool
	done      chan error

	lines          <-chan stdinLine
	confirmTimeout time.Duration
	answerGrace    time.Duration
	pendingID      string
	promptedAt     time.Time
	confirmExpired <-chan time.Time
}

func newSession(ctx context.Context, conn *transport.Conn, cfg webrtc.Configuration, selfID string) *session {
	return &session{
		ctx:    ctx,
		conn:   conn,
		cfg:    cfg,
		selfID: selfID,
		done:   make(chan error, 1),

		confirmTimeout: hostConfirmTimeout,
		answerGrace:    hostAnswerGrace,
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
	if err := s.startPeer(hostID); err != nil {
		return err
	}
	// The answerer receives the DataChannel created by the other side.
	s.peer.OnDataChannel(setupChannel)

	return s.send(signaling.TypeConnect, hostID, nil)
}

// loop handles server messages until the session ends.
func (s *session) loop() error {
	msgs := s.conn.Messages()
	lines := s.lines

	for {
		var answers <-chan stdinLine
		if s.peer == nil {
			answers = lines
		}

		select {
		case <-s.ctx.Done():
			return nil

		case err := <-s.done:
			return err

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
			return err
		}
		answer, err := s.peer.HandleOffer(offer)
		if err != nil {
			return err
		}
		return s.send(signaling.TypeAnswer, s.remoteID, answer)

	case signaling.TypeAnswer:
		if !s.fromRemote(msg) {
			return nil
		}
		var answer webrtc.SessionDescription
		if err := msg.Decode(&answer); err != nil {
			return err
		}
		return s.peer.HandleAnswer(answer)

	case signaling.TypeCandidate:
		if !s.fromRemote(msg) {
			return nil
		}
		var c webrtc.ICECandidateInit
		if err := msg.Decode(&c); err != nil {
			return err
		}
		return s.peer.AddICECandidate(c)

	case signaling.TypeReject:
		if !s.fromRemote(msg) || s.connected.Load() {
			return nil
		}
		var e signaling.ErrorPayload
		_ = msg.Decode(&e)
		return errors.New("connection refused: " + sanitize(e.Message))

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
	if s.peer != nil || s.pendingID != "" {
		return s.send(signaling.TypeReject, from, signaling.ErrorPayload{Message: hostBusyMessage})
	}

	s.pendingID = from
	s.confirmExpired = time.After(s.confirmTimeout)

	fmt.Printf("Connection request from %q\n", sanitize(from))
	fmt.Print("Accept? [y/N] ")
	s.promptedAt = time.Now()
	return nil
}

func (s *session) onAnswer(line stdinLine) error {
	if s.pendingID == "" || line.readAt.Before(s.promptedAt) {
		return nil
	}
	if line.readAt.Before(s.promptedAt.Add(s.answerGrace)) {
		fmt.Print("Answer ignored: too fast. Accept? [y/N] ")
		return nil
	}
	if !isAcceptAnswer(line.text) {
		from := s.clearPending()
		fmt.Printf("Request from %q refused.\n", sanitize(from))
		return s.send(signaling.TypeReject, from, signaling.ErrorPayload{Message: refusedByHostMessage})
	}
	return s.acceptPending()
}

func (s *session) onConfirmTimeout() error {
	from := s.clearPending()
	if from == "" {
		return nil
	}
	fmt.Printf("\nRequest from %q refused: timed out.\n", sanitize(from))
	return s.send(signaling.TypeReject, from, signaling.ErrorPayload{Message: confirmTimeoutMessage})
}

func (s *session) clearPending() string {
	from := s.pendingID
	s.pendingID = ""
	s.confirmExpired = nil
	return from
}

func isAcceptAnswer(line string) bool {
	answer := strings.TrimSpace(line)
	return answer == "y" || answer == "Y"
}

func (s *session) acceptPending() error {
	if err := s.startPeer(s.clearPending()); err != nil {
		return err
	}

	// The offerer creates the DataChannel.
	dc, err := s.peer.CreateDataChannel("chat")
	if err != nil {
		return err
	}
	setupChannel(dc)

	offer, err := s.peer.CreateOffer()
	if err != nil {
		return err
	}
	return s.send(signaling.TypeOffer, s.remoteID, offer)
}

func (s *session) startPeer(remoteID string) error {
	p, err := peer.New(s.cfg)
	if err != nil {
		return err
	}

	s.remoteID = remoteID
	s.peer = p

	// Trickle ICE: each local candidate goes to the other side as soon as it appears.
	p.OnICECandidate(func(c webrtc.ICECandidateInit) {
		if err := s.send(signaling.TypeCandidate, s.remoteID, c); err != nil {
			fmt.Println("failed to send candidate:", err)
		}
	})

	// Handler that tracks and prints connection state changes.
	p.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer connection state changed: %s\n", state.String())

		switch state {
		case webrtc.PeerConnectionStateConnected:
			s.connected.Store(true)
		// The state changes to failed after 30 seconds
		// without activity or on any other failure.
		case webrtc.PeerConnectionStateFailed:
			s.finish(errors.New("peer connection failed"))
		case webrtc.PeerConnectionStateClosed:
			s.finish(nil)
		}
	})

	return nil
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

func (s *session) finish(err error) {
	select {
	case s.done <- err:
	default:
	}
}
