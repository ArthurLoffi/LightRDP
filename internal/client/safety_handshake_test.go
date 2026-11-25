package client

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func runSafetyHandshake(t *testing.T) (joiner, host *safetyHandshake, reveal string) {
	t.Helper()

	joiner = &safetyHandshake{joining: true}
	host = &safetyHandshake{}

	commit, err := joiner.start()
	if err != nil {
		t.Fatalf("Expected commit, got: %v", err)
	}
	nonce, err := host.receive(commit)
	if err != nil {
		t.Fatalf("Expected host nonce, got: %v", err)
	}
	reveal, err = joiner.receive(nonce)
	if err != nil {
		t.Fatalf("Expected reveal, got: %v", err)
	}
	return joiner, host, reveal
}

func expectHandshakeError(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil || err.Error() != want {
		t.Fatalf("Expected %q, got: %v", want, err)
	}
}

func TestSafetyHandshakeGivesSameNoncesOnBothSides(t *testing.T) {
	joiner, host, reveal := runSafetyHandshake(t)

	reply, err := host.receive(reveal)
	if err != nil || reply != "" {
		t.Fatalf("Expected reveal accepted without reply, got: reply %q, err %v", reply, err)
	}
	if !joiner.done || !host.done {
		t.Fatalf("Expected both sides done, got: joiner %v, host %v", joiner.done, host.done)
	}

	joinerSideJoinerNonce, joinerSideHostNonce := joiner.nonces()
	hostSideJoinerNonce, hostSideHostNonce := host.nonces()
	if !bytes.Equal(joinerSideJoinerNonce, hostSideJoinerNonce) || !bytes.Equal(joinerSideHostNonce, hostSideHostNonce) {
		t.Fatal("Expected the same nonces on both sides, got: different")
	}
	if bytes.Equal(joinerSideJoinerNonce, joinerSideHostNonce) {
		t.Fatal("Expected different joiner and host nonces, got: the same")
	}
}

func TestSafetyHandshakeMessages(t *testing.T) {
	joiner := &safetyHandshake{joining: true}
	commit, err := joiner.start()
	if err != nil {
		t.Fatalf("Expected commit, got: %v", err)
	}
	commitment := commitNonce(joiner.localNonce)
	if commit != commitMessagePrefix+hex.EncodeToString(commitment[:]) {
		t.Fatalf("Expected commit of the joiner nonce, got: %q", commit)
	}
	if strings.Contains(commit, hex.EncodeToString(joiner.localNonce)) {
		t.Fatalf("Expected commit without the nonce, got: %q", commit)
	}

	host := &safetyHandshake{}
	nonce, err := host.receive(commit)
	if err != nil {
		t.Fatalf("Expected host nonce, got: %v", err)
	}
	if nonce != nonceMessagePrefix+hex.EncodeToString(host.localNonce) {
		t.Fatalf("Expected host nonce message, got: %q", nonce)
	}

	reveal, err := joiner.receive(nonce)
	if err != nil {
		t.Fatalf("Expected reveal, got: %v", err)
	}
	if reveal != revealMessagePrefix+hex.EncodeToString(joiner.localNonce) {
		t.Fatalf("Expected reveal of the joiner nonce, got: %q", reveal)
	}
}

func TestSafetyHandshakeRefusesWrongReveal(t *testing.T) {
	_, host, _ := runSafetyHandshake(t)

	_, err := host.receive(revealMessagePrefix + hex.EncodeToString(hostNonceA))
	expectHandshakeError(t, err, commitmentMismatchError)
	if host.done {
		t.Fatal("Expected host not done after a wrong reveal, got: done")
	}
}

func TestSafetyHandshakeHostNeedsCommitBeforeNonce(t *testing.T) {
	host := &safetyHandshake{}

	_, err := host.start()
	expectHandshakeError(t, err, handshakeFailedError+"unexpected start")

	_, err = host.receive(revealMessagePrefix + hex.EncodeToString(joinerNonceA))
	expectHandshakeError(t, err, handshakeFailedError+"unexpected reveal")

	if host.localNonce != nil {
		t.Fatalf("Expected no host nonce before the commit, got: %x", host.localNonce)
	}
}

