package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"LightRDP/internal/signaling"
)

func answerAfterGrace(s *session, text string) stdinLine {
	return stdinLine{text: text, readAt: s.promptedAt.Add(s.answerGrace)}
}

func connectMessage(from string) signaling.Message {
	return signaling.Message{Type: signaling.TypeConnect, From: from, To: "host"}
}

func expectReject(t *testing.T, conn interface {
	Messages() <-chan signaling.Message
}, want string) {
	t.Helper()

	var msg signaling.Message
	select {
	case got, ok := <-conn.Messages():
		if !ok {
			t.Fatal("Expected reject, connection closed")
		}
		msg = got
	case <-time.After(testTimeout):
		t.Fatal("Expected reject before timeout")
	}

	if msg.Type != signaling.TypeReject || msg.From != "host" {
		t.Fatalf("Expected reject from host, got: %+v", msg)
	}
	var payload signaling.ErrorPayload
	if err := msg.Decode(&payload); err != nil {
		t.Fatalf("Expected reject payload, got: %v", err)
	}
	if payload.Message != want {
		t.Fatalf("Expected %q, got: %q", want, payload.Message)
	}
}

func captureStdout(t *testing.T) *os.File {
	t.Helper()

	file, err := os.Create(filepath.Join(t.TempDir(), "stdout"))
	if err != nil {
		t.Fatalf("Expected temp file, got: %v", err)
	}

	original := os.Stdout
	os.Stdout = file
	t.Cleanup(func() {
		os.Stdout = original
		file.Close()
	})

	return file
}

func TestConnectWaitsForConfirmation(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	requester := registerProbe(t, url, "requester")

	if err := host.handle(connectMessage("requester")); err != nil {
		t.Fatalf("Expected connect to wait for confirmation, got: %v", err)
	}
	if host.peer != nil {
		t.Fatal("Expected no peer before confirmation, got: peer")
	}
	if host.pendingID != "requester" {
		t.Fatalf("Expected pending 'requester', got: %q", host.pendingID)
	}

	if err := host.onAnswer(answerAfterGrace(host, "y\n")); err != nil {
		t.Fatalf("Expected accept, got: %v", err)
	}
	if offer := nextMessage(t, requester); offer.Type != signaling.TypeOffer || offer.From != "host" {
		t.Fatalf("Expected offer from host after accept, got: %+v", offer)
	}
	if host.remoteID != "requester" || host.pendingID != "" {
		t.Fatalf("Expected remote 'requester' and no pending, got: remote %q, pending %q", host.remoteID, host.pendingID)
	}
}

func TestAcceptAnswers(t *testing.T) {
	for _, answer := range []string{"y\n", "Y\n", "  y \n"} {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			url := newTestHub(t)
			host := newTestSession(t, url, "host")
			requester := registerProbe(t, url, "requester")

			if err := host.handle(connectMessage("requester")); err != nil {
				t.Fatalf("Expected connect pending, got: %v", err)
			}
			if err := host.onAnswer(answerAfterGrace(host, answer)); err != nil {
				t.Fatalf("Expected accept, got: %v", err)
			}
			if offer := nextMessage(t, requester); offer.Type != signaling.TypeOffer {
				t.Fatalf("Expected offer after %q, got: %+v", answer, offer)
			}
		})
	}
}

func TestRefuseAnswers(t *testing.T) {
	for _, answer := range []string{"n\n", "N\n", "\n", "yes\n", "s\n", "S\n"} {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			url := newTestHub(t)
			host := newTestSession(t, url, "host")
			requester := registerProbe(t, url, "requester")

			if err := host.handle(connectMessage("requester")); err != nil {
				t.Fatalf("Expected connect pending, got: %v", err)
			}
			if err := host.onAnswer(answerAfterGrace(host, answer)); err != nil {
				t.Fatalf("Expected refuse without error, got: %v", err)
			}

			expectReject(t, requester, "refused by host")
			if host.peer != nil || host.pendingID != "" {
				t.Fatalf("Expected no peer and no pending after refuse, got: peer %v, pending %q", host.peer, host.pendingID)
			}
		})
	}
}

