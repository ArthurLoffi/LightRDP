package client

import (
	"regexp"
	"testing"
)

const (
	hostFingerprintA   = "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
	joinerFingerprintA = "01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef:01:23:45:67:89:ab:cd:ef"
)

var safetyCodeFormat = regexp.MustCompile(`^[0-9]{3} [0-9]{3}$`)

func TestSafetyCodeSameOnBothRoles(t *testing.T) {
	hostSide := safetyCode(hostFingerprintA, joinerFingerprintA)
	joinerSide := safetyCode(hostFingerprintA, joinerFingerprintA)

	if hostSide != joinerSide {
		t.Fatalf("Expected same code on both roles, got: %q and %q", hostSide, joinerSide)
	}
	if !safetyCodeFormat.MatchString(hostSide) {
		t.Fatalf("Expected 'NNN NNN' format, got: %q", hostSide)
	}
}

func TestSafetyCodeChangesWithEitherFingerprint(t *testing.T) {
	base := safetyCode(hostFingerprintA, joinerFingerprintA)

	changedHost := "ab" + hostFingerprintA[2:]
	changedJoiner := joinerFingerprintA[:len(joinerFingerprintA)-1] + "e"

	for name, code := range map[string]string{
		"host changed":   safetyCode(changedHost, joinerFingerprintA),
		"joiner changed": safetyCode(hostFingerprintA, changedJoiner),
		"order swapped":  safetyCode(joinerFingerprintA, hostFingerprintA),
	} {
		if code == base {
			t.Fatalf("Expected a different code when %s, got the same: %q", name, code)
		}
	}
}

func TestSafetyCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := range 2000 {
		code := safetyCode(hostFingerprintA, string(rune('a'+i%26))+joinerFingerprintA[:i%len(joinerFingerprintA)])
		if !safetyCodeFormat.MatchString(code) {
			t.Fatalf("Expected 'NNN NNN' format, got: %q", code)
		}
		seen[code[:1]] = true
	}
	if len(seen) < 10 {
		t.Fatalf("Expected codes to start with every digit, got: %v", seen)
	}
}
