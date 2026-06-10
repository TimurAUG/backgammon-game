// Command server — точка входа backgammon-game.
//
// Слушает HTTP на addr и поднимает WS-handler в /ws.
//
// Storage выбирается по переменной окружения:
//   - DATABASE_URL задан → PostgresStorage (InitSchema вызывается при старте).
//   - иначе → in-memory.
//
// Источник случайности для бросков — crypto/rand.
package main

import (
	"context"
	crand "crypto/rand"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/transport/rest"
	"github.com/TimurAUG/backgammon-game/internal/transport/ws"
)

func main() {
	addr := flag.String("addr", ":8080", "адрес HTTP-сервера")
	flag.Parse()

	mgr, cleanup, err := buildManager(context.Background())
	if err != nil {
		log.Fatalf("init manager: %v", err)
	}
	defer cleanup()

	mux := http.NewServeMux()
	mux.Handle("/ws", ws.NewHandler(mgr))
	rest.NewHandler(mgr).Register(mux)

	// В проде (Docker/Fly) отдаём собранный SPA из STATIC_DIR с того же origin,
	// что и /ws и /api — тогда wss и проверка Origin работают без CORS/прокси.
	// Локально переменная не задана → статику отдаёт Vite-dev, Go только API/WS.
	// Паттерн "/" наименее специфичен, поэтому /ws и /api/* имеют приоритет.
	if staticDir := os.Getenv("STATIC_DIR"); staticDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
		log.Printf("serving static SPA from %s", staticDir)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func buildManager(ctx context.Context) (*game.Manager, func(), error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Print("storage: in-memory (DATABASE_URL не задан)")
		return game.NewManager(), func() {}, nil
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	storage := game.NewPostgresStorage(pool)
	if err := storage.InitSchema(ctx); err != nil {
		pool.Close()
		return nil, nil, err
	}
	log.Print("storage: postgres")
	return game.NewManagerWithStorage(storage, crand.Reader), pool.Close, nil
}
