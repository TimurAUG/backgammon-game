package auth

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrNickTaken is returned when nick_lower already exists.
	ErrNickTaken = errors.New("auth: nick taken")
	// ErrInvalidCredentials is returned for unknown nick or wrong password.
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	// ErrNotFound is returned when a session or user row is missing.
	ErrNotFound = errors.New("auth: not found")
)

// User is a persisted account row (no plaintext password).
type User struct {
	ID           string
	Nick         string
	NickLower    string
	PasswordHash string
	CreatedAt    time.Time
}

// Session is a persisted login cookie row.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Store persists users and sessions. Implementations: MemoryStore, PostgresStore.
type Store interface {
	CreateUser(ctx context.Context, nick, password string) (*User, error)
	Authenticate(ctx context.Context, nick, password string) (*User, error)
	CreateSession(ctx context.Context, userID string, expiresAt time.Time) (*Session, error)
	GetSessionUser(ctx context.Context, sessionID string, now time.Time) (*User, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

// MemoryStore is an in-process Store for local/dev without DATABASE_URL.
type MemoryStore struct {
	mu       sync.RWMutex
	users    map[string]*User    // id -> user
	byNick   map[string]string   // nick_lower -> id
	sessions map[string]*Session // id -> session
}

// NewMemoryStore returns an empty in-memory auth store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:    make(map[string]*User),
		byNick:   make(map[string]string),
		sessions: make(map[string]*Session),
	}
}

func (s *MemoryStore) CreateUser(_ context.Context, nick, password string) (*User, error) {
	nick = NormalizeNick(nick)
	if err := ValidateCredentials(nick, password); err != nil {
		return nil, err
	}
	lower := NickKey(nick)
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byNick[lower]; ok {
		return nil, ErrNickTaken
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	u := &User{
		ID:           id,
		Nick:         nick,
		NickLower:    lower,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	s.users[id] = u
	s.byNick[lower] = id
	return cloneUser(u), nil
}

func (s *MemoryStore) Authenticate(_ context.Context, nick, password string) (*User, error) {
	nick = NormalizeNick(nick)
	lower := NickKey(nick)
	s.mu.RLock()
	id, ok := s.byNick[lower]
	var u *User
	if ok {
		u = s.users[id]
	}
	s.mu.RUnlock()
	if !ok || u == nil {
		DummyVerify()
		return nil, ErrInvalidCredentials
	}
	match, err := VerifyPassword(u.PasswordHash, password)
	if err != nil || !match {
		return nil, ErrInvalidCredentials
	}
	return cloneUser(u), nil
}

func (s *MemoryStore) CreateSession(_ context.Context, userID string, expiresAt time.Time) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return nil, ErrNotFound
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	sess := &Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt.UTC(),
		CreatedAt: time.Now().UTC(),
	}
	s.sessions[id] = sess
	return cloneSession(sess), nil
}

func (s *MemoryStore) GetSessionUser(_ context.Context, sessionID string, now time.Time) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrNotFound
	}
	if !sess.ExpiresAt.After(now.UTC()) {
		return nil, ErrNotFound
	}
	u, ok := s.users[sess.UserID]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneUser(u), nil
}

func (s *MemoryStore) DeleteSession(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

func cloneUser(u *User) *User {
	cp := *u
	return &cp
}

func cloneSession(s *Session) *Session {
	cp := *s
	return &cp
}
