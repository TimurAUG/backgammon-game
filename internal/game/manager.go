// Package game содержит менеджер игровых сессий: создание игр, регистрация
// игроков, доступ к состоянию.
//
// Хранение пока in-memory; persistence через Postgres появится на #36.
package game

import (
	"errors"
	"sync"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
)

// Conn — абстрактный «канал» к одному клиенту. Реализуется на уровне
// транспорта (WS) и используется в game для рассылки сообщений.
type Conn interface {
	Send(msg protocol.ServerMessage) error
}

// ErrRoomFull возвращается из JoinGame, если в игре уже зарегистрированы оба
// игрока.
var ErrRoomFull = errors.New("room full")

// Game — одна партия в памяти.
//
// Содержит идентификатор, доменное состояние и две позиции для соединений
// игроков (индексируются по domain.Color). Поля conns защищены mu и могут
// быть nil, пока соответствующий игрок не подключился (или отключился).
type Game struct {
	ID    string
	State domain.GameState

	mu    sync.Mutex
	conns [2]Conn
}

// Opponent возвращает соединение соперника для цвета c или nil, если соперник
// не подключён.
func (g *Game) Opponent(c domain.Color) Conn {
	g.mu.Lock()
	defer g.mu.Unlock()
	if c == domain.White {
		return g.conns[domain.Black]
	}
	return g.conns[domain.White]
}

// Detach снимает регистрацию соединения цвета c. Идемпотентно.
func (g *Game) Detach(c domain.Color) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.conns[c] = nil
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

// JoinGame регистрирует соединение conn в игре с id и возвращает присвоенный
// цвет: первый присоединившийся — White, второй — Black. Если игры ещё не
// было, она создаётся с начальной доской и Turn=White.
//
// Если игра уже содержит двух игроков, возвращает ErrRoomFull.
func (m *Manager) JoinGame(id string, conn Conn) (domain.Color, *Game, error) {
	m.mu.Lock()
	g, ok := m.games[id]
	if !ok {
		g = &Game{
			ID: id,
			State: domain.GameState{
				Board: domain.InitialBoard(),
				Turn:  domain.White,
			},
		}
		m.games[id] = g
	}
	m.mu.Unlock()

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conns[domain.White] == nil {
		g.conns[domain.White] = conn
		return domain.White, g, nil
	}
	if g.conns[domain.Black] == nil {
		g.conns[domain.Black] = conn
		return domain.Black, g, nil
	}
	return 0, nil, ErrRoomFull
}
