package client

import (
	"bytes"
	"regexp"
	"testing"
)

const (
	hostFingerprintA   = "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
	joinerFingerprintA = "01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef"
)

var (
	safetyCodeFormat = regexp.MustCompile(`^[0-9]{3} [0-9]{3}$`)
	joinerNonceA     = bytes.Repeat([]byte{0x11}, safetyNonceSize)
	hostNonceA       = bytes.Repeat([]byte{0x22}, safetyNonceSize)
)

func TestSafetyCodeSameOnBothRoles(t *testing.T) {
	hostSide := safetyCode(hostFingerprintA, joinerFingerprintA, joinerNonceA, hostNonceA)
	joinerSide := safetyCode(hostFingerprintA, joinerFingerprintA, joinerNonceA, hostNonceA)

	if hostSide != joinerSide {
		t.Fatalf("Expected same code on both roles, got: %q and %q", hostSide, joinerSide)
	}
	if !safetyCodeFormat.MatchString(hostSide) {
		t.Fatalf("Expected 'NNN NNN' format, got: %q", hostSide)
	}
}

func TestSafetyCodeChangesWithAnyInput(t *testing.T) {
	base := safetyCode(hostFingerprintA, joinerFingerprintA, joinerNonceA, hostNonceA)

	changedHost := "ab" + hostFingerprintA[2:]
	changedJoiner := joinerFingerprintA[:len(joinerFingerprintA)-1] + "e"
	changedJoinerNonce := append([]byte{0x12}, joinerNonceA[1:]...)
	changedHostNonce := append(append([]byte{}, hostNonceA[:safetyNonceSize-1]...), 0x23)

	for name, code := range map[string]string{
		"host fingerprint changed":   safetyCode(changedHost, joinerFingerprintA, joinerNonceA, hostNonceA),
		"joiner fingerprint changed": safetyCode(hostFingerprintA, changedJoiner, joinerNonceA, hostNonceA),
		"fingerprints swapped":       safetyCode(joinerFingerprintA, hostFingerprintA, joinerNonceA, hostNonceA),
		"joiner nonce changed":       safetyCode(hostFingerprintA, joinerFingerprintA, changedJoinerNonce, hostNonceA),
		"host nonce changed":         safetyCode(hostFingerprintA, joinerFingerprintA, joinerNonceA, changedHostNonce),
		"nonces swapped":             safetyCode(hostFingerprintA, joinerFingerprintA, hostNonceA, joinerNonceA),
	} {
		if code == base {
			t.Fatalf("Expected a different code when %s, got the same: %q", name, code)
		}
	}
}

func TestSafetyCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := range 2000 {
		code := safetyCode(hostFingerprintA, string(rune('a'+i%26))+joinerFingerprintA[:i%len(joinerFingerprintA)], joinerNonceA, hostNonceA)
		if !safetyCodeFormat.MatchString(code) {
			t.Fatalf("Expected 'NNN NNN' format, got: %q", code)
		}
		seen[code[:1]] = true
	}
	if len(seen) < 10 {
		t.Fatalf("Expected codes to start with every digit, got: %v", seen)
	}
}

func TestNewNonce(t *testing.T) {
	first, err := newNonce()
	if err != nil {
		t.Fatalf("Expected nonce, got: %v", err)
	}
	second, err := newNonce()
	if err != nil {
		t.Fatalf("Expected nonce, got: %v", err)
	}

	if len(first) != safetyNonceSize || len(second) != safetyNonceSize {
		t.Fatalf("Expected %d-byte nonces, got: %d and %d", safetyNonceSize, len(first), len(second))
	}
	if bytes.Equal(first, second) {
		t.Fatalf("Expected different nonces, got the same: %x", first)
	}
}

func TestVerifyCommitment(t *testing.T) {
	commitment := commitNonce(joinerNonceA)

	if !verifyCommitment(commitment[:], joinerNonceA) {
		t.Fatal("Expected commitment to match its nonce, got: mismatch")
	}
	if verifyCommitment(commitment[:], hostNonceA) {
		t.Fatal("Expected commitment to refuse another nonce, got: match")
	}
	if verifyCommitment(commitment[:safetyNonceSize-1], joinerNonceA) {
		t.Fatal("Expected truncated commitment refused, got: match")
	}
}
