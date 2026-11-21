package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

func safetyCode(hostFingerprint, joinerFingerprint string) string {
	sum := sha256.Sum256([]byte(hostFingerprint + joinerFingerprint))
	n := binary.BigEndian.Uint64(sum[:8]) % 1_000_000
	return fmt.Sprintf("%03d %03d", n/1000, n%1000)
}
