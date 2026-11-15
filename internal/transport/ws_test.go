package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"LightRDP/internal/signaling"
)

const testTimeout = 5 * time.Second

type testServer struct {
	url      string
	received chan signaling.Message
	conns    chan *websocket.Conn
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	ts := &testServer{
		received: make(chan signaling.Message, 64),
		conns:    make(chan *websocket.Conn, 1),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		ts.conns <- conn
		for {
			var msg signaling.Message
			if err := wsjson.Read(r.Context(), conn, &msg); err != nil {
				return
			}
			ts.received <- msg
		}
	}))
	t.Cleanup(srv.Close)

	ts.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	return ts
}

func (ts *testServer) serverConn(t *testing.T) *websocket.Conn {
	t.Helper()

	select {
	case conn := <-ts.conns:
		return conn
	case <-time.After(testTimeout):
		t.Fatal("Expected server connection before timeout")
		return nil
	}
}

func dialTestServer(t *testing.T, url string) *Conn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, err := Dial(ctx, url)
	if err != nil {
		t.Fatalf("Expected connection, got: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return conn
}

func waitClosed(t *testing.T, msgs <-chan signaling.Message) {
	t.Helper()

	deadline := time.After(testTimeout)
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			t.Fatalf("Expected Messages closed, got message: %+v", msg)
		case <-deadline:
			t.Fatal("Expected Messages closed before timeout")
		}
	}
}

func TestDialUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, err := Dial(ctx, url)
	if err == nil {
		conn.Close()
		t.Fatal("Expected dial error, got: nil")
	}
	if !strings.HasPrefix(err.Error(), "transport: dial "+url) {
		t.Fatalf("Expected error to start with 'transport: dial %s', got: %v", url, err)
	}
}

func TestSend(t *testing.T) {
	ts := newTestServer(t)
	conn := dialTestServer(t, ts.url)

	msg := signaling.Message{Type: signaling.TypeRegister, From: "alice"}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := conn.Send(ctx, msg); err != nil {
		t.Fatalf("Expected message sent, got: %v", err)
	}

	select {
	case got := <-ts.received:
		if got.Type != msg.Type || got.From != msg.From {
			t.Fatalf("Expected %+v, got: %+v", msg, got)
		}
	case <-time.After(testTimeout):
		t.Fatal("Expected message on the server before timeout")
	}
}

func TestMessages(t *testing.T) {
	ts := newTestServer(t)
	conn := dialTestServer(t, ts.url)
	server := ts.serverConn(t)

	want := signaling.NewError("alice", "id vazio")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := wsjson.Write(ctx, server, want); err != nil {
		t.Fatalf("Expected server message sent, got: %v", err)
	}

	select {
	case got, ok := <-conn.Messages():
		if !ok {
			t.Fatalf("Expected message, got closed channel: %v", conn.Err())
		}
		if got.Type != want.Type || got.To != want.To || string(got.Payload) != string(want.Payload) {
			t.Fatalf("Expected %+v, got: %+v", want, got)
		}
	case <-time.After(testTimeout):
		t.Fatal("Expected message before timeout")
	}
}

func TestServerCloseEndsMessages(t *testing.T) {
	ts := newTestServer(t)
	conn := dialTestServer(t, ts.url)
	server := ts.serverConn(t)

	server.Close(websocket.StatusGoingAway, "bye")

	waitClosed(t, conn.Messages())
	if conn.Err() == nil {
		t.Fatal("Expected Err after server close, got: nil")
	}
	if status := websocket.CloseStatus(conn.Err()); status != websocket.StatusGoingAway {
		t.Fatalf("Expected close status %v, got: %v", websocket.StatusGoingAway, status)
	}
}

func TestClose(t *testing.T) {
	ts := newTestServer(t)
	conn := dialTestServer(t, ts.url)
	ts.serverConn(t)

	if err := conn.Close(); err != nil {
		t.Fatalf("Expected clean close, got: %v", err)
	}

	waitClosed(t, conn.Messages())

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := conn.Send(ctx, signaling.Message{Type: signaling.TypeRegister}); err == nil {
		t.Fatal("Expected send error after close, got: nil")
	}
}

func TestConcurrentSend(t *testing.T) {
	ts := newTestServer(t)
	conn := dialTestServer(t, ts.url)

	const senders = 20

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	var wg sync.WaitGroup
	errs := make(chan error, senders)
	for range senders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- conn.Send(ctx, signaling.Message{Type: signaling.TypeCandidate, To: "bob"})
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Expected concurrent send, got: %v", err)
		}
	}

	for i := range senders {
		select {
		case <-ts.received:
		case <-time.After(testTimeout):
			t.Fatalf("Expected %d messages on the server, got: %d", senders, i)
		}
	}
}
