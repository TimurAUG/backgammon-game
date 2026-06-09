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

// InitSchema создаёт таблицу games, если её ещё нет. Идемпотентно.
func (s *PostgresStorage) InitSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS games (
			id                     TEXT PRIMARY KEY,
			state                  JSONB NOT NULL,
			token_white            TEXT NOT NULL DEFAULT '',
			token_black            TEXT NOT NULL DEFAULT '',
			rolled_for_first_white BOOLEAN NOT NULL DEFAULT FALSE,
			rolled_for_first_black BOOLEAN NOT NULL DEFAULT FALSE
		)
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
	id := g.ID
	g.mu.Unlock()

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO games (id, state, token_white, token_black,
		                   rolled_for_first_white, rolled_for_first_black)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			state                  = EXCLUDED.state,
			token_white            = EXCLUDED.token_white,
			token_black            = EXCLUDED.token_black,
			rolled_for_first_white = EXCLUDED.rolled_for_first_white,
			rolled_for_first_black = EXCLUDED.rolled_for_first_black
	`, id, stateJSON, tokens[0], tokens[1], rfr[0], rfr[1])
	return err
}

// LoadGame читает игру по id. При отсутствии (pgx.ErrNoRows) возвращает
// (nil, false). conns/rng/mu — нулевые; вызывающий обязан установить rng
// перед использованием.
func (s *PostgresStorage) LoadGame(id string) (*Game, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stateJSON []byte
	var tokenW, tokenB string
	var rfrW, rfrB bool
	err := s.pool.QueryRow(ctx, `
		SELECT state, token_white, token_black,
		       rolled_for_first_white, rolled_for_first_black
		FROM games WHERE id = $1
	`, id).Scan(&stateJSON, &tokenW, &tokenB, &rfrW, &rfrB)
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

	return &Game{
		ID:             id,
		State:          state,
		tokens:         [2]string{tokenW, tokenB},
		rolledForFirst: [2]bool{rfrW, rfrB},
	}, true
}
