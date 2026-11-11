package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Card #2 AC1 / edge — password stored only as argon2id hash; never plaintext.
func TestHashPassword_ProducesArgon2idPHC_NotPlaintext(t *testing.T) {
	hash, err := HashPassword("secret-pass")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hash, "$argon2id$"), "PHC argon2id encoding")
	require.NotContains(t, hash, "secret-pass")
}

// Card #2 AC1 — verify accepts the matching password.
func TestVerifyPassword_Matching_ReturnsTrue(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	ok, err := VerifyPassword(hash, "correct-horse")
	require.NoError(t, err)
	require.True(t, ok)
}

// Card #2 AC4 — wrong password does not verify.
func TestVerifyPassword_Wrong_ReturnsFalse(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	ok, err := VerifyPassword(hash, "wrong-password")
	require.NoError(t, err)
	require.False(t, ok)
}

// Card #2 — two hashes of the same password differ (random salt).
func TestHashPassword_SamePassword_DifferentSalts(t *testing.T) {
	a, err := HashPassword("same-password")
	require.NoError(t, err)
	b, err := HashPassword("same-password")
	require.NoError(t, err)
	require.NotEqual(t, a, b)
}
