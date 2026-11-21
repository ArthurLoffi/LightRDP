package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"LightRDP/internal/signaling"
)

const testTimeout = 5 * time.Second

func captureStderr(t *testing.T) *os.File {
	t.Helper()

	file, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatalf("Expected temp file, got: %v", err)
	}

	original := os.Stderr
	os.Stderr = file
	t.Cleanup(func() {
		os.Stderr = original
		file.Close()
	})

	return file
}

func readCaptured(t *testing.T, file *os.File) string {
	t.Helper()

	b, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("Expected captured output, got: %v", err)
	}
	return string(b)
}

func TestNewLoggerWithoutDebug(t *testing.T) {
	stderr := captureStderr(t)

	log := newLogger(false)
	log.Debug("debug message")
	log.Info("info message")

	out := readCaptured(t, stderr)
	if strings.Contains(out, "debug message") {
		t.Fatalf("Expected no debug log, got: %q", out)
	}
	if !strings.Contains(out, "level=INFO") || !strings.Contains(out, "info message") {
		t.Fatalf("Expected info log, got: %q", out)
	}
}

func TestNewLoggerWithDebug(t *testing.T) {
	stderr := captureStderr(t)

	log := newLogger(true)
	log.Debug("debug message")

	out := readCaptured(t, stderr)
	if !strings.Contains(out, "level=DEBUG") || !strings.Contains(out, "debug message") {
		t.Fatalf("Expected debug log, got: %q", out)
	}
}

func TestNewServerSettings(t *testing.T) {
	srv := newServer("127.0.0.1:9999", slog.New(slog.NewTextHandler(io.Discard, nil)))

	if srv.Addr != "127.0.0.1:9999" {
		t.Fatalf("Expected Addr '127.0.0.1:9999', got: %q", srv.Addr)
	}
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("Expected ReadHeaderTimeout 10s, got: %v", srv.ReadHeaderTimeout)
	}
	if srv.Handler == nil {
		t.Fatal("Expected a handler, got: nil")
	}
}

func TestNewServerRoutesWebSocket(t *testing.T) {
	srv := newServer("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)

	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	alice := dialServer(t, url)
	bob := dialServer(t, url)

	writeMessage(t, alice, signaling.Message{Type: signaling.TypeRegister, From: "alice"})
	writeMessage(t, bob, signaling.Message{Type: signaling.TypeRegister, From: "bob"})

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bobMessages := readInBackground(ctx, bob)
	writeMessage(t, alice, signaling.Message{Type: signaling.TypeConnect, To: "bob"})
	aliceMessages := readInBackground(ctx, alice)

	lastServerError := ""
	for {
		select {
		case result := <-bobMessages:
			if result.err != nil {
				t.Fatalf("Expected relayed message, got: %v", result.err)
			}
			if result.msg.Type != signaling.TypeConnect || result.msg.From != "alice" {
				t.Fatalf("Expected connect from alice, got: %+v", result.msg)
			}
			return
		case result := <-aliceMessages:
			if result.err != nil {
				t.Fatalf("Expected server reply to alice, got: %v", result.err)
			}
			lastServerError = serverErrorText(t, result.msg)
			if lastServerError != "target not found: bob" {
				t.Fatalf("Expected 'target not found: bob', got: %q", lastServerError)
			}
			writeMessage(t, alice, signaling.Message{Type: signaling.TypeConnect, To: "bob"})
			aliceMessages = readInBackground(ctx, alice)
		case <-ctx.Done():
			t.Fatalf("Expected connect from alice before timeout, got: %v (last server error to alice: %q)", ctx.Err(), lastServerError)
		}
	}
}

func TestNewServerUnknownPath(t *testing.T) {
	srv := newServer("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)

	for _, path := range []string{"/", "/other", "/ws/extra"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("Expected HTTP response for %s, got: %v", path, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Expected 404 for %s, got: %d", path, resp.StatusCode)
		}
	}
}

func dialServer(t *testing.T, url string) *websocket.Conn {
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

func writeMessage(t *testing.T, conn *websocket.Conn, msg signaling.Message) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := wsjson.Write(ctx, conn, msg); err != nil {
		t.Fatalf("Expected message sent, got: %v", err)
	}
}

type readResult struct {
	msg signaling.Message
	err error
}

func readInBackground(ctx context.Context, conn *websocket.Conn) <-chan readResult {
	results := make(chan readResult, 1)
	go func() {
		var msg signaling.Message
		err := wsjson.Read(ctx, conn, &msg)
		results <- readResult{msg: msg, err: err}
	}()
	return results
}

func serverErrorText(t *testing.T, msg signaling.Message) string {
	t.Helper()

	if msg.Type != signaling.TypeError || msg.From != "" {
		t.Fatalf("Expected error message from the server, got: %+v", msg)
	}

	var payload signaling.ErrorPayload
	if err := msg.Decode(&payload); err != nil {
		t.Fatalf("Expected error payload, got: %v", err)
	}
	return payload.Message
}
