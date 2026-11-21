package client

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

	conn, err := transport.Dial(ctx, url, nil)
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

	msg := signaling.Message{Type: signaling.TypeError, From: "eve", To: "alice", Payload: errorPayload(t, "fake")}
	if err := s.handle(msg); err != nil {
		t.Fatalf("Expected error from client ignored, got: %v", err)
	}
}

func TestHandleServerErrorBeforeConnect(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")

	err := s.handle(signaling.NewError("alice", "target not found: \x1b[31mbob"))
	if err == nil {
		t.Fatal("Expected session error, got: nil")
	}
	if err.Error() != "server: target not found: [31mbob" {
		t.Fatalf("Expected 'server: target not found: [31mbob', got: %q", err.Error())
	}
}

func TestHandleServerErrorAfterConnect(t *testing.T) {
	url := newTestHub(t)
	s := newTestSession(t, url, "alice")
	s.connected.Store(true)

	if err := s.handle(signaling.NewError("alice", "notice")); err != nil {
		t.Fatalf("Expected server error after connect not to end the session, got: %v", err)
	}
}

func newJoiningSession(t *testing.T) *session {
	t.Helper()

	url := newTestHub(t)
	registerProbe(t, url, "bob")
	s := newTestSession(t, url, "alice")
	if err := s.join("bob"); err != nil {
		t.Fatalf("Expected join sent, got: %v", err)
	}
	return s
}

func TestHandleRejectBeforeConnect(t *testing.T) {
	s := newJoiningSession(t)

	msg := signaling.Message{Type: signaling.TypeReject, From: "bob", To: "alice", Payload: errorPayload(t, "host busy")}
	err := s.handle(msg)
	if err == nil {
		t.Fatal("Expected session error, got: nil")
	}
	if err.Error() != "connection refused: host busy" {
		t.Fatalf("Expected 'connection refused: host busy', got: %q", err.Error())
	}
}

func TestHandleRejectWhileSafetyCodePending(t *testing.T) {
	s := newJoiningSession(t)
	s.connected.Store(true)
	s.prompt(joinerQuestion)

	msg := signaling.Message{Type: signaling.TypeReject, From: "bob", To: "alice", Payload: errorPayload(t, "refused by host")}
	err := s.handle(msg)
	if err == nil || err.Error() != "connection refused: refused by host" {
		t.Fatalf("Expected 'connection refused: refused by host', got: %v", err)
	}
}

func TestHandleRejectAfterLocalConfirmation(t *testing.T) {
	s := newJoiningSession(t)
	s.connected.Store(true)
	s.prompt(joinerQuestion)
	if err := s.onAnswer(answerAfterGrace(s, "y\n")); err != nil {
		t.Fatalf("Expected accept, got: %v", err)
	}

	msg := signaling.Message{Type: signaling.TypeReject, From: "bob", To: "alice", Payload: errorPayload(t, "refused by host")}
	err := s.handle(msg)
	if err == nil || err.Error() != "connection refused: refused by host" {
		t.Fatalf("Expected 'connection refused: refused by host', got: %v", err)
	}
}

func TestHandleRejectAfterChatOpen(t *testing.T) {
	s := newJoiningSession(t)
	s.connected.Store(true)
	s.chat.open = true

	msg := signaling.Message{Type: signaling.TypeReject, From: "bob", To: "alice", Payload: errorPayload(t, "host busy")}
	if err := s.handle(msg); err != nil {
		t.Fatalf("Expected reject after chat open ignored, got: %v", err)
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
		t.Fatalf("Expected first negotiation started, got: %v", err)
	}
	expectOffer(t, first)
	host.prompt("Accept \"first\" with this code? [y/N] ")
	if err := host.onAnswer(answerAfterGrace(host, "y\n")); err != nil {
		t.Fatalf("Expected first confirmed, got: %v", err)
	}

	if err := host.handle(signaling.Message{Type: signaling.TypeConnect, From: "second", To: "host"}); err != nil {
		t.Fatalf("Expected second connect rejected without error, got: %v", err)
	}

	reject := nextMessageOtherThan(t, second, signaling.TypeCandidate)
	if reject.Type != signaling.TypeReject || reject.From != "host" {
		t.Fatalf("Expected reject from host, got: %+v", reject)
	}
	var payload signaling.ErrorPayload
	if err := reject.Decode(&payload); err != nil {
		t.Fatalf("Expected reject payload, got: %v", err)
	}
	if payload.Message != "host busy" {
		t.Fatalf("Expected 'host busy', got: %q", payload.Message)
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
		if !strings.HasPrefix(err.Error(), "signaling server disconnected: ") {
			t.Fatalf("Expected 'signaling server disconnected: ...', got: %q", err.Error())
		}
		if websocket.CloseStatus(err) != websocket.StatusGoingAway {
			t.Fatalf("Expected wrapped close status %v, got: %v", websocket.StatusGoingAway, err)
		}
	case <-time.After(testTimeout):
		t.Fatal("Expected loop to return before timeout")
	}
}

type sessionHelper struct {
	cmd *exec.Cmd
	in  io.Writer
	out <-chan string
}

func startSessionHelper(t *testing.T, url, mode, id string) *sessionHelper {
	t.Helper()

	cmd := helperCommand(t, mode, helperServerEnv+"="+url, helperIDEnv+"="+id)
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Expected stdin pipe, got: %v", err)
	}
	out := startHelper(t, cmd)
	waitForLine(t, out, "ready")
	return &sessionHelper{cmd: cmd, in: in, out: out}
}

