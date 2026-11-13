package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/TimurAUG/backgammon-game/internal/auth"
	"github.com/TimurAUG/backgammon-game/internal/domain"
)

func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
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
	t.Cleanup(pool.Close)
	return pool
}

// Card #2 AC8 — sessions survive process restart (new store on same DSN).
func TestPostgresStore_SessionSurvivesNewStore(t *testing.T) {
	pool := startPostgres(t)
	store := auth.NewPostgresStore(pool)
	require.NoError(t, store.InitSchema(context.Background()))

	u, err := store.CreateUser(context.Background(), "Eve", "password1")
	require.NoError(t, err)
	sess, err := store.CreateSession(context.Background(), u.ID, auth.SessionExpiry(time.Now()))
	require.NoError(t, err)

	// Simulate restart: new store wrapper on the same pool/DSN.
	store2 := auth.NewPostgresStore(pool)
	got, err := store2.GetSessionUser(context.Background(), sess.ID, time.Now())
	require.NoError(t, err)
	require.Equal(t, "Eve", got.Nick)
}

// Card #2 AC2 — unique nick_lower enforced in Postgres.
func TestPostgresStore_DuplicateNick_ErrNickTaken(t *testing.T) {
	pool := startPostgres(t)
	store := auth.NewPostgresStore(pool)
	require.NoError(t, store.InitSchema(context.Background()))

	_, err := store.CreateUser(context.Background(), "Frank", "password1")
	require.NoError(t, err)
	_, err = store.CreateUser(context.Background(), "frank", "password2")
	require.ErrorIs(t, err, auth.ErrNickTaken)
}

// Card #22 — counters persist via Postgres ApplyFinishedGame / UserStats.
func TestPostgresStore_ApplyFinishedGame_PersistsCounters(t *testing.T) {
	pool := startPostgres(t)
	store := auth.NewPostgresStore(pool)
	require.NoError(t, store.InitSchema(context.Background()))

	white, err := store.CreateUser(context.Background(), "WPg", "password1")
	require.NoError(t, err)
	black, err := store.CreateUser(context.Background(), "BPg", "password1")
	require.NoError(t, err)

	require.NoError(t, store.ApplyFinishedGame(context.Background(), white.ID, black.ID, domain.White))

	p, w, l, err := store.UserStats(context.Background(), white.ID)
	require.NoError(t, err)
	require.Equal(t, 1, p)
	require.Equal(t, 1, w)
	require.Equal(t, 0, l)

	p, w, l, err = store.UserStats(context.Background(), black.ID)
	require.NoError(t, err)
	require.Equal(t, 1, p)
	require.Equal(t, 0, w)
	require.Equal(t, 1, l)
}