func TestConfirmTimeoutRefuses(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	requester := registerProbe(t, url, "requester")

	if err := host.handle(connectMessage("requester")); err != nil {
		t.Fatalf("Expected connect pending, got: %v", err)
	}
	if err := host.onConfirmTimeout(); err != nil {
		t.Fatalf("Expected timeout refuse without error, got: %v", err)
	}

	expectReject(t, requester, "timed out waiting for host")
	if host.pendingID != "" || host.confirmExpired != nil {
		t.Fatalf("Expected pending cleared, got: pending %q", host.pendingID)
	}
}

func TestLateAnswerAfterTimeoutIgnored(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	requester := registerProbe(t, url, "requester")

	if err := host.handle(connectMessage("requester")); err != nil {
		t.Fatalf("Expected connect pending, got: %v", err)
	}
	if err := host.onConfirmTimeout(); err != nil {
		t.Fatalf("Expected timeout refuse, got: %v", err)
	}
	expectReject(t, requester, "timed out waiting for host")

	if err := host.onAnswer(answerAfterGrace(host, "y\n")); err != nil {
		t.Fatalf("Expected late answer ignored, got: %v", err)
	}
	if host.peer != nil {
		t.Fatal("Expected no peer after late answer, got: peer")
	}
}

func TestConnectWhilePendingIsBusy(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	registerProbe(t, url, "first")
	second := registerProbe(t, url, "second")

	if err := host.handle(connectMessage("first")); err != nil {
		t.Fatalf("Expected first connect pending, got: %v", err)
	}
	if err := host.handle(connectMessage("second")); err != nil {
		t.Fatalf("Expected second connect rejected without error, got: %v", err)
	}

	expectReject(t, second, "host busy")
	if host.pendingID != "first" {
		t.Fatalf("Expected pending to stay 'first', got: %q", host.pendingID)
	}
}

func TestMessagesFromPendingRequesterIgnored(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")

	if err := host.handle(connectMessage("requester")); err != nil {
		t.Fatalf("Expected connect pending, got: %v", err)
	}

	for _, msgType := range []signaling.Type{signaling.TypeOffer, signaling.TypeAnswer, signaling.TypeCandidate, signaling.TypeReject} {
		msg := signaling.Message{Type: msgType, From: "requester", To: "host", Payload: json.RawMessage(`"invalid"`)}
		if err := host.handle(msg); err != nil {
			t.Fatalf("Expected %s from pending requester ignored, got: %v", msgType, err)
		}
	}
	if host.peer != nil {
		t.Fatal("Expected no peer for pending requester, got: peer")
	}
}

func TestNewRequestAfterRefusalIsPrompted(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	first := registerProbe(t, url, "first")
	second := registerProbe(t, url, "second")

	if err := host.handle(connectMessage("first")); err != nil {
		t.Fatalf("Expected first connect pending, got: %v", err)
	}
	if err := host.onAnswer(answerAfterGrace(host, "n\n")); err != nil {
		t.Fatalf("Expected refuse, got: %v", err)
	}
	expectReject(t, first, "refused by host")

	if err := host.handle(connectMessage("second")); err != nil {
		t.Fatalf("Expected second connect pending, got: %v", err)
	}
	if host.pendingID != "second" {
		t.Fatalf("Expected pending 'second', got: %q", host.pendingID)
	}
	if err := host.onAnswer(answerAfterGrace(host, "y\n")); err != nil {
		t.Fatalf("Expected accept, got: %v", err)
	}
	if offer := nextMessage(t, second); offer.Type != signaling.TypeOffer {
		t.Fatalf("Expected offer to second, got: %+v", offer)
	}
}

func TestPromptSanitizesRequesterID(t *testing.T) {
	stdout := captureStdout(t)
	url := newTestHub(t)
	host := newTestSession(t, url, "host")

	if err := host.handle(connectMessage("eve\x1b[31m‮")); err != nil {
		t.Fatalf("Expected connect pending, got: %v", err)
	}

	b, err := os.ReadFile(stdout.Name())
	if err != nil {
		t.Fatalf("Expected captured output, got: %v", err)
	}
	want := "Connection request from \"eve[31m\"\nAccept? [y/N] "
	if string(b) != want {
		t.Fatalf("Expected %q, got: %q", want, string(b))
	}
}

