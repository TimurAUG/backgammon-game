// Package game содержит менеджер игровых сессий: создание игр, регистрация
// игроков, доступ к состоянию.
//
// Хранение пока in-memory; persistence через Postgres появится на #36.
package game

import (
	crand "crypto/rand"
	"errors"
	"io"
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

// ErrMustUsePip возвращается из EndTurn, если у текущего игрока ещё есть
// пипсы, которые можно использовать.
var ErrMustUsePip = errors.New("must use pip")

// ErrNotYourTurn возвращается из методов хода, если действие пришло не от
// игрока, чей сейчас ход.
var ErrNotYourTurn = errors.New("not your turn")

// Game — одна партия в памяти.
//
// Содержит идентификатор, доменное состояние, источник случайности для
// бросков и две позиции для соединений игроков (индексируются по
// domain.Color). Поля conns и rolledForFirst защищены mu.
type Game struct {
	ID    string
	State domain.GameState

	mu             sync.Mutex
	conns          [2]Conn
	rng            io.Reader
	rolledForFirst [2]bool
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

// HandleMove обрабатывает MOVE от игрока цвета c. Вычисляет pip по From/To
// через NextPoint, выполняет Apply, рассылает STATE обоим и LEGAL_MOVES
// активному игроку.
//
// Возвращает ошибку, если ход не текущего игрока, неверный from/to,
// или Apply вернул ошибку. Сейчас ошибки молча проглатываются — позже
// будут конвертироваться в ERROR сообщения (#34e+).
//
// TDD plan #34 (часть 4).
func (g *Game) HandleMove(c domain.Color, from, to uint8) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State.Turn != c {
		return ErrNotYourTurn
	}
	pip, ok := findPipFor(g.State, domain.Point(from), domain.Point(to))
	if !ok {
		return errors.New("no matching pip")
	}
	newState, err := domain.Apply(g.State, domain.Move{
		From: domain.Point(from),
		To:   domain.Point(to),
		Pip:  pip,
	})
	if err != nil {
		return err
	}
	g.State = newState
	g.broadcastStateLocked()
	g.sendLegalMovesLocked(g.State.Turn)
	return nil
}

// findPipFor подбирает значение пипса из Remaining такое, что ход (from→to)
// валиден. Для выкида (to==0) предпочитает точный пипс над переборным.
func findPipFor(s domain.GameState, from, to domain.Point) (uint8, bool) {
	if to == 0 {
		// Выкид: пробуем сначала точный пипс, потом переборные.
		// Точный пипс для белого = from; для чёрного = 19 - from.
		exact := uint8(from)
		if s.Turn == domain.Black {
			exact = uint8(19 - int(from))
		}
		for _, p := range s.Dice.Remaining {
			if p == exact && domain.IsLegalBearOff(s.Board, s.Turn, from, p) {
				return p, true
			}
		}
		for _, p := range s.Dice.Remaining {
			if domain.IsLegalBearOff(s.Board, s.Turn, from, p) {
				return p, true
			}
		}
		return 0, false
	}
	for _, p := range s.Dice.Remaining {
		if domain.NextPoint(s.Turn, from, p) != to {
			continue
		}
		if domain.IsLegalStep(s.Board, s.Turn, domain.Move{From: from, To: to, Pip: p}) {
			return p, true
		}
	}
	return 0, false
}

// EndTurn передаёт ход сопернику.
//
// Если у текущего игрока ещё остались пипсы, которые можно использовать
// (см. domain.IsTurnComplete), возвращает ErrMustUsePip.
//
// При успехе обнуляет HeadConsumed, сбрасывает Dice, ставит Status в
// StatusWaitingForRoll, переключает Turn и помечает IsFirstMove[c] = false.
// Рассылает STATE обоим клиентам.
//
// Правило шести (SixBlockAllowed на финальной позиции) на этом этапе НЕ
// проверяется — будет добавлено следующим циклом (закрытие #20).
//
// TDD plan #34 (часть 5).
func (g *Game) EndTurn(c domain.Color) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State.Turn != c {
		return ErrNotYourTurn
	}
	if !domain.IsTurnComplete(g.State) {
		return ErrMustUsePip
	}

	next := domain.Black
	if c == domain.Black {
		next = domain.White
	}

	isFirstMove := g.State.IsFirstMove
	isFirstMove[c] = false

	g.State = domain.GameState{
		Board:        g.State.Board,
		Turn:         next,
		Dice:         domain.Dice{},
		BorneOff:     g.State.BorneOff,
		Status:       domain.StatusWaitingForRoll,
		HeadConsumed: [2]uint8{},
		IsFirstMove:  isFirstMove,
	}
	g.broadcastStateLocked()
	return nil
}

