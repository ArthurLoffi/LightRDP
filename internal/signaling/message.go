// Package signaling define o protocolo trocado entre clientes e o servidor
// de sinalização. O servidor só lê o envelope (Type, From, To); o Payload
// é repassado sem ser decodificado.
package signaling

import (
	"encoding/json"
	"fmt"
)

type Type string

const (
	// Cliente -> servidor: registra o ID do remetente.
	TypeRegister Type = "register"
	// Cliente -> host: pede para iniciar uma sessão.
	TypeConnect Type = "connect"
	// Payload: webrtc.SessionDescription.
	TypeOffer  Type = "offer"
	TypeAnswer Type = "answer"
	// Payload: webrtc.ICECandidateInit.
	TypeCandidate Type = "candidate"
	// Host -> cliente: recusa o pedido de sessão. Payload: ErrorPayload.
	TypeReject Type = "reject"
	// Somente servidor -> cliente (From vazio). Payload: ErrorPayload.
	TypeError Type = "error"
)

// Relayable informa se um cliente pode enviar este tipo para outro cliente.
// register é tratado pelo servidor; error só o servidor pode emitir.
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

// New cria uma mensagem serializando o payload em JSON. payload pode ser nil.
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

// NewError cria uma mensagem de erro destinada a "to".
func NewError(to, text string) Message {
	b, _ := json.Marshal(ErrorPayload{Message: text})
	return Message{Type: TypeError, To: to, Payload: b}
}

// Decode decodifica o payload da mensagem em v.
func (m Message) Decode(v any) error {
	if len(m.Payload) == 0 {
		return fmt.Errorf("signaling: %s sem payload", m.Type)
	}
	if err := json.Unmarshal(m.Payload, v); err != nil {
		return fmt.Errorf("signaling: decode %s payload: %w", m.Type, err)
	}
	return nil
}
