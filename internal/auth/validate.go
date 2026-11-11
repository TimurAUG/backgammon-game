package auth

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	// ErrInvalidInput is returned for bad nick/password shape or missing JSON fields.
	ErrInvalidInput = errors.New("auth: invalid input")
)

const (
	nickMinLen     = 3
	nickMaxLen     = 20
	passwordMinLen = 8
	passwordMaxLen = 128
)

// NormalizeNick trims leading/trailing whitespace; casing is preserved.
func NormalizeNick(nick string) string {
	return strings.TrimSpace(nick)
}

// NickKey is the case-insensitive uniqueness key.
func NickKey(nick string) string {
	return strings.ToLower(NormalizeNick(nick))
}

// ValidateCredentials checks nick/password length rules after trim.
// Empty / all-whitespace nick, out-of-range lengths → ErrInvalidInput.
func ValidateCredentials(nick, password string) error {
	nick = NormalizeNick(nick)
	n := utf8.RuneCountInString(nick)
	if n < nickMinLen || n > nickMaxLen {
		return ErrInvalidInput
	}
	p := utf8.RuneCountInString(password)
	if p < passwordMinLen || p > passwordMaxLen {
		return ErrInvalidInput
	}
	return nil
}