// RollForFirst обрабатывает сигнал «готов» от игрока c в фазе определения
// первого хода.
//
// Семантика: команда не означает «бросок» сама по себе — это сигнал готовности.
// Когда оба игрока сигналили, сервер бросает по одному кубику обоим
// (White первым, Black вторым по rng), при равенстве — переброс до победителя.
// После определения State.Turn = победитель, State.Dice = NewDice(winner,
// loser), Status = WaitingForMove. STATE рассылается обоим клиентам.
//
// Если c уже сигналил ранее — игнорирует.
//
// TDD plan #34 (часть 2).
func (g *Game) RollForFirst(c domain.Color) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.rolledForFirst[c] {
		return nil
	}
	g.rolledForFirst[c] = true

	if !(g.rolledForFirst[domain.White] && g.rolledForFirst[domain.Black]) {
		return nil
	}

	for {
		whiteVal, err := domain.RollOne(g.rng)
		if err != nil {
			return err
		}
		blackVal, err := domain.RollOne(g.rng)
		if err != nil {
			return err
		}
		if whiteVal == blackVal {
			continue
		}
		winner, winnerVal, loserVal := domain.White, whiteVal, blackVal
		if blackVal > whiteVal {
			winner, winnerVal, loserVal = domain.Black, blackVal, whiteVal
		}
		g.State.Turn = winner
		g.State.Dice = domain.NewDice(winnerVal, loserVal)
		g.State.Status = domain.StatusWaitingForMove
		break
	}

	g.broadcastStateLocked()
	g.sendLegalMovesLocked(g.State.Turn)
	return nil
}

// broadcastStateLocked рассылает текущее состояние всем подключённым
// соединениям. Вызывается с захваченным mu.
func (g *Game) broadcastStateLocked() {
	msg := StateMessage(g.State)
	for _, conn := range g.conns {
		if conn != nil {
			_ = conn.Send(msg)
		}
	}
}

// sendLegalMovesLocked отправляет LEGAL_MOVES игроку цвета c — только тому,
// чей сейчас ход. Вызывается с захваченным mu.
func (g *Game) sendLegalMovesLocked(c domain.Color) {
	if g.conns[c] == nil {
		return
	}
	moves := domain.LegalMoves(g.State)
	payload := make([]protocol.MovePayload, len(moves))
	for i, m := range moves {
		payload[i] = protocol.MovePayload{From: uint8(m.From), To: uint8(m.To), Pip: m.Pip}
	}
	_ = g.conns[c].Send(protocol.ServerMessage{Type: "LEGAL_MOVES", Moves: payload})
}

// StateMessage конвертирует доменное состояние в STATE-сообщение протокола.
func StateMessage(s domain.GameState) protocol.ServerMessage {
	board := make([]int8, len(s.Board))
	for i, v := range s.Board {
		board[i] = v
	}
	msg := protocol.ServerMessage{
		Type:   "STATE",
		Board:  board,
		Turn:   colorString(s.Turn),
		Status: statusString(s.Status),
	}
	if len(s.Dice.Remaining) > 0 || s.Dice.A != 0 || s.Dice.B != 0 {
		msg.Dice = &protocol.DicePayload{
			A:         s.Dice.A,
			B:         s.Dice.B,
			IsDouble:  s.Dice.IsDouble,
			Remaining: s.Dice.Remaining,
		}
	}
	return msg
}

func colorString(c domain.Color) string {
	if c == domain.Black {
		return "black"
	}
	return "white"
}

func statusString(s domain.GameStatus) string {
	switch s {
	case domain.StatusWaitingForMove:
		return "waitingForMove"
	case domain.StatusFinished:
		return "finished"
	default:
		return "waitingForRoll"
	}
}

// Manager хранит активные игры в памяти. Безопасен для одновременного
// доступа из горутин WS-handler'а.
type Manager struct {
	mu    sync.Mutex
	games map[string]*Game
	rng   io.Reader
}

// NewManager создаёт пустой менеджер с crypto/rand в качестве источника
// случайности для бросков.
func NewManager() *Manager {
	return &Manager{games: map[string]*Game{}, rng: crand.Reader}
}

// NewManagerWithRand — конструктор для тестов: фиксированный io.Reader
// (bytes.Reader с заранее подготовленными байтами) делает броски
// детерминированными.
func NewManagerWithRand(rng io.Reader) *Manager {
	return &Manager{games: map[string]*Game{}, rng: rng}
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
				Board:       domain.InitialBoard(),
				Turn:        domain.White,
				Status:      domain.StatusWaitingForRoll,
				IsFirstMove: [2]bool{true, true},
			},
			rng: m.rng,
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
