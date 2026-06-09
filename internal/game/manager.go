// Package game содержит менеджер игровых сессий: создание игр, регистрация
// игроков, доступ к состоянию.
//
// Хранение пока in-memory; persistence через Postgres появится на #36.
package game

import (
	"sync"

	"github.com/TimurAUG/backgammon-game/internal/domain"
)

// Game — одна партия в памяти. Содержит идентификатор и доменное состояние.
//
// Поля, которые появятся позже: игроки (соединения), статус ожидания,
// время последнего хода, история — будут вводиться по мере надобности
// следующих циклов транспорта.
type Game struct {
	ID    string
	State domain.GameState
}

// Manager хранит активные игры в памяти. Безопасен для одновременного
// доступа из горутин WS-handler'а.
type Manager struct {
	mu    sync.Mutex
	games map[string]*Game
}

// NewManager создаёт пустой менеджер.
func NewManager() *Manager {
	return &Manager{games: map[string]*Game{}}
}

// JoinGame возвращает игру с заданным id, создавая её при необходимости
// с начальной доской и ходом белых.
//
// На данный момент функция не различает «первый/второй игрок» — это будет
// добавлено в #34 вместе с подключением соединений к игре.
func (m *Manager) JoinGame(id string) *Game {
	m.mu.Lock()
	defer m.mu.Unlock()
	if g, ok := m.games[id]; ok {
		return g
	}
	g := &Game{
		ID: id,
		State: domain.GameState{
			Board: domain.InitialBoard(),
			Turn:  domain.White,
		},
	}
	m.games[id] = g
	return g
}
