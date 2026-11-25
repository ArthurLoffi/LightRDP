package client

import (
	"encoding/hex"
	"errors"
	"strings"
)

const (
	commitMessagePrefix = "lightrdp:commit:"
	nonceMessagePrefix  = "lightrdp:nonce:"
	revealMessagePrefix = "lightrdp:reveal:"

	commitmentMismatchError = "safety code commitment mismatch"
	handshakeFailedError    = "safety code handshake failed: "
)

var safetyHandshakeSteps = []struct {
	prefix string
	name   string
}{
	{commitMessagePrefix, "commit"},
	{nonceMessagePrefix, "nonce"},
	{revealMessagePrefix, "reveal"},
}

type safetyHandshake struct {
	joining          bool
	localNonce       []byte
	remoteCommitment []byte
	remoteNonce      []byte
	done             bool
}

type safetyHandshakeResult struct {
	chat        *chat
	joinerNonce []byte
	hostNonce   []byte
	err         error
}

func handshakeFailed(detail string) error {
	return errors.New(handshakeFailedError + detail)
}

func isSafetyHandshakeMessage(text string) bool {
	for _, step := range safetyHandshakeSteps {
		if strings.HasPrefix(text, step.prefix) {
			return true
		}
	}
	return false
}

func parseSafetyHandshakeMessage(text string) (string, []byte, error) {
	for _, step := range safetyHandshakeSteps {
		encoded, found := strings.CutPrefix(text, step.prefix)
		if !found {
			continue
		}
		value, err := hex.DecodeString(encoded)
		if err != nil || len(value) != safetyNonceSize {
			return "", nil, handshakeFailed("invalid " + step.name)
		}
		return step.prefix, value, nil
	}
	return "", nil, handshakeFailed("unknown message")
}

func unexpectedStep(prefix string) error {
	for _, step := range safetyHandshakeSteps {
		if step.prefix == prefix {
			return handshakeFailed("unexpected " + step.name)
		}
	}
	return handshakeFailed("unknown message")
}

func (h *safetyHandshake) start() (string, error) {
	if !h.joining || h.localNonce != nil {
		return "", handshakeFailed("unexpected start")
	}
	nonce, err := newNonce()
	if err != nil {
		return "", err
	}
	h.localNonce = nonce
	commitment := commitNonce(nonce)
	return commitMessagePrefix + hex.EncodeToString(commitment[:]), nil
}

func (h *safetyHandshake) receive(text string) (string, error) {
	prefix, value, err := parseSafetyHandshakeMessage(text)
	if err != nil {
		return "", err
	}
	if h.joining {
		return h.receiveAsJoiner(prefix, value)
	}
	return h.receiveAsHost(prefix, value)
}

func (h *safetyHandshake) receiveAsJoiner(prefix string, value []byte) (string, error) {
	if prefix != nonceMessagePrefix || h.localNonce == nil || h.done {
		return "", unexpectedStep(prefix)
	}
	h.remoteNonce = value
	h.done = true
	return revealMessagePrefix + hex.EncodeToString(h.localNonce), nil
}

func (h *safetyHandshake) receiveAsHost(prefix string, value []byte) (string, error) {
	switch {
	case prefix == commitMessagePrefix && h.remoteCommitment == nil:
		nonce, err := newNonce()
		if err != nil {
			return "", err
		}
		h.remoteCommitment = value
		h.localNonce = nonce
		return nonceMessagePrefix + hex.EncodeToString(nonce), nil

	case prefix == revealMessagePrefix && h.remoteCommitment != nil && !h.done:
		if !verifyCommitment(h.remoteCommitment, value) {
			return "", errors.New(commitmentMismatchError)
		}
		h.remoteNonce = value
		h.done = true
		return "", nil
	}
	return "", unexpectedStep(prefix)
}

func (h *safetyHandshake) nonces() (joinerNonce, hostNonce []byte) {
	if h.joining {
		return h.localNonce, h.remoteNonce
	}
	return h.remoteNonce, h.localNonce
}
