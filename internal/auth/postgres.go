package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TimurAUG/backgammon-game/internal/domain"
)

// PostgresStore persists users and sessions in the same Postgres as games.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore wraps an existing pool. Call InitSchema at process start.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// InitSchema creates users and sessions tables (idempotent).
func (s *PostgresStore) InitSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			nick          TEXT NOT NULL,
			nick_lower    TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS sessions (
			id         TEXT PRIMARY KEY,
			user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);
		CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);
	`); err != nil {
		return err
	}
	// Card #22 — finished-game counters on users (defaults keep old rows at zero).
	if _, err := s.pool.Exec(ctx, `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS played INT NOT NULL DEFAULT 0
	`); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS wins INT NOT NULL DEFAULT 0
	`); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS losses INT NOT NULL DEFAULT 0
	`)
	return err
}

func (s *PostgresStore) CreateUser(ctx context.Context, nick, password string) (*User, error) {
	nick = NormalizeNick(nick)
	if err := ValidateCredentials(nick, password); err != nil {
		return nil, err
	}
	lower := NickKey(nick)
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (id, nick, nick_lower, password_hash, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, nick, lower, hash, now)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrNickTaken
		}
		return nil, err
	}
	return &User{
		ID:           id,
		Nick:         nick,
		NickLower:    lower,
		PasswordHash: hash,
		CreatedAt:    now,
	}, nil
}

func (s *PostgresStore) Authenticate(ctx context.Context, nick, password string) (*User, error) {
	nick = NormalizeNick(nick)
	lower := NickKey(nick)
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, nick, nick_lower, password_hash, created_at
		FROM users WHERE nick_lower = $1
	`, lower).Scan(&u.ID, &u.Nick, &u.NickLower, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			DummyVerify()
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	match, vErr := VerifyPassword(u.PasswordHash, password)
	if vErr != nil || !match {
		return nil, ErrInvalidCredentials
	}
	return &u, nil
}

func (s *PostgresStore) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (*Session, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`, id, userID, expiresAt.UTC(), now)
	if err != nil {
		return nil, err
	}
	return &Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt.UTC(),
		CreatedAt: now,
	}, nil
}

func (s *PostgresStore) GetSessionUser(ctx context.Context, sessionID string, now time.Time) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.nick, u.nick_lower, u.password_hash, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > $2
	`, sessionID, now.UTC()).Scan(&u.ID, &u.Nick, &u.NickLower, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

func (s *PostgresStore) UserStats(ctx context.Context, userID string) (played, wins, losses int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT played, wins, losses FROM users WHERE id = $1
	`, userID).Scan(&played, &wins, &losses)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, 0, ErrNotFound
		}
		return 0, 0, 0, err
	}
	return played, wins, losses, nil
}

func (s *PostgresStore) ApplyFinishedGame(ctx context.Context, whiteUserID, blackUserID string, winner domain.Color) error {
	if whiteUserID == "" && blackUserID == "" {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	inc := func(userID string, won bool) error {
		if userID == "" {
			return nil
		}
		if won {
			_, err := tx.Exec(ctx, `
				UPDATE users SET played = played + 1, wins = wins + 1 WHERE id = $1
			`, userID)
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE users SET played = played + 1, losses = losses + 1 WHERE id = $1
		`, userID)
		return err
	}

	if whiteUserID != "" && whiteUserID == blackUserID {
		if err := inc(whiteUserID, true); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if err := inc(whiteUserID, winner == domain.White); err != nil {
		return err
	}
	if err := inc(blackUserID, winner == domain.Black); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
