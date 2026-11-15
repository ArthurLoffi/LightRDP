package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"LightRDP/internal/signaling"
)

func TestServeHTTPWithoutUpgrade(t *testing.T) {
	hub := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	hub.ServeHTTP(rec, req)

	if rec.Code < 400 {
		t.Fatalf("Expected an HTTP error status, got: %d", rec.Code)
	}

	hub.mu.RLock()
	clients := len(hub.clients)
	hub.mu.RUnlock()
	if clients != 0 {
		t.Fatalf("Expected no registered clients, got: %d", clients)
	}
}

func TestServeHTTPKeepsWorkingAfterFailedUpgrade(t *testing.T) {
	hub, url := newTestHub(t)

	resp, err := http.Get("http" + url[len("ws"):])
	if err != nil {
		t.Fatalf("Expected HTTP response, got: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Fatalf("Expected an HTTP error status, got: %d", resp.StatusCode)
	}

	alice := registerClient(t, hub, url, "alice")
	bob := registerClient(t, hub, url, "bob")
	sendMessage(t, alice, signaling.Message{Type: signaling.TypeConnect, To: "bob"})

	if got := readMessage(t, bob); got.From != "alice" {
		t.Fatalf("Expected connect from alice, got: %+v", got)
	}
}
