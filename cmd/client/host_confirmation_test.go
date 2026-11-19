package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pion/webrtc/v5"

	"LightRDP/internal/signaling"
	"LightRDP/internal/transport"
)

const hostQuestion = "Accept \"requester\" with this code? [y/N] "

func answerAfterGrace(s *session, text string) stdinLine {
	return stdinLine{text: text, readAt: s.promptedAt.Add(s.answerGrace)}
}

func connectMessage(from string) signaling.Message {
	return signaling.Message{Type: signaling.TypeConnect, From: from, To: "host"}
}

func nextMessageOtherThan(t *testing.T, conn *transport.Conn, skip ...signaling.Type) signaling.Message {
	t.Helper()

	for {
		msg := nextMessage(t, conn)
		skipped := false
		for _, s := range skip {
			skipped = skipped || msg.Type == s
		}
		if !skipped {
			return msg
		}
	}
}

func expectReject(t *testing.T, conn *transport.Conn, from, want string) {
	t.Helper()

	msg := nextMessageOtherThan(t, conn, signaling.TypeOffer, signaling.TypeCandidate, signaling.TypeConnect)
	if msg.Type != signaling.TypeReject || msg.From != from {
		t.Fatalf("Expected reject from %s, got: %+v", from, msg)
	}
	var payload signaling.ErrorPayload
	if err := msg.Decode(&payload); err != nil {
		t.Fatalf("Expected reject payload, got: %v", err)
	}
	if payload.Message != want {
		t.Fatalf("Expected %q, got: %q", want, payload.Message)
	}
}

func expectOffer(t *testing.T, conn *transport.Conn) {
	t.Helper()

	msg := nextMessageOtherThan(t, conn, signaling.TypeCandidate)
	if msg.Type != signaling.TypeOffer || msg.From != "host" {
		t.Fatalf("Expected offer from host, got: %+v", msg)
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

func readStdout(t *testing.T, file *os.File) string {
	t.Helper()

	b, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("Expected captured output, got: %v", err)
	}
	return string(b)
}

func newHostWithRequest(t *testing.T) (*session, *transport.Conn) {
	t.Helper()

	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	requester := registerProbe(t, url, "requester")

	if err := host.handle(connectMessage("requester")); err != nil {
		t.Fatalf("Expected negotiation started, got: %v", err)
	}
	expectOffer(t, requester)
	return host, requester
}

func TestConnectStartsNegotiationWithoutPrompt(t *testing.T) {
	host, _ := newHostWithRequest(t)

	if host.peer == nil || host.remoteID != "requester" {
		t.Fatalf("Expected peer for 'requester', got: peer %v, remote %q", host.peer, host.remoteID)
	}
	if host.question != "" || host.confirmed {
		t.Fatalf("Expected no prompt before connected, got: question %q, confirmed %v", host.question, host.confirmed)
	}
	if host.confirmExpired == nil {
		t.Fatal("Expected timeout started at the request, got: nil")
	}
}

func TestConnectPrintsSanitizedRequestOnly(t *testing.T) {
	stdout := captureStdout(t)
	url := newTestHub(t)
	host := newTestSession(t, url, "host")

	if err := host.handle(connectMessage("eve\x1b[31m‮")); err != nil {
		t.Fatalf("Expected negotiation started, got: %v", err)
	}
	host.dropAttempt()

	want := "Connection request from \"eve[31m\"\n"
	if got := readStdout(t, stdout); got != want {
		t.Fatalf("Expected %q, got: %q", want, got)
	}
}

func TestHostPromptRepeatedOnFastAnswer(t *testing.T) {
	stdout := captureStdout(t)
	host, _ := newHostWithRequest(t)

	host.prompt(hostQuestion)
	fast := stdinLine{text: "y\n", readAt: host.promptedAt.Add(host.answerGrace - time.Millisecond)}
	if err := host.onAnswer(fast); err != nil {
		t.Fatalf("Expected fast answer ignored, got: %v", err)
	}
	if host.confirmed {
		t.Fatal("Expected fast answer not to confirm, got: confirmed")
	}
	host.dropAttempt()

	want := "Connection request from \"requester\"\n" + hostQuestion + "Answer ignored: too fast. " + hostQuestion
	if got := readStdout(t, stdout); got != want {
		t.Fatalf("Expected %q, got: %q", want, got)
	}
}

func TestHostAcceptAnswers(t *testing.T) {
	for _, answer := range []string{"y\n", "Y\n", "  y \n"} {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			host, _ := newHostWithRequest(t)
			host.prompt(hostQuestion)

			if err := host.onAnswer(answerAfterGrace(host, answer)); err != nil {
				t.Fatalf("Expected accept, got: %v", err)
			}
			if !host.confirmed || host.question != "" || host.confirmExpired != nil {
				t.Fatalf("Expected confirmed without prompt or timeout, got: confirmed %v, question %q", host.confirmed, host.question)
			}
			if host.peer == nil || host.chatOpen() {
				t.Fatalf("Expected peer kept and chat waiting for the other side, got: peer %v, chat open %v", host.peer, host.chatOpen())
			}
		})
	}
}

