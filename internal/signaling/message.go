// Package signaling defines the protocol exchanged between clients and the
// signaling server. The server only reads the envelope (Type, From, To); the
// Payload is relayed without being decoded.
package signaling

import (
	"encoding/json"
	"fmt"
)

type Type string

const (
	// Client -> server: registers the sender ID.
	TypeRegister Type = "register"
	// Client -> host: asks to start a session.
	TypeConnect Type = "connect"
	// Payload: webrtc.SessionDescription.
	TypeOffer  Type = "offer"
	TypeAnswer Type = "answer"
	// Payload: webrtc.ICECandidateInit.
	TypeCandidate Type = "candidate"
	// Host -> client: refuses the session request. Payload: ErrorPayload.
	TypeReject Type = "reject"
	// Server -> client only (empty From). Payload: ErrorPayload.
	TypeError Type = "error"
)

// Relayable reports whether a client may send this type to another client.
// register is handled by the server; only the server may send error.
func Relayable(t Type) bool {
	switch t {
	case TypeConnect, TypeOffer, TypeAnswer, TypeCandidate, TypeReject:
		return true
	}
	return false
}

type Message struct {
	Type    Type            `json:"type"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

// New creates a message, serializing the payload as JSON. payload may be nil.
func New(t Type, from, to string, payload any) (Message, error) {
	msg := Message{Type: t, From: from, To: to}
	if payload == nil {
		return msg, nil
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("signaling: marshal %s payload: %w", t, err)
	}
	msg.Payload = b

	return msg, nil
}

// NewError creates an error message addressed to "to".
func NewError(to, text string) Message {
	b, _ := json.Marshal(ErrorPayload{Message: text})
	return Message{Type: TypeError, To: to, Payload: b}
}

// Decode decodes the message payload into v.
func (m Message) Decode(v any) error {
	if len(m.Payload) == 0 {
		return fmt.Errorf("signaling: %s has no payload", m.Type)
	}
	if err := json.Unmarshal(m.Payload, v); err != nil {
		return fmt.Errorf("signaling: decode %s payload: %w", m.Type, err)
	}
	return nil
}
