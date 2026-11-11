package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (PHC reference-style): t=2, m=64 MiB, p=1.
const (
	argonTime    = 2
	argonMemory  = 64 * 1024
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

var (
	errInvalidHash  = errors.New("auth: invalid password hash")
	errHashMismatch = errors.New("auth: password hash format mismatch")
	b64             = base64.RawStdEncoding
)

// HashPassword returns a PHC-encoded argon2id hash of password.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		b64.EncodeToString(salt), b64.EncodeToString(key),
	), nil
}

// VerifyPassword checks password against a PHC-encoded argon2id hash.
// Comparison of the derived key is constant-time.
func VerifyPassword(encoded, password string) (bool, error) {
	memory, timeCost, threads, salt, want, err := parsePHC(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, nil
	}
	return true, nil
}

func parsePHC(encoded string) (memory, timeCost uint32, threads uint8, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	// "", "argon2id", "v=19", "m=...,t=...,p=...", salt, hash
	if len(parts) != 6 || parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, errInvalidHash
	}
	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return 0, 0, 0, nil, nil, errInvalidHash
	}
	if version != argon2.Version {
		return 0, 0, 0, nil, nil, errHashMismatch
	}
	var p int
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &p); err != nil {
		return 0, 0, 0, nil, nil, errInvalidHash
	}
	if p < 1 || p > 255 {
		return 0, 0, 0, nil, nil, errInvalidHash
	}
	threads = uint8(p)
	salt, err = b64.DecodeString(parts[4])
	if err != nil {
		return 0, 0, 0, nil, nil, errInvalidHash
	}
	hash, err = b64.DecodeString(parts[5])
	if err != nil {
		return 0, 0, 0, nil, nil, errInvalidHash
	}
	return memory, timeCost, threads, salt, hash, nil
}

// DummyVerify runs a throwaway argon2id derivation so login timing for an
// unknown nick is closer to a wrong-password path.
func DummyVerify() {
	salt := make([]byte, argonSaltLen)
	_ = argon2.IDKey([]byte("timing-dummy-password"), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}