func (h *sessionHelper) write(t *testing.T, text string) {
	t.Helper()

	if _, err := io.WriteString(h.in, text); err != nil {
		t.Fatalf("Expected %q written, got: %v", text, err)
	}
}

func (h *sessionHelper) answerAfterGrace(t *testing.T, text string) {
	t.Helper()

	time.Sleep(hostAnswerGrace + 200*time.Millisecond)
	h.write(t, text)
}

func (h *sessionHelper) stop(t *testing.T) string {
	t.Helper()

	stopHelper(h.cmd)
	stderr := h.cmd.Stderr.(*bytes.Buffer).String()
	if strings.Contains(stderr, "DATA RACE") {
		t.Fatalf("Expected no data race in helper, got: %s", stderr)
	}
	return stderr
}

func waitSafetyCode(t *testing.T, h *sessionHelper) string {
	t.Helper()

	waitForLine(t, h.out, "Peer connection state changed: connected")
	line := waitForPrefix(t, h.out, "Safety code: ")
	code := strings.TrimPrefix(line, "Safety code: ")
	if !safetyCodeFormat.MatchString(code) {
		t.Fatalf("Expected 'NNN NNN' code, got: %q", line)
	}
	return code
}

func startHostAndJoin(t *testing.T, joinID string) (url string, host, join *sessionHelper) {
	t.Helper()

	url = newTestHub(t)
	host = startSessionHelper(t, url, "host", "")
	join = startSessionHelper(t, url, "join", joinID)

	waitForLine(t, host.out, `Connection request from "`+joinID+`"`)
	hostCode := waitSafetyCode(t, host)
	joinCode := waitSafetyCode(t, join)
	if hostCode != joinCode {
		t.Fatalf("Expected same safety code on both sides, got: host %q, join %q", hostCode, joinCode)
	}
	return url, host, join
}

func TestHostAndJoinConnect(t *testing.T) {
	_, host, join := startHostAndJoin(t, "join1")

	host.answerAfterGrace(t, "y\n")
	host.write(t, "typed before both confirmed\n")
	join.answerAfterGrace(t, "y\n")

	waitForLine(t, host.out, `Accept "join1" with this code? [y/N]`)
	waitForLine(t, host.out, "DataChannel 'chat' open! Type messages:")
	waitForLine(t, join.out, "Same code on the host? [y/N]")
	waitForLine(t, join.out, "DataChannel 'chat' open! Type messages:")

	host.write(t, "hello\n")
	seen := waitForLine(t, join.out, "< hello")
	join.write(t, "back\n")
	seen = append(seen, waitForLine(t, host.out, "< back")...)

	for _, line := range seen {
		if strings.HasPrefix(line, "< ") || strings.Contains(line, safetyCodeConfirmedMessage) {
			t.Fatalf("Expected no other chat text, got: %q", line)
		}
	}

	host.stop(t)
	join.stop(t)
}

func TestHostRefusesSafetyCode(t *testing.T) {
	url, host, join := startHostAndJoin(t, "join1")

	host.answerAfterGrace(t, "n\n")
	waitForLine(t, host.out, `Accept "join1" with this code? [y/N] Request from "join1" refused.`)
	waitForLine(t, join.out, "Same code on the host? [y/N]")
	waitForLine(t, join.out, "loop ended: connection refused: refused by host")

	next := startSessionHelper(t, url, "join", "join2")
	waitForLine(t, host.out, `Connection request from "join2"`)
	waitSafetyCode(t, host)
	waitSafetyCode(t, next)

	host.stop(t)
	join.stop(t)
	next.stop(t)
}

func TestJoinerRefusesSafetyCode(t *testing.T) {
	url, host, join := startHostAndJoin(t, "join1")

	join.answerAfterGrace(t, "n\n")
	waitForLine(t, join.out, "Same code on the host? [y/N] loop ended: safety code not confirmed")
	waitForLine(t, host.out, `Accept "join1" with this code? [y/N]`)

	next := startSessionHelper(t, url, "join", "join2")
	waitForLine(t, host.out, `Connection request from "join2"`)

	hostStderr := host.stop(t)
	join.stop(t)
	next.stop(t)

	if !strings.Contains(hostStderr, "error: connection refused: safety code not confirmed\n") {
		t.Fatalf("Expected host to show the refusal, got: %q", hostStderr)
	}
}

func TestJoinerRefusesAfterHostConfirmed(t *testing.T) {
	url, host, join := startHostAndJoin(t, "join1")

	host.answerAfterGrace(t, "y\n")
	join.answerAfterGrace(t, "n\n")
	waitForLine(t, join.out, "Same code on the host? [y/N] loop ended: safety code not confirmed")

	next := startSessionHelper(t, url, "join", "join2")
	waitForLine(t, host.out, `Accept "join1" with this code? [y/N] Connection request from "join2"`)

	hostStderr := host.stop(t)
	join.stop(t)
	next.stop(t)

	if !strings.Contains(hostStderr, "error: connection refused: safety code not confirmed\n") {
		t.Fatalf("Expected host to show the refusal, got: %q", hostStderr)
	}
}

func runSessionHelper(t *testing.T, selfID, hostID string) {
	ctx, cancel := context.WithTimeout(context.Background(), helperTimeout)
	defer cancel()

	conn, err := transport.Dial(ctx, os.Getenv(helperServerEnv), nil)
	if err != nil {
		t.Fatalf("Expected connection to hub, got: %v", err)
	}
	defer conn.Close()

	s := newSession(ctx, conn, webrtc.Configuration{}, selfID)
	defer s.close()
	s.lines = stdinLines()

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
		fmt.Println("loop ended:", err)
	}
}
