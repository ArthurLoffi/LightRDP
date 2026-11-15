package signaling

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRelayable(t *testing.T) {
	cases := map[Type]bool{
		TypeConnect:   true,
		TypeOffer:     true,
		TypeAnswer:    true,
		TypeCandidate: true,
		TypeReject:    true,
		TypeRegister:  false,
		TypeError:     false,
		Type("other"): false,
	}

	for msgType, want := range cases {
		if got := Relayable(msgType); got != want {
			t.Errorf("Expected Relayable(%q) = %v, got: %v", msgType, want, got)
		}
	}
}

func TestNewWithNilPayload(t *testing.T) {
	msg, err := New(TypeConnect, "alice", "bob", nil)
	if err != nil {
		t.Fatalf("Expected message, got: %v", err)
	}

	if msg.Type != TypeConnect || msg.From != "alice" || msg.To != "bob" {
		t.Fatalf("Expected connect from alice to bob, got: %+v", msg)
	}
	if len(msg.Payload) != 0 {
		t.Fatalf("Expected empty payload, got: %s", msg.Payload)
	}
}

func TestNewWithPayload(t *testing.T) {
	msg, err := New(TypeReject, "alice", "bob", ErrorPayload{Message: "host ocupado"})
	if err != nil {
		t.Fatalf("Expected message, got: %v", err)
	}

	want := `{"message":"host ocupado"}`
	if string(msg.Payload) != want {
		t.Fatalf("Expected payload %s, got: %s", want, msg.Payload)
	}
}

func TestNewWithInvalidPayload(t *testing.T) {
	_, err := New(TypeOffer, "alice", "bob", make(chan int))
	if err == nil {
		t.Fatal("Expected marshal error, got: nil")
	}

	var unsupported *json.UnsupportedTypeError
	if !errors.As(err, &unsupported) {
		t.Fatalf("Expected wrapped *json.UnsupportedTypeError, got: %v", err)
	}
	if !strings.HasPrefix(err.Error(), "signaling: marshal offer payload") {
		t.Fatalf("Expected error to start with 'signaling: marshal offer payload', got: %v", err)
	}
}

func TestNewError(t *testing.T) {
	msg := NewError("bob", "id vazio")

	if msg.Type != TypeError {
		t.Fatalf("Expected type error, got: %q", msg.Type)
	}
	if msg.From != "" {
		t.Fatalf("Expected empty From, got: %q", msg.From)
	}
	if msg.To != "bob" {
		t.Fatalf("Expected To 'bob', got: %q", msg.To)
	}

	var payload ErrorPayload
	if err := msg.Decode(&payload); err != nil {
		t.Fatalf("Expected decoded payload, got: %v", err)
	}
	if payload.Message != "id vazio" {
		t.Fatalf("Expected message 'id vazio', got: %q", payload.Message)
	}
}

func TestDecodeEmptyPayload(t *testing.T) {
	msg := Message{Type: TypeOffer}

	var payload ErrorPayload
	err := msg.Decode(&payload)
	if err == nil {
		t.Fatal("Expected error for empty payload, got: nil")
	}
	if err.Error() != "signaling: offer sem payload" {
		t.Fatalf("Expected 'signaling: offer sem payload', got: %v", err)
	}
}

func TestDecodeInvalidPayload(t *testing.T) {
	msg := Message{Type: TypeAnswer, Payload: json.RawMessage(`{"message":`)}

	var payload ErrorPayload
	err := msg.Decode(&payload)
	if err == nil {
		t.Fatal("Expected error for invalid payload, got: nil")
	}
	if !strings.HasPrefix(err.Error(), "signaling: decode answer payload") {
		t.Fatalf("Expected error to start with 'signaling: decode answer payload', got: %v", err)
	}
}

func TestDecodeValidPayload(t *testing.T) {
	msg := Message{Type: TypeReject, Payload: json.RawMessage(`{"message":"recusado"}`)}

	var payload ErrorPayload
	if err := msg.Decode(&payload); err != nil {
		t.Fatalf("Expected decoded payload, got: %v", err)
	}
	if payload.Message != "recusado" {
		t.Fatalf("Expected message 'recusado', got: %q", payload.Message)
	}
}

func TestMessageJSONOmitsEmptyFields(t *testing.T) {
	b, err := json.Marshal(Message{Type: TypeRegister})
	if err != nil {
		t.Fatalf("Expected JSON, got: %v", err)
	}

	want := `{"type":"register"}`
	if string(b) != want {
		t.Fatalf("Expected %s, got: %s", want, b)
	}
}

func TestMessageJSONWithAllFields(t *testing.T) {
	msg := Message{Type: TypeOffer, From: "alice", To: "bob", Payload: json.RawMessage(`{"sdp":"x"}`)}

	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Expected JSON, got: %v", err)
	}

	want := `{"type":"offer","from":"alice","to":"bob","payload":{"sdp":"x"}}`
	if string(b) != want {
		t.Fatalf("Expected %s, got: %s", want, b)
	}
}
