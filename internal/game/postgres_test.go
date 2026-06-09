package game_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
)

// TestPostgresStorage_SaveAndLoad — round-trip: SaveGame + LoadGame через
// реальный Postgres-контейнер. Проверяет, что доменное состояние, токены и
// rolledForFirst сохраняются и восстанавливаются эквивалентно.
//
// TDD plan #36.
func TestPostgresStorage_SaveAndLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skip postgres integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("nardy"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	storage := game.NewPostgresStorage(pool)
	require.NoError(t, storage.InitSchema(ctx))

	// Создаём игру с произвольным заполненным состоянием. Доступ к
	// приватным tokens/rolledForFirst идёт через метод-геттеры; для setup
	// используем JoinGame + прямую правку публичных полей.
	mgr := game.NewManagerWithStorage(storage, nil)
	white := &mockConn{}
	black := &mockConn{}
	_, _, err = mgr.JoinGame("g1", "tok-w", white)
	require.NoError(t, err)
	_, original, err := mgr.JoinGame("g1", "tok-b", black)
	require.NoError(t, err)

	original.State = domain.GameState{
		Board:        domain.InitialBoard(),
		Turn:         domain.Black,
		Dice:         domain.NewDice(5, 3),
		BorneOff:     [2]uint8{2, 4},
		Status:       domain.StatusWaitingForMove,
		HeadConsumed: [2]uint8{1, 0},
		IsFirstMove:  [2]bool{false, true},
	}

	require.NoError(t, storage.SaveGame(original))

	loaded, ok := storage.LoadGame("g1")
	require.True(t, ok, "игра должна быть найдена")
	require.Equal(t, "g1", loaded.ID)
	require.Equal(t, original.State.Board, loaded.State.Board)
	require.Equal(t, original.State.Turn, loaded.State.Turn)
	require.Equal(t, original.State.Dice.A, loaded.State.Dice.A)
	require.Equal(t, original.State.Dice.B, loaded.State.Dice.B)
	require.Equal(t, original.State.Dice.IsDouble, loaded.State.Dice.IsDouble)
	require.Equal(t, original.State.Dice.Remaining, loaded.State.Dice.Remaining)
	require.Equal(t, original.State.BorneOff, loaded.State.BorneOff)
	require.Equal(t, original.State.Status, loaded.State.Status)
	require.Equal(t, original.State.HeadConsumed, loaded.State.HeadConsumed)
	require.Equal(t, original.State.IsFirstMove, loaded.State.IsFirstMove)
	require.Equal(t, original.Tokens(), loaded.Tokens())
	require.Equal(t, original.RolledForFirst(), loaded.RolledForFirst())

	_, missing := storage.LoadGame("nope")
	require.False(t, missing, "несуществующая игра — ok=false")

	// auto-save: новая партия после JoinGame должна попадать в БД без явного
	// SaveGame от вызывающего.
	autoMgr := game.NewManagerWithStorage(storage, nil)
	autoConn := &mockConn{}
	_, _, err = autoMgr.JoinGame("g-auto", "auto-tok", autoConn)
	require.NoError(t, err)

	autoLoaded, ok := storage.LoadGame("g-auto")
	require.True(t, ok, "JoinGame должен auto-save игру в Storage")
	require.Equal(t, [2]string{"auto-tok", ""}, autoLoaded.Tokens())
	require.Equal(t, int8(15), autoLoaded.State.Board[23],
		"начальная доска: 15 белых на пункте 24")
}
