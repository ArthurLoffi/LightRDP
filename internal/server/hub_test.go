package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"LightRDP/internal/signaling"
)

const testTimeout = 5 * time.Second

func newTestHub(t *testing.T) (*Hub, string) {
	t.Helper()

	hub := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(hub)
	t.Cleanup(srv.Close)

	return hub, "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dialClient(t *testing.T, url string) *websocket.Conn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Expected WebSocket connection, got: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })

	return conn
}

func sendMessage(t *testing.T, conn *websocket.Conn, msg signaling.Message) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := wsjson.Write(ctx, conn, msg); err != nil {
		t.Fatalf("Expected message sent, got: %v", err)
	}
}

func readMessage(t *testing.T, conn *websocket.Conn) signaling.Message {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	var msg signaling.Message
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		t.Fatalf("Expected message received, got: %v", err)
	}
	return msg
}

func readServerError(t *testing.T, conn *websocket.Conn) signaling.Message {
	t.Helper()

	msg := readMessage(t, conn)
	if msg.Type != signaling.TypeError {
		t.Fatalf("Expected error message, got: %+v", msg)
	}
	if msg.From != "" {
		t.Fatalf("Expected error from the server (empty From), got: %q", msg.From)
	}
	return msg
}

func errorText(t *testing.T, msg signaling.Message) string {
	t.Helper()

	var payload signaling.ErrorPayload
	if err := msg.Decode(&payload); err != nil {
		t.Fatalf("Expected error payload, got: %v", err)
	}
	return payload.Message
}

