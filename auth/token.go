package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// Token returns a cryptographically random 64-char hex string (32 bytes),
// used for opaque session ids and CSRF tokens.
func Token() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