func TestSafetyHandshakeRefusesOutOfOrderMessages(t *testing.T) {
	validValue := hex.EncodeToString(joinerNonceA)

	cases := []struct {
		name    string
		setup   func(t *testing.T) *safetyHandshake
		message string
		want    string
	}{
		{
			name:    "joiner nonce before start",
			setup:   func(t *testing.T) *safetyHandshake { return &safetyHandshake{joining: true} },
			message: nonceMessagePrefix + validValue,
			want:    "unexpected nonce",
		},
		{
			name: "joiner commit",
			setup: func(t *testing.T) *safetyHandshake {
				joiner := &safetyHandshake{joining: true}
				if _, err := joiner.start(); err != nil {
					t.Fatalf("Expected commit, got: %v", err)
				}
				return joiner
			},
			message: commitMessagePrefix + validValue,
			want:    "unexpected commit",
		},
		{
			name: "joiner repeated nonce",
			setup: func(t *testing.T) *safetyHandshake {
				joiner, _, _ := runSafetyHandshake(t)
				return joiner
			},
			message: nonceMessagePrefix + validValue,
			want:    "unexpected nonce",
		},
		{
			name: "host repeated commit",
			setup: func(t *testing.T) *safetyHandshake {
				_, host, _ := runSafetyHandshake(t)
				return host
			},
			message: commitMessagePrefix + validValue,
			want:    "unexpected commit",
		},
		{
			name: "host repeated reveal",
			setup: func(t *testing.T) *safetyHandshake {
				_, host, reveal := runSafetyHandshake(t)
				if _, err := host.receive(reveal); err != nil {
					t.Fatalf("Expected reveal accepted, got: %v", err)
				}
				return host
			},
			message: revealMessagePrefix + validValue,
			want:    "unexpected reveal",
		},
		{
			name:    "host nonce",
			setup:   func(t *testing.T) *safetyHandshake { return &safetyHandshake{} },
			message: nonceMessagePrefix + validValue,
			want:    "unexpected nonce",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := c.setup(t)
			_, err := h.receive(c.message)
			expectHandshakeError(t, err, handshakeFailedError+c.want)
		})
	}
}

func TestSafetyHandshakeRefusesInvalidValues(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{"not hex", commitMessagePrefix + strings.Repeat("zz", safetyNonceSize), "invalid commit"},
		{"too short", commitMessagePrefix + hex.EncodeToString(joinerNonceA[:safetyNonceSize-1]), "invalid commit"},
		{"too long", revealMessagePrefix + hex.EncodeToString(append(joinerNonceA, 0x00)), "invalid reveal"},
		{"empty", nonceMessagePrefix, "invalid nonce"},
		{"unknown", "lightrdp:other:" + hex.EncodeToString(joinerNonceA), "unknown message"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			host := &safetyHandshake{}
			_, err := host.receive(c.message)
			expectHandshakeError(t, err, handshakeFailedError+c.want)
			if host.remoteCommitment != nil || host.localNonce != nil {
				t.Fatal("Expected no state change after an invalid message, got: changed")
			}
		})
	}
}

func TestIsSafetyHandshakeMessage(t *testing.T) {
	for text, want := range map[string]bool{
		commitMessagePrefix + "00": true,
		nonceMessagePrefix:         true,
		revealMessagePrefix + "zz": true,
		safetyCodeConfirmedMessage: false,
		"hello":                    false,
		"lightrdp:commit":          false,
		" " + commitMessagePrefix:  false,
	} {
		if got := isSafetyHandshakeMessage(text); got != want {
			t.Fatalf("Expected isSafetyHandshakeMessage(%q) = %v, got: %v", text, want, got)
		}
	}
}