func waitRegistered(t *testing.T, hub *Hub, id string, want bool) {
	t.Helper()

	deadline := time.Now().Add(testTimeout)
	for {
		if _, ok := hub.lookup(id); ok == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Expected registered(%q) = %v before timeout", id, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func registerClient(t *testing.T, hub *Hub, url, id string) *websocket.Conn {
	t.Helper()

	conn := dialClient(t, url)
	sendMessage(t, conn, signaling.Message{Type: signaling.TypeRegister, From: id})
	waitRegistered(t, hub, id, true)

	return conn
}

func TestRelayBetweenRegisteredClients(t *testing.T) {
	hub, url := newTestHub(t)
	alice := registerClient(t, hub, url, "alice")
	bob := registerClient(t, hub, url, "bob")

	sendMessage(t, alice, signaling.Message{Type: signaling.TypeConnect, To: "bob"})

	got := readMessage(t, bob)
	if got.Type != signaling.TypeConnect || got.From != "alice" || got.To != "bob" {
		t.Fatalf("Expected connect from alice to bob, got: %+v", got)
	}
}

func TestRelayKeepsPayload(t *testing.T) {
	hub, url := newTestHub(t)
	alice := registerClient(t, hub, url, "alice")
	bob := registerClient(t, hub, url, "bob")

	payload := json.RawMessage(`{"type":"offer","sdp":"v=0"}`)
	sendMessage(t, alice, signaling.Message{Type: signaling.TypeOffer, To: "bob", Payload: payload})

	got := readMessage(t, bob)
	if string(got.Payload) != string(payload) {
		t.Fatalf("Expected payload %s, got: %s", payload, got.Payload)
	}
}

func TestRelayOverwritesFrom(t *testing.T) {
	hub, url := newTestHub(t)
	alice := registerClient(t, hub, url, "alice")
	bob := registerClient(t, hub, url, "bob")

	sendMessage(t, alice, signaling.Message{Type: signaling.TypeConnect, From: "mallory", To: "bob"})

	got := readMessage(t, bob)
	if got.From != "alice" {
		t.Fatalf("Expected From 'alice', got: %q", got.From)
	}
}

func TestRegisterWithEmptyID(t *testing.T) {
	_, url := newTestHub(t)
	conn := dialClient(t, url)

	sendMessage(t, conn, signaling.Message{Type: signaling.TypeRegister})

	if text := errorText(t, readServerError(t, conn)); text != "empty id" {
		t.Fatalf("Expected 'empty id', got: %q", text)
	}
}

func TestRegisterWithIDInUse(t *testing.T) {
	hub, url := newTestHub(t)
	registerClient(t, hub, url, "alice")
	other := dialClient(t, url)

	sendMessage(t, other, signaling.Message{Type: signaling.TypeRegister, From: "alice"})

	msg := readServerError(t, other)
	if text := errorText(t, msg); text != "id already registered" {
		t.Fatalf("Expected 'id already registered', got: %q", text)
	}
	if msg.To != "alice" {
		t.Fatalf("Expected To 'alice', got: %q", msg.To)
	}
}

func TestRegisterTwiceOnSameConnection(t *testing.T) {
	hub, url := newTestHub(t)
	conn := registerClient(t, hub, url, "alice")

	sendMessage(t, conn, signaling.Message{Type: signaling.TypeRegister, From: "alice2"})

	if text := errorText(t, readServerError(t, conn)); text != "connection already registered" {
		t.Fatalf("Expected 'connection already registered', got: %q", text)
	}
	if _, ok := hub.lookup("alice2"); ok {
		t.Fatal("Expected 'alice2' not registered")
	}
}

func TestSendBeforeRegister(t *testing.T) {
	hub, url := newTestHub(t)
	registerClient(t, hub, url, "bob")
	conn := dialClient(t, url)

	sendMessage(t, conn, signaling.Message{Type: signaling.TypeConnect, To: "bob"})

	msg := readServerError(t, conn)
	if text := errorText(t, msg); text != "register before sending messages" {
		t.Fatalf("Expected 'register before sending messages', got: %q", text)
	}
	if msg.To != "" {
		t.Fatalf("Expected empty To, got: %q", msg.To)
	}
}

func TestSendTypeNotAllowed(t *testing.T) {
	for _, msgType := range []signaling.Type{signaling.TypeError, signaling.Type("unknown")} {
		t.Run(string(msgType), func(t *testing.T) {
			hub, url := newTestHub(t)
			alice := registerClient(t, hub, url, "alice")
			bob := registerClient(t, hub, url, "bob")

			sendMessage(t, alice, signaling.Message{Type: msgType, To: "bob"})

			msg := readServerError(t, alice)
			want := "message type not allowed: " + string(msgType)
			if text := errorText(t, msg); text != want {
				t.Fatalf("Expected %q, got: %q", want, text)
			}
			if msg.To != "alice" {
				t.Fatalf("Expected To 'alice', got: %q", msg.To)
			}

			assertNoMessage(t, bob)
		})
	}
}

func TestSendToUnknownID(t *testing.T) {
	hub, url := newTestHub(t)
	alice := registerClient(t, hub, url, "alice")

	sendMessage(t, alice, signaling.Message{Type: signaling.TypeConnect, To: "nobody"})

	msg := readServerError(t, alice)
	if text := errorText(t, msg); text != "target not found: nobody" {
		t.Fatalf("Expected 'target not found: nobody', got: %q", text)
	}
	if msg.To != "alice" {
		t.Fatalf("Expected To 'alice', got: %q", msg.To)
	}
}

func TestDisconnectRemovesID(t *testing.T) {
	hub, url := newTestHub(t)
	alice := registerClient(t, hub, url, "alice")

	alice.Close(websocket.StatusNormalClosure, "")
	waitRegistered(t, hub, "alice", false)

	again := registerClient(t, hub, url, "alice")
	bob := registerClient(t, hub, url, "bob")

	sendMessage(t, bob, signaling.Message{Type: signaling.TypeConnect, To: "alice"})

	got := readMessage(t, again)
	if got.Type != signaling.TypeConnect || got.From != "bob" {
		t.Fatalf("Expected connect from bob, got: %+v", got)
	}
}

func TestUnregisterKeepsIDOfOtherConnection(t *testing.T) {
	hub := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))

	current := &Client{hub: hub}
	if err := hub.register(current, "alice"); err != nil {
		t.Fatalf("Expected registration, got: %v", err)
	}

	stale := &Client{hub: hub, id: "alice"}
	hub.unregister(stale)

	got, ok := hub.lookup("alice")
	if !ok || got != current {
		t.Fatalf("Expected 'alice' to still point to the current connection, got: %v, %v", got, ok)
	}
}

func TestUnregisterWithoutID(t *testing.T) {
	hub := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))

	registered := &Client{hub: hub}
	if err := hub.register(registered, "alice"); err != nil {
		t.Fatalf("Expected registration, got: %v", err)
	}

	hub.unregister(&Client{hub: hub})

	if _, ok := hub.lookup("alice"); !ok {
		t.Fatal("Expected 'alice' still registered")
	}
}

func TestMessageTooLargeClosesConnection(t *testing.T) {
	hub, url := newTestHub(t)
	conn := registerClient(t, hub, url, "alice")

	big := signaling.Message{
		Type:    signaling.TypeOffer,
		To:      "bob",
		Payload: json.RawMessage(`"` + strings.Repeat("a", maxMsgSize) + `"`),
	}
	sendMessage(t, conn, big)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	var msg signaling.Message
	err := wsjson.Read(ctx, conn, &msg)
	if err == nil {
		t.Fatalf("Expected connection closed, got message: %+v", msg)
	}
	if status := websocket.CloseStatus(err); status != websocket.StatusMessageTooBig {
		t.Fatalf("Expected close status %v, got: %v (%v)", websocket.StatusMessageTooBig, status, err)
	}

	waitRegistered(t, hub, "alice", false)
}

func assertNoMessage(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	var msg signaling.Message
	if err := wsjson.Read(ctx, conn, &msg); err == nil {
		t.Fatalf("Expected no message, got: %+v", msg)
	}
}
