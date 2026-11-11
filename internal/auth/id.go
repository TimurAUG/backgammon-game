package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// newID returns a 32-char hex id from 16 random bytes.
func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