func TestLoopRefusesOnConfirmTimeout(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	host.confirmTimeout = 50 * time.Millisecond
	host.lines = make(chan stdinLine)
	requester := registerProbe(t, url, "requester")

	go host.loop()

	sendFrom(t, requester, connectMessage("requester"))
	expectReject(t, requester, "timed out waiting for host")
}

func TestLoopDiscardsAnswerWithoutPendingRequest(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	host.confirmTimeout = 200 * time.Millisecond
	lines := make(chan stdinLine)
	host.lines = lines
	requester := registerProbe(t, url, "requester")

	go host.loop()

	select {
	case lines <- stdinLine{text: "y\n", readAt: time.Now()}:
	case <-time.After(testTimeout):
		t.Fatal("Expected loop to read the early answer")
	}

	sendFrom(t, requester, connectMessage("requester"))
	expectReject(t, requester, "timed out waiting for host")
}

func TestAnswerReadBeforePromptIgnored(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	registerProbe(t, url, "requester")

	if err := host.handle(connectMessage("requester")); err != nil {
		t.Fatalf("Expected connect pending, got: %v", err)
	}
	early := stdinLine{text: "y\n", readAt: host.promptedAt.Add(-time.Millisecond)}
	if err := host.onAnswer(early); err != nil {
		t.Fatalf("Expected early answer ignored, got: %v", err)
	}

	if host.peer != nil || host.pendingID != "requester" {
		t.Fatalf("Expected request still pending, got: peer %v, pending %q", host.peer, host.pendingID)
	}
}

func TestAnswerWithinGraceIgnored(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	requester := registerProbe(t, url, "requester")

	if err := host.handle(connectMessage("requester")); err != nil {
		t.Fatalf("Expected connect pending, got: %v", err)
	}
	for _, text := range []string{"y\n", "n\n"} {
		fast := stdinLine{text: text, readAt: host.promptedAt.Add(host.answerGrace - time.Millisecond)}
		if err := host.onAnswer(fast); err != nil {
			t.Fatalf("Expected fast answer %q ignored, got: %v", text, err)
		}
		if host.peer != nil || host.pendingID != "requester" {
			t.Fatalf("Expected request still pending after %q, got: peer %v, pending %q", text, host.peer, host.pendingID)
		}
	}

	if err := host.onAnswer(answerAfterGrace(host, "y\n")); err != nil {
		t.Fatalf("Expected accept after grace, got: %v", err)
	}
	if offer := nextMessage(t, requester); offer.Type != signaling.TypeOffer {
		t.Fatalf("Expected offer after grace, got: %+v", offer)
	}
}

func TestLateAnswerDoesNotAcceptNextRequester(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	first := registerProbe(t, url, "first")
	registerProbe(t, url, "second")

	if err := host.handle(connectMessage("first")); err != nil {
		t.Fatalf("Expected first connect pending, got: %v", err)
	}
	if err := host.onConfirmTimeout(); err != nil {
		t.Fatalf("Expected timeout refuse, got: %v", err)
	}
	expectReject(t, first, "timed out waiting for host")

	if err := host.handle(connectMessage("second")); err != nil {
		t.Fatalf("Expected second connect pending, got: %v", err)
	}
	meantForFirst := stdinLine{text: "y\n", readAt: host.promptedAt.Add(200 * time.Millisecond)}
	if err := host.onAnswer(meantForFirst); err != nil {
		t.Fatalf("Expected late answer ignored, got: %v", err)
	}

	if host.peer != nil || host.pendingID != "second" {
		t.Fatalf("Expected second still pending, got: peer %v, pending %q", host.peer, host.pendingID)
	}
}

func TestReadLines(t *testing.T) {
	lines := readLines(strings.NewReader("a\n\nb\npartial"))

	var got []string
	for line := range lines {
		if line.readAt.IsZero() {
			t.Fatalf("Expected read time for %q, got: zero", line.text)
		}
		got = append(got, line.text)
	}

	want := []string{"a\n", "\n", "b\n"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("Expected %q, got: %q", want, got)
	}
}

func sendFrom(t *testing.T, conn interface {
	Send(context.Context, signaling.Message) error
}, msg signaling.Message) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := conn.Send(ctx, msg); err != nil {
		t.Fatalf("Expected message sent, got: %v", err)
	}
}
