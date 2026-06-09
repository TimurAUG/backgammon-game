package game

import "sync"

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
