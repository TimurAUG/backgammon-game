package game_test

import (
	"bytes"
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

// startPostgresPool поднимает одноразовый Postgres-контейнер и возвращает пул,
// закрываемый по завершении теста.
func startPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

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
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// TestManager_Postgres_BroadcastReachesBothPlayers — с PostgresStorage
// (LoadGame отдаёт КОПИЮ игры, conns не сохраняются) два игрока всё равно
// должны делить один live-объект партии, иначе broadcast не дойдёт до
// соперника. Без in-memory реестра в Manager white и black оказались бы в
// разных объектах Game: каждый видел бы только свой rolledForFirst, розыгрыш
// не запустился бы, и FIRST_ROLL не получил бы никто — тот самый баг
// «оба бросают за первый ход, ничего не происходит» после перехода на БД.
//
// TDD plan #42.
func TestManager_Postgres_BroadcastReachesBothPlayers(t *testing.T) {
	if testing.Short() {
		t.Skip("skip postgres integration test in short mode")
	}
	pool := startPostgresPool(t)
	storage := game.NewPostgresStorage(pool)
	require.NoError(t, storage.InitSchema(context.Background()))

	// rng [0,1] → розыгрыш white=1, black=2 (не равны, без переброса).
	mgr := game.NewManagerWithStorage(storage, bytes.NewReader([]byte{0, 1}))

	white := &mockConn{}
	black := &mockConn{}
	_, gw, err := mgr.JoinGame("g1", "tok-w", white)
	require.NoError(t, err)
	_, gb, err := mgr.JoinGame("g1", "tok-b", black)
	require.NoError(t, err)

	// Каждый игрок сигналит через свой объект игры — как два независимых
	// WS-handler'а. При общем live-объекте оба флага накапливаются и розыгрыш
	// проходит.
	require.NoError(t, gw.RollForFirst(domain.White))
	require.NoError(t, gb.RollForFirst(domain.Black))

	require.NotNil(t, findMessage(white.Messages(), "FIRST_ROLL"),
		"white должен получить FIRST_ROLL — оба игрока делят один live-объект игры")
	require.NotNil(t, findMessage(black.Messages(), "FIRST_ROLL"),
		"black должен получить FIRST_ROLL")
}

// TestManager_Leave_UnloadsGameWhenBothDisconnected — партия остаётся в
// реестре, пока подключён хотя бы один игрок; после ухода обоих выгружается
// (память не течёт). Состояние при этом уже в Storage.
//
// TDD plan #43.
func TestManager_Leave_UnloadsGameWhenBothDisconnected(t *testing.T) {
	mgr := game.NewManagerWithRand(bytes.NewReader([]byte{0, 0}))
	cw, _, err := mgr.JoinGame("g1", "tok-w", &mockConn{})
	require.NoError(t, err)
	cb, _, err := mgr.JoinGame("g1", "tok-b", &mockConn{})
	require.NoError(t, err)

	mgr.Leave("g1", cw)
	require.Equal(t, 1, mgr.ActiveCount(),
		"пока подключён хотя бы один игрок — партия остаётся в реестре")

	mgr.Leave("g1", cb)
	require.Equal(t, 0, mgr.ActiveCount(),
		"после ухода обоих партия выгружается из реестра")
}

// TestManager_RejoinAfterUnload_RestoresSlotFromStorage — после выгрузки
// повторный JOIN тем же токеном поднимает партию из Storage и возвращает
// прежний цвет (состояние партии переживает выгрузку из памяти).
//
// TDD plan #43.
func TestManager_RejoinAfterUnload_RestoresSlotFromStorage(t *testing.T) {
	mgr := game.NewManagerWithRand(bytes.NewReader([]byte{0, 0}))
	cw, _, err := mgr.JoinGame("g1", "tok-w", &mockConn{})
	require.NoError(t, err)
	cb, _, err := mgr.JoinGame("g1", "tok-b", &mockConn{})
	require.NoError(t, err)
	mgr.Leave("g1", cw)
	mgr.Leave("g1", cb)

	color, _, err := mgr.JoinGame("g1", "tok-w", &mockConn{})
	require.NoError(t, err)
	require.Equal(t, cw, color, "реконнект по токену после выгрузки возвращает прежний слот")
}