func TestHostRefuseAnswersKeepWaiting(t *testing.T) {
	for _, answer := range []string{"n\n", "N\n", "\n", "yes\n", "s\n", "S\n"} {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			url := newTestHub(t)
			host := newTestSession(t, url, "host")
			requester := registerProbe(t, url, "requester")
			next := registerProbe(t, url, "next")

			if err := host.handle(connectMessage("requester")); err != nil {
				t.Fatalf("Expected negotiation started, got: %v", err)
			}
			host.prompt(hostQuestion)
			if err := host.onAnswer(answerAfterGrace(host, answer)); err != nil {
				t.Fatalf("Expected refuse without error, got: %v", err)
			}

			expectReject(t, requester, "host", "refused by host")
			if host.peer != nil || host.question != "" || host.confirmExpired != nil {
				t.Fatalf("Expected attempt dropped, got: peer %v, question %q", host.peer, host.question)
			}

			if err := host.handle(connectMessage("next")); err != nil {
				t.Fatalf("Expected new request accepted for negotiation, got: %v", err)
			}
			expectOffer(t, next)
		})
	}
}

func TestJoinerAnswers(t *testing.T) {
	url := newTestHub(t)
	hostProbe := registerProbe(t, url, "host")

	joiner := newTestSession(t, url, "joiner")
	if err := joiner.join("host"); err != nil {
		t.Fatalf("Expected join sent, got: %v", err)
	}
	joiner.prompt(joinerQuestion)

	if err := joiner.onAnswer(answerAfterGrace(joiner, "y\n")); err != nil {
		t.Fatalf("Expected accept, got: %v", err)
	}
	if !joiner.confirmed || joiner.question != "" {
		t.Fatalf("Expected confirmed, got: confirmed %v, question %q", joiner.confirmed, joiner.question)
	}
	if msg := nextMessage(t, hostProbe); msg.Type != signaling.TypeConnect {
		t.Fatalf("Expected connect, got: %+v", msg)
	}
}

func TestJoinerRefuseAnswers(t *testing.T) {
	for _, answer := range []string{"n\n", "\n", "yes\n"} {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			url := newTestHub(t)
			hostProbe := registerProbe(t, url, "host")

			joiner := newTestSession(t, url, "joiner")
			if err := joiner.join("host"); err != nil {
				t.Fatalf("Expected join sent, got: %v", err)
			}
			joiner.prompt(joinerQuestion)

			err := joiner.onAnswer(answerAfterGrace(joiner, answer))
			if err == nil || err.Error() != "safety code not confirmed" {
				t.Fatalf("Expected 'safety code not confirmed', got: %v", err)
			}
			expectReject(t, hostProbe, "joiner", "safety code not confirmed")
		})
	}
}

