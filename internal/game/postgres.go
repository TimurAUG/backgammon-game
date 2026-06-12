package game

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TimurAUG/backgammon-game/internal/domain"
)

// PostgresStorage хранит игры в Postgres: одна строка на партию.
//
// Состояние GameState сериализуется в JSONB. Tokens и rolledForFirst —
// отдельными колонками, чтобы можно было искать игру по token (для будущего
// auth) и быстро восстанавливать соединения.
//
// Conns не сохраняются — это runtime-сущности WS-соединений. После LoadGame
// игра приходит без подключений; реконнект восстановит их.
type PostgresStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage создаёт хранилище поверх существующего пула pgx.
// Схему создаёт InitSchema, его нужно вызвать один раз при старте.
func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

// InitSchema создаёт таблицу games, если её ещё нет, и догоняет схему до
// текущей версии. Идемпотентно.
func (s *PostgresStorage) InitSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS games (
			id                     TEXT PRIMARY KEY,
			state                  JSONB NOT NULL,
			token_white            TEXT NOT NULL DEFAULT '',
			token_black            TEXT NOT NULL DEFAULT '',
			rolled_for_first_white BOOLEAN NOT NULL DEFAULT FALSE,
			rolled_for_first_black BOOLEAN NOT NULL DEFAULT FALSE
		)
	`); err != nil {
		return err
	}
	// chat — колонка этапа 14 (история чата партии). Не доменная сущность,
	// поэтому отдельной колонкой, а не внутри state-JSONB. ADD COLUMN IF NOT
	// EXISTS — идемпотентная миграция для БД, созданных до этапа 14.
	_, err := s.pool.Exec(ctx, `
		ALTER TABLE games ADD COLUMN IF NOT EXISTS chat JSONB NOT NULL DEFAULT '[]'
	`)
	return err
}

// SaveGame делает upsert: вставляет новую игру или обновляет существующую.
// Захватывает g.mu для согласованного снимка полей.
func (s *PostgresStorage) SaveGame(g *Game) error {
	g.mu.Lock()
	state := g.State
	tokens := g.tokens
	rfr := g.rolledForFirst
	chat := g.chat
	id := g.ID
	g.mu.Unlock()

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}
	chatJSON, err := json.Marshal(chat)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO games (id, state, token_white, token_black,
		                   rolled_for_first_white, rolled_for_first_black, chat)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			state                  = EXCLUDED.state,
			token_white            = EXCLUDED.token_white,
			token_black            = EXCLUDED.token_black,
			rolled_for_first_white = EXCLUDED.rolled_for_first_white,
			rolled_for_first_black = EXCLUDED.rolled_for_first_black,
			chat                   = EXCLUDED.chat
	`, id, stateJSON, tokens[0], tokens[1], rfr[0], rfr[1], chatJSON)
	return err
}

// LoadGame читает игру по id. При отсутствии (pgx.ErrNoRows) возвращает
// (nil, false). conns/rng/mu — нулевые; вызывающий обязан установить rng
// перед использованием.
func (s *PostgresStorage) LoadGame(id string) (*Game, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stateJSON, chatJSON []byte
	var tokenW, tokenB string
	var rfrW, rfrB bool
	err := s.pool.QueryRow(ctx, `
		SELECT state, token_white, token_black,
		       rolled_for_first_white, rolled_for_first_black, chat
		FROM games WHERE id = $1
	`, id).Scan(&stateJSON, &tokenW, &tokenB, &rfrW, &rfrB, &chatJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}

	var state domain.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, false
	}

	var chat []ChatMessage
	if len(chatJSON) > 0 {
		if err := json.Unmarshal(chatJSON, &chat); err != nil {
			return nil, false
		}
	}

	return &Game{
		ID:             id,
		State:          state,
		tokens:         [2]string{tokenW, tokenB},
		rolledForFirst: [2]bool{rfrW, rfrB},
		chat:           chat,
	}, true
}
