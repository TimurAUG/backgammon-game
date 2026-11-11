package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Card #2 AC1 — register creates user with hash only.
func TestMemoryStore_CreateUser_StoresHashNotPlaintext(t *testing.T) {
	s := NewMemoryStore()
	u, err := s.CreateUser(context.Background(), "Ann", "password1")
	require.NoError(t, err)
	require.Equal(t, "Ann", u.Nick)
	require.Equal(t, "ann", u.NickLower)
	require.True(t, strings.HasPrefix(u.PasswordHash, "$argon2id$"))
	require.NotContains(t, u.PasswordHash, "password1")
}

// Card #2 AC2 / edge — nicks compare case-insensitively.
func TestMemoryStore_CreateUser_DuplicateNickAnyCase_ErrNickTaken(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.CreateUser(context.Background(), "Ann", "password1")
	require.NoError(t, err)
	_, err = s.CreateUser(context.Background(), "ann", "password2")
	require.ErrorIs(t, err, ErrNickTaken)
}

// Card #2 edge — empty/short/long nick or short/long password → ErrInvalidInput.
func TestValidateCredentials_Table(t *testing.T) {
	cases := []struct {
		name, nick, password string
		wantErr              bool
	}{
		{"ok", "abc", "password1", false},
		{"empty nick", "", "password1", true},
		{"whitespace nick", "   ", "password1", true},
		{"short nick", "ab", "password1", true},
		{"long nick", strings.Repeat("a", 21), "password1", true},
		{"short password", "abc", "short", true},
		{"long password", "abc", strings.Repeat("x", 129), true},
		{"trim ok", "  abc  ", "password1", false},
		{"max nick", strings.Repeat("я", 20), "password1", false},
		{"max password", "abc", strings.Repeat("x", 128), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCredentials(NormalizeNick(tc.nick), tc.password)
			if tc.wantErr {
				require.ErrorIs(t, err, ErrInvalidInput)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Card #2 AC3/AC4 — authenticate success and generic failure.
func TestMemoryStore_Authenticate_SuccessAndFailure(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.CreateUser(context.Background(), "Bob", "password1")
	require.NoError(t, err)

	u, err := s.Authenticate(context.Background(), "bob", "password1")
	require.NoError(t, err)
	require.Equal(t, "Bob", u.Nick)

	_, err = s.Authenticate(context.Background(), "bob", "wrong-pass")
	require.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = s.Authenticate(context.Background(), "nobody", "password1")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

// Card #2 AC7 / edge — multi-session; delete one leaves the other.
func TestMemoryStore_Sessions_LogoutOneKeepsSibling(t *testing.T) {
	s := NewMemoryStore()
	u, err := s.CreateUser(context.Background(), "Carla", "password1")
	require.NoError(t, err)
	exp := time.Now().UTC().Add(time.Hour)
	a, err := s.CreateSession(context.Background(), u.ID, exp)
	require.NoError(t, err)
	b, err := s.CreateSession(context.Background(), u.ID, exp)
	require.NoError(t, err)
	require.NotEqual(t, a.ID, b.ID)

	require.NoError(t, s.DeleteSession(context.Background(), a.ID))
	_, err = s.GetSessionUser(context.Background(), a.ID, time.Now())
	require.ErrorIs(t, err, ErrNotFound)
	got, err := s.GetSessionUser(context.Background(), b.ID, time.Now())
	require.NoError(t, err)
	require.Equal(t, "Carla", got.Nick)
}

// Card #2 edge — session past expires_at → not found.
func TestMemoryStore_GetSessionUser_Expired_ErrNotFound(t *testing.T) {
	s := NewMemoryStore()
	u, err := s.CreateUser(context.Background(), "Dan", "password1")
	require.NoError(t, err)
	past := time.Now().UTC().Add(-time.Minute)
	sess, err := s.CreateSession(context.Background(), u.ID, past)
	require.NoError(t, err)
	_, err = s.GetSessionUser(context.Background(), sess.ID, time.Now())
	require.ErrorIs(t, err, ErrNotFound)
}
