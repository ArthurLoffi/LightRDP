package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/pion/webrtc/v5"

	"LightRDP/internal/server"
	"LightRDP/internal/signaling"
	"LightRDP/internal/transport"
)

const testTimeout = 5 * time.Second

func newTestHub(t *testing.T) string {
	t.Helper()

	hub := server.NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(hub)
	t.Cleanup(srv.Close)

	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dialHub(t *testing.T, url string) *transport.Conn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, err := transport.Dial(ctx, url)
	if err != nil {
		t.Fatalf("Expected connection to hub, got: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return conn
}

func newTestSession(t *testing.T, url, id string) *session {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	s := newSession(ctx, dialHub(t, url), webrtc.Configuration{}, id)
	t.Cleanup(s.close)

	if err := s.register(); err != nil {
		t.Fatalf("Expected register sent, got: %v", err)
	}
	waitSelfEcho(t, s.conn, id)

	return s
}

func waitSelfEcho(t *testing.T, conn *transport.Conn, id string) {
	t.Helper()

	echo, err := signaling.New(signaling.TypeReject, id, id, nil)
	if err != nil {
		t.Fatalf("Expected echo message, got: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := conn.Send(ctx, echo); err != nil {
		t.Fatalf("Expected echo sent, got: %v", err)
	}

	msg := nextMessage(t, conn)
	if msg.Type != signaling.TypeReject || msg.From != id {
		t.Fatalf("Expected own echo after register, got: %+v", msg)
	}
}

func nextMessage(t *testing.T, conn *transport.Conn) signaling.Message {
	t.Helper()

	select {
	case msg, ok := <-conn.Messages():
		if !ok {
			t.Fatalf("Expected message, connection closed: %v", conn.Err())
		}
		return msg
	case <-time.After(testTimeout):
		t.Fatal("Expected message before timeout")
		return signaling.Message{}
	}
}

func registerProbe(t *testing.T, url, id string) *transport.Conn {
	t.Helper()

	conn := dialHub(t, url)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := conn.Send(ctx, signaling.Message{Type: signaling.TypeRegister, From: id}); err != nil {
		t.Fatalf("Expected register sent, got: %v", err)
	}
	waitSelfEcho(t, conn, id)

	return conn
}

func errorPayload(t *testing.T, text string) json.RawMessage {
	t.Helper()

	b, err := json.Marshal(signaling.ErrorPayload{Message: text})
	if err != nil {
		t.Fatalf("Expected payload, got: %v", err)
	}
	return b
}

func TestHandleIgnoresMessagesFromOtherPeers(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")

	if err := s.startPeer("bob"); err != nil {
		t.Fatalf("Expected peer started, got: %v", err)
	}

	for _, msgType := range []signaling.Type{signaling.TypeOffer, signaling.TypeAnswer, signaling.TypeCandidate, signaling.TypeReject} {
		msg := signaling.Message{Type: msgType, From: "eve", To: "alice", Payload: json.RawMessage(`"invalid"`)}
		if err := s.handle(msg); err != nil {
			t.Fatalf("Expected %s from 'eve' ignored, got: %v", msgType, err)
		}
	}
}

func TestHandleIgnoresMessagesWithoutPeer(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")

	for _, msgType := range []signaling.Type{signaling.TypeOffer, signaling.TypeAnswer, signaling.TypeCandidate, signaling.TypeReject} {
		msg := signaling.Message{Type: msgType, From: "bob", To: "alice", Payload: json.RawMessage(`"invalid"`)}
		if err := s.handle(msg); err != nil {
			t.Fatalf("Expected %s without peer ignored, got: %v", msgType, err)
		}
	}
}

func TestHandleIgnoresErrorFromClient(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")

	msg := signaling.Message{Type: signaling.TypeError, From: "eve", To: "alice", Payload: errorPayload(t, "falso")}
	if err := s.handle(msg); err != nil {
		t.Fatalf("Expected error from client ignored, got: %v", err)
	}
}

func TestHandleServerErrorBeforeConnect(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")

	err := s.handle(signaling.NewError("alice", "destino não encontrado: \x1b[31mbob"))
	if err == nil {
		t.Fatal("Expected session error, got: nil")
	}
	if err.Error() != "servidor: destino não encontrado: [31mbob" {
		t.Fatalf("Expected 'servidor: destino não encontrado: [31mbob', got: %q", err.Error())
	}
}

func TestHandleServerErrorAfterConnect(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")
	s.connected.Store(true)

	if err := s.handle(signaling.NewError("alice", "aviso")); err != nil {
		t.Fatalf("Expected server error after connect not to end the session, got: %v", err)
	}
}

func TestHandleRejectBeforeConnect(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")

	if err := s.startPeer("bob"); err != nil {
		t.Fatalf("Expected peer started, got: %v", err)
	}

	msg := signaling.Message{Type: signaling.TypeReject, From: "bob", To: "alice", Payload: errorPayload(t, "host ocupado")}
	err := s.handle(msg)
	if err == nil {
		t.Fatal("Expected session error, got: nil")
	}
	if err.Error() != "conexão recusada: host ocupado" {
		t.Fatalf("Expected 'conexão recusada: host ocupado', got: %q", err.Error())
	}
}

func TestHandleRejectAfterConnect(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")

	if err := s.startPeer("bob"); err != nil {
		t.Fatalf("Expected peer started, got: %v", err)
	}
	s.connected.Store(true)

	msg := signaling.Message{Type: signaling.TypeReject, From: "bob", To: "alice", Payload: errorPayload(t, "host ocupado")}
	if err := s.handle(msg); err != nil {
		t.Fatalf("Expected reject after connect ignored, got: %v", err)
	}
}

func TestHandleUnknownType(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")

	if err := s.handle(signaling.Message{Type: signaling.Type("unknown"), From: "bob"}); err != nil {
		t.Fatalf("Expected unknown type not to end the session, got: %v", err)
	}
}

func TestHandleConnectWhileBusy(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	first := registerProbe(t, url, "first")
	second := registerProbe(t, url, "second")

	if err := host.handle(signaling.Message{Type: signaling.TypeConnect, From: "first", To: "host"}); err != nil {
		t.Fatalf("Expected first connect accepted, got: %v", err)
	}
	if offer := nextMessage(t, first); offer.Type != signaling.TypeOffer || offer.From != "host" {
		t.Fatalf("Expected offer from host, got: %+v", offer)
	}

	if err := host.handle(signaling.Message{Type: signaling.TypeConnect, From: "second", To: "host"}); err != nil {
		t.Fatalf("Expected second connect rejected without error, got: %v", err)
	}

	reject := nextMessage(t, second)
	if reject.Type != signaling.TypeReject || reject.From != "host" {
		t.Fatalf("Expected reject from host, got: %+v", reject)
	}
	var payload signaling.ErrorPayload
	if err := reject.Decode(&payload); err != nil {
		t.Fatalf("Expected reject payload, got: %v", err)
	}
	if payload.Message != "host ocupado" {
		t.Fatalf("Expected 'host ocupado', got: %q", payload.Message)
	}
	if host.remoteID != "first" {
		t.Fatalf("Expected remote to stay 'first', got: %q", host.remoteID)
	}
}

func TestLoopReturnsOnContextCancel(t *testing.T) {
	url := newTestHub(t)

	ctx, cancel := context.WithCancel(context.Background())
	s := newSession(ctx, dialHub(t, url), webrtc.Configuration{}, "alice")

	result := make(chan error, 1)
	go func() { result <- s.loop() }()
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Expected nil on context cancel, got: %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("Expected loop to return before timeout")
	}
}

func TestLoopReturnsWhenServerDisconnects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		conn.Close(websocket.StatusGoingAway, "")
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s := newSession(ctx, dialHub(t, "ws"+strings.TrimPrefix(srv.URL, "http")), webrtc.Configuration{}, "alice")

	result := make(chan error, 1)
	go func() { result <- s.loop() }()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("Expected error on server disconnect, got: nil")
		}
		if !strings.HasPrefix(err.Error(), "servidor de sinalização desconectado: ") {
			t.Fatalf("Expected 'servidor de sinalização desconectado: ...', got: %q", err.Error())
		}
		if websocket.CloseStatus(err) != websocket.StatusGoingAway {
			t.Fatalf("Expected wrapped close status %v, got: %v", websocket.StatusGoingAway, err)
		}
	case <-time.After(testTimeout):
		t.Fatal("Expected loop to return before timeout")
	}
}

