package client

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
)

const safetyNonceSize = 32

func safetyCode(hostFingerprint, joinerFingerprint string, joinerNonce, hostNonce []byte) string {
	h := sha256.New()
	h.Write([]byte(hostFingerprint))
	h.Write([]byte(joinerFingerprint))
	h.Write(joinerNonce)
	h.Write(hostNonce)
	sum := h.Sum(nil)
	n := binary.BigEndian.Uint64(sum[:8]) % 1_000_000
	return fmt.Sprintf("%03d %03d", n/1000, n%1000)
}

func newNonce() ([]byte, error) {
	nonce := make([]byte, safetyNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return nonce, nil
}

func commitNonce(nonce []byte) [32]byte {
	return sha256.Sum256(nonce)
}

func verifyCommitment(commitment, nonce []byte) bool {
	expected := commitNonce(nonce)
	return subtle.ConstantTimeCompare(commitment, expected[:]) == 1
}
