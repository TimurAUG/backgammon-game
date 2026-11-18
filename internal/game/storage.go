package game

import (
	"sync"

	"github.com/TimurAUG/backgammon-game/internal/domain"
)

// Storage — абстрактное хранилище активных партий. Конкретная реализация
// определяет, где живут данные: в памяти (тесты, MVP), в Postgres (#36),
// или в Redis (потенциальный кэш).
//
// Все методы должны быть безопасны для одновременного использования.
type Storage interface {
	// LoadGame возвращает игру по id и true. Если игры нет — ok==false,
	// возвращаемый указатель не используется вызывающим.
	LoadGame(id string) (g *Game, ok bool)

	// SaveGame сохраняет игру. Если игра с таким id уже есть, поведение
	// зависит от реализации (memory просто перезаписывает указатель).
	SaveGame(g *Game) error

	// FindActiveGameByUserID returns one unfinished game where userID sits
	// on white or black. Empty userID never matches (including empty seats).
	// Finished games are not active. Card #32 / SPEC §2 card #34.
	FindActiveGameByUserID(userID string) (g *Game, ok bool)
}

// memoryStorage — in-memory реализация Storage поверх map. По умолчанию
// используется в NewManager и NewManagerWithRand.
type memoryStorage struct {
	mu    sync.Mutex
	games map[string]*Game
}

// NewMemoryStorage создаёт пустое in-memory хранилище.
func NewMemoryStorage() Storage {
	return &memoryStorage{games: map[string]*Game{}}
}

func (s *memoryStorage) LoadGame(id string) (*Game, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[id]
	return g, ok
}

func (s *memoryStorage) SaveGame(g *Game) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[g.ID] = g
	return nil
}

func (s *memoryStorage) FindActiveGameByUserID(userID string) (*Game, bool) {
	if userID == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var found *Game
	for _, g := range s.games {
		if g == nil {
			continue
		}
		g.mu.Lock()
		finished := g.State.Status == domain.StatusFinished
		uids := g.userIDs
		g.mu.Unlock()
		if finished {
			continue
		}
		if uids[domain.White] != userID && uids[domain.Black] != userID {
			continue
		}
		// Deterministic pick when corrupt multi-active rows exist.
		if found == nil || g.ID < found.ID {
			found = g
		}
	}
	if found == nil {
		return nil, false
	}
	return found, true
}
