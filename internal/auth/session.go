package auth

import (
	"net/http"
	"strings"
	"time"
)

const (
	// CookieName is the session cookie set by register/login.
	CookieName = "nardy_session"
	// SessionTTL is the cookie Max-Age and session expires_at window.
	SessionTTL = 30 * 24 * time.Hour
	// sessionMaxAgeSeconds is Max-Age for Set-Cookie.
	sessionMaxAgeSeconds = int(SessionTTL / time.Second)
)

// SetSessionCookie writes the session id cookie onto the response.
// Secure is true when the request is HTTPS (TLS or X-Forwarded-Proto: https).
func SetSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   sessionMaxAgeSeconds,
		HttpOnly: true,
		Secure:   requestIsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie expires the session cookie (logout / idempotent clear).
func ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   requestIsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// SessionIDFromRequest reads the session cookie value, if any.
func SessionIDFromRequest(r *http.Request) (string, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}

func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// SessionExpiry returns now + SessionTTL in UTC.
func SessionExpiry(now time.Time) time.Time {
	return now.UTC().Add(SessionTTL)
}