func TestAnswerWithoutPromptIgnored(t *testing.T) {
	host, _ := newHostWithRequest(t)

	if err := host.onAnswer(stdinLine{text: "y\n", readAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("Expected answer ignored, got: %v", err)
	}
	if host.confirmed {
		t.Fatal("Expected no confirmation before the prompt, got: confirmed")
	}
}

func TestAnswerReadBeforePromptIgnored(t *testing.T) {
	host, _ := newHostWithRequest(t)
	host.prompt(hostQuestion)

	early := stdinLine{text: "y\n", readAt: host.promptedAt.Add(-time.Millisecond)}
	if err := host.onAnswer(early); err != nil {
		t.Fatalf("Expected early answer ignored, got: %v", err)
	}
	if host.confirmed || host.question != hostQuestion {
		t.Fatalf("Expected prompt still pending, got: confirmed %v, question %q", host.confirmed, host.question)
	}
}

func TestAnswerWithinGraceIgnored(t *testing.T) {
	host, _ := newHostWithRequest(t)
	host.prompt(hostQuestion)

	for _, text := range []string{"y\n", "n\n"} {
		fast := stdinLine{text: text, readAt: host.promptedAt.Add(host.answerGrace - time.Millisecond)}
		if err := host.onAnswer(fast); err != nil {
			t.Fatalf("Expected fast answer %q ignored, got: %v", text, err)
		}
		if host.peer == nil || host.confirmed {
			t.Fatalf("Expected prompt still pending after %q, got: peer %v, confirmed %v", text, host.peer, host.confirmed)
		}
	}

	if err := host.onAnswer(answerAfterGrace(host, "y\n")); err != nil {
		t.Fatalf("Expected accept after grace, got: %v", err)
	}
	if !host.confirmed {
		t.Fatal("Expected confirmed after grace, got: not confirmed")
	}
}

func TestConfirmTimeoutRefuses(t *testing.T) {
	host, requester := newHostWithRequest(t)
	host.prompt(hostQuestion)

	if err := host.onConfirmTimeout(); err != nil {
		t.Fatalf("Expected timeout refuse without error, got: %v", err)
	}

	expectReject(t, requester, "host", "timed out waiting for host")
	if host.peer != nil || host.question != "" || host.confirmExpired != nil {
		t.Fatalf("Expected attempt dropped, got: peer %v, question %q", host.peer, host.question)
	}
}

func TestConfirmTimeoutAfterHostConfirmedIgnored(t *testing.T) {
	host, _ := newHostWithRequest(t)
	host.prompt(hostQuestion)
	if err := host.onAnswer(answerAfterGrace(host, "y\n")); err != nil {
		t.Fatalf("Expected accept, got: %v", err)
	}

	if err := host.onConfirmTimeout(); err != nil {
		t.Fatalf("Expected stale timeout ignored, got: %v", err)
	}
	if host.peer == nil || !host.confirmed {
		t.Fatalf("Expected attempt kept, got: peer %v, confirmed %v", host.peer, host.confirmed)
	}
}

func TestLateAnswerAfterTimeoutIgnored(t *testing.T) {
	host, requester := newHostWithRequest(t)
	host.prompt(hostQuestion)
	if err := host.onConfirmTimeout(); err != nil {
		t.Fatalf("Expected timeout refuse, got: %v", err)
	}
	expectReject(t, requester, "host", "timed out waiting for host")

	if err := host.onAnswer(answerAfterGrace(host, "y\n")); err != nil {
		t.Fatalf("Expected late answer ignored, got: %v", err)
	}
	if host.peer != nil || host.confirmed {
		t.Fatalf("Expected nothing accepted, got: peer %v, confirmed %v", host.peer, host.confirmed)
	}
}

func TestLateAnswerDoesNotAcceptNextRequester(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	first := registerProbe(t, url, "first")
	registerProbe(t, url, "second")

	if err := host.handle(connectMessage("first")); err != nil {
		t.Fatalf("Expected negotiation started, got: %v", err)
	}
	if err := host.onConfirmTimeout(); err != nil {
		t.Fatalf("Expected timeout refuse, got: %v", err)
	}
	expectReject(t, first, "host", "timed out waiting for host")

	if err := host.handle(connectMessage("second")); err != nil {
		t.Fatalf("Expected second negotiation started, got: %v", err)
	}
	host.prompt("Accept \"second\" with this code? [y/N] ")
	meantForFirst := stdinLine{text: "y\n", readAt: host.promptedAt.Add(200 * time.Millisecond)}
	if err := host.onAnswer(meantForFirst); err != nil {
		t.Fatalf("Expected late answer ignored, got: %v", err)
	}

	if host.confirmed || host.remoteID != "second" {
		t.Fatalf("Expected second still pending, got: confirmed %v, remote %q", host.confirmed, host.remoteID)
	}
}

func TestConnectWhileNegotiatingIsBusy(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	registerProbe(t, url, "first")
	second := registerProbe(t, url, "second")

	if err := host.handle(connectMessage("first")); err != nil {
		t.Fatalf("Expected first negotiation started, got: %v", err)
	}
	if err := host.handle(connectMessage("second")); err != nil {
		t.Fatalf("Expected second connect rejected without error, got: %v", err)
	}

	expectReject(t, second, "host", "host busy")
	if host.remoteID != "first" {
		t.Fatalf("Expected remote to stay 'first', got: %q", host.remoteID)
	}
}

func TestConnectWhileDecidingIsBusy(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	registerProbe(t, url, "first")
	second := registerProbe(t, url, "second")

	if err := host.handle(connectMessage("first")); err != nil {
		t.Fatalf("Expected first negotiation started, got: %v", err)
	}
	host.prompt("Accept \"first\" with this code? [y/N] ")
	if err := host.handle(connectMessage("second")); err != nil {
		t.Fatalf("Expected second connect rejected without error, got: %v", err)
	}

	expectReject(t, second, "host", "host busy")
	if host.question == "" || host.remoteID != "first" {
		t.Fatalf("Expected first still deciding, got: question %q, remote %q", host.question, host.remoteID)
	}
}

func TestHostAcceptsJoinerRejectWhilePending(t *testing.T) {
	url := newTestHub(t)
	host := newTestSession(t, url, "host")
	requester := registerProbe(t, url, "requester")
	next := registerProbe(t, url, "next")

	if err := host.handle(connectMessage("requester")); err != nil {
		t.Fatalf("Expected negotiation started, got: %v", err)
	}
	expectOffer(t, requester)
	host.connected.Store(true)
	host.prompt(hostQuestion)

	reject := signaling.Message{Type: signaling.TypeReject, From: "requester", To: "host", Payload: errorPayload(t, "safety code not confirmed")}
	if err := host.handle(reject); err != nil {
		t.Fatalf("Expected host to keep running, got: %v", err)
	}
	if host.peer != nil || host.question != "" || host.connected.Load() {
		t.Fatalf("Expected attempt dropped, got: peer %v, question %q", host.peer, host.question)
	}

	if err := host.handle(connectMessage("next")); err != nil {
		t.Fatalf("Expected new request accepted for negotiation, got: %v", err)
	}
	expectOffer(t, next)
}

func TestHostPeerFailureBeforeChatKeepsWaiting(t *testing.T) {
	for _, state := range []webrtc.PeerConnectionState{webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed} {
		t.Run(state.String(), func(t *testing.T) {
			host, _ := newHostWithRequest(t)

			ended, err := host.onPeerState(peerState{peer: host.peer, state: state})
			if ended || err != nil {
				t.Fatalf("Expected host to keep running, got: ended %v, err %v", ended, err)
			}
			if host.peer != nil {
				t.Fatal("Expected attempt dropped, got: peer")
			}
		})
	}
}

func TestHostPeerFailureAfterChatOpenEnds(t *testing.T) {
	host, _ := newHostWithRequest(t)
	host.chat.open = true

	ended, err := host.onPeerState(peerState{peer: host.peer, state: webrtc.PeerConnectionStateFailed})
	if !ended || err == nil || err.Error() != "peer connection failed" {
		t.Fatalf("Expected 'peer connection failed', got: ended %v, err %v", ended, err)
	}
}

func TestJoinerPeerFailureEnds(t *testing.T) {
	url := newTestHub(t)
	registerProbe(t, url, "host")
	joiner := newTestSession(t, url, "joiner")
	if err := joiner.join("host"); err != nil {
		t.Fatalf("Expected join sent, got: %v", err)
	}

	ended, err := joiner.onPeerState(peerState{peer: joiner.peer, state: webrtc.PeerConnectionStateFailed})
	if !ended || err == nil || err.Error() != "peer connection failed" {
		t.Fatalf("Expected 'peer connection failed', got: ended %v, err %v", ended, err)
	}
}

func TestPeerStateFromDroppedAttemptIgnored(t *testing.T) {
	host, requester := newHostWithRequest(t)
	dropped := host.peer
	if err := host.onConfirmTimeout(); err != nil {
		t.Fatalf("Expected timeout refuse, got: %v", err)
	}
	expectReject(t, requester, "host", "timed out waiting for host")

	for _, state := range []webrtc.PeerConnectionState{webrtc.PeerConnectionStateConnected, webrtc.PeerConnectionStateFailed} {
		ended, err := host.onPeerState(peerState{peer: dropped, state: state})
		if ended || err != nil {
			t.Fatalf("Expected %s from dropped peer ignored, got: ended %v, err %v", state, ended, err)
		}
	}
	if host.question != "" {
		t.Fatalf("Expected no prompt for dropped peer, got: %q", host.question)
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
	expectReject(t, requester, "host", "timed out waiting for host")
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
	expectReject(t, requester, "host", "timed out waiting for host")
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