func TestHostAndJoinConnect(t *testing.T) {
	url := newTestHub(t)

	host := helperCommand(t, "host", helperServerEnv+"="+url)
	hostOut := startHelper(t, host)
	waitForLine(t, hostOut, "ready")

	join := helperCommand(t, "join", helperServerEnv+"="+url)
	joinOut := startHelper(t, join)

	waitForLine(t, hostOut, `Pedido de conexão de "join1"`)
	waitForLine(t, hostOut, "Peer connection state changed: connected")
	waitForLine(t, joinOut, "Peer connection state changed: connected")

	stopHelper(host)
	stopHelper(join)

	for name, cmd := range map[string]*exec.Cmd{"host": host, "join": join} {
		if out := cmd.Stderr.(*bytes.Buffer).String(); strings.Contains(out, "DATA RACE") {
			t.Fatalf("Expected no data race in %s helper, got: %s", name, out)
		}
	}
}

func runSessionHelper(t *testing.T, selfID, hostID string) {
	ctx, cancel := context.WithTimeout(context.Background(), helperTimeout)
	defer cancel()

	conn, err := transport.Dial(ctx, os.Getenv(helperServerEnv))
	if err != nil {
		t.Fatalf("Expected connection to hub, got: %v", err)
	}
	defer conn.Close()

	s := newSession(ctx, conn, webrtc.Configuration{}, selfID)
	defer s.close()

	if err := s.register(); err != nil {
		t.Fatalf("Expected register sent, got: %v", err)
	}
	waitSelfEcho(t, conn, selfID)
	fmt.Println("ready")

	if hostID != "" {
		if err := s.join(hostID); err != nil {
			t.Fatalf("Expected join sent, got: %v", err)
		}
	}

	if err := s.loop(); err != nil {
		t.Fatalf("Expected session loop to end cleanly, got: %v", err)
	}
}
