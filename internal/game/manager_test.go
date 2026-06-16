package game_test

import (
	"bytes"
	"sync"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/stretchr/testify/require"
)

// mockConn — тестовая реализация game.Conn. Сохраняет все отправленные
// сообщения для последующего assert.
type mockConn struct {
	mu   sync.Mutex
	sent []protocol.ServerMessage
}

func (m *mockConn) Send(msg protocol.ServerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockConn) Messages() []protocol.ServerMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]protocol.ServerMessage, len(m.sent))
	copy(out, m.sent)
	return out
}

// TestGame_EndTurn_RejectsSixBlockViolation проверяет, что EndTurn отказывает
// в передаче хода, если финальная позиция нарушает правило шести (закрытие
// SPEC #20).
//
// Сетап: белый блок 6+ подряд на пунктах 7..12, чёрные шашки только на
// пункте 1 — в доме чёрных (13..18) ни одной. Это финал хода: Remaining
// пуст, IsTurnComplete=true. END_TURN должен вернуть ErrRuleOfSix.
func TestGame_EndTurn_RejectsSixBlockViolation(t *testing.T) {
	mgr := game.NewManagerWithRand(bytes.NewReader([]byte{0, 0}))
	white := &mockConn{}
	black := &mockConn{}
	_, _, err := mgr.JoinGame("g1", "", white)
	require.NoError(t, err)
	_, g, err := mgr.JoinGame("g1", "", black)
	require.NoError(t, err)

	var b domain.Board
	b[6], b[7], b[8], b[9], b[10], b[11] = 1, 1, 1, 1, 1, 1
	b[0] = -1

	g.State = domain.GameState{
		Board:       b,
		Turn:        domain.White,
		Dice:        domain.Dice{},
		Status:      domain.StatusWaitingForMove,
		IsFirstMove: [2]bool{false, true},
	}

	err = g.EndTurn(domain.White)
	require.ErrorIs(t, err, game.ErrRuleOfSix)
	require.Equal(t, domain.White, g.State.Turn, "Turn не должен переключаться при отказе")
}

// TestGame_HandleMove_TriggersGameOverMars — выкид последней (15-й) шашки
// белого. У чёрного BorneOff=0 и нет шашек ни в доме белых (1..6), ни на
// голове белых (24) → марс.
//
// Сервер должен разослать обоим STATE и GAME_OVER{winner=white, kind=mars},
// а Status перейти в Finished. LEGAL_MOVES не отправляется.
func TestGame_HandleMove_TriggersGameOverMars(t *testing.T) {
	mgr := game.NewManagerWithRand(bytes.NewReader([]byte{0, 0}))
	white := &mockConn{}
	black := &mockConn{}
	_, _, err := mgr.JoinGame("g1", "", white)
	require.NoError(t, err)
	_, g, err := mgr.JoinGame("g1", "", black)
	require.NoError(t, err)

	var b domain.Board
	b[0] = 1    // 1 белая на пункте 1 (последняя, всё остальное уже выкинуто)
	b[11] = -10 // 10 чёрных на пункте 12 (вне дома белых и не на 24)

	g.State = domain.GameState{
		Board:    b,
		Turn:     domain.White,
		Dice:     domain.NewDice(1, 2),
		BorneOff: [2]uint8{14, 0},
		Status:   domain.StatusWaitingForMove,
	}

	err = g.HandleMove(domain.White, 1, 0)
	require.NoError(t, err)
	require.Equal(t, domain.StatusFinished, g.State.Status)

	gameOver := findMessage(white.Messages(), "GAME_OVER")
	require.NotNil(t, gameOver, "white должен получить GAME_OVER")
	require.Equal(t, "white", gameOver.Winner)
	require.Equal(t, "mars", gameOver.Kind)

	gameOverBlack := findMessage(black.Messages(), "GAME_OVER")
	require.NotNil(t, gameOverBlack, "black должен получить GAME_OVER")
	require.Equal(t, "white", gameOverBlack.Winner)
	require.Equal(t, "mars", gameOverBlack.Kind)

	// LEGAL_MOVES не должно прийти.
	require.Nil(t, findMessage(white.Messages(), "LEGAL_MOVES"),
		"LEGAL_MOVES после GAME_OVER не отправляется")
}

func findMessage(msgs []protocol.ServerMessage, typ string) *protocol.ServerMessage {
	for i := range msgs {
		if msgs[i].Type == typ {
			return &msgs[i]
		}
	}
	return nil
}

// TestGame_Roll_NoLegalMoves_BroadcastsTurnSkipped — если после броска у игрока
// нет НИ ОДНОГО легального хода, сервер авто-передаёт ход и шлёт обоим
// TURN_SKIPPED с выпавшими кубиками. Иначе очередь «проскакивает» молча и игрок
// не понимает, что произошло.
func TestGame_Roll_NoLegalMoves_BroadcastsTurnSkipped(t *testing.T) {
	// rng-байты [0,1] → кубики (1,2). Позиция: все белые на голове (24), оба
	// пипса упираются в чёрных (23, 22) — ходить нечем, выкид невозможен.
	mgr := game.NewManagerWithRand(bytes.NewReader([]byte{0, 1}))
	white := &mockConn{}
	black := &mockConn{}
	_, _, err := mgr.JoinGame("g1", "tok-w", white)
	require.NoError(t, err)
	_, g, err := mgr.JoinGame("g1", "tok-b", black)
	require.NoError(t, err)

	var b domain.Board
	b[23] = 15 // пункт 24: 15 белых (голова)
	b[22] = -1 // пункт 23: 1 чёрная — блок 24→23 (пип 1)
	b[21] = -1 // пункт 22: 1 чёрная — блок 24→22 (пип 2)
	b[11] = -13
	g.State = domain.GameState{
		Board:       b,
		Turn:        domain.White,
		Status:      domain.StatusWaitingForRoll,
		IsFirstMove: [2]bool{false, false},
	}

	require.NoError(t, g.Roll(domain.White))

	skip := findMessage(white.Messages(), "TURN_SKIPPED")
	require.NotNil(t, skip, "игрок, чей ход пропущен, должен получить TURN_SKIPPED")
	require.Equal(t, "white", skip.Color)
	require.NotNil(t, skip.Dice)
	require.Equal(t, uint8(1), skip.Dice.A)
	require.Equal(t, uint8(2), skip.Dice.B)

	require.NotNil(t, findMessage(black.Messages(), "TURN_SKIPPED"),
		"соперник тоже получает TURN_SKIPPED — узнаёт, что ход перешёл к нему")
}

// TestGame_Roll_FirstMoveDoubleSix_NotSkipped — регрессия бага: на самом первом
// ходу партии дубль 6:6 ошибочно авто-пропускался как «нет ходов» (TURN_SKIPPED).
// На деле с головы можно снять две шашки (исключение головы 6:6/4:4/3:3), и ход
// обязан играться. Проверяем сквозь Roll: белым приходит LEGAL_MOVES (24→18),
// а TURN_SKIPPED — нет.
func TestGame_Roll_FirstMoveDoubleSix_NotSkipped(t *testing.T) {
	// rng-байты [5,5] → кубики (6,6): RollOne(5) == 5%6+1 == 6.
	mgr := game.NewManagerWithRand(bytes.NewReader([]byte{5, 5}))
	white := &mockConn{}
	black := &mockConn{}
	_, _, err := mgr.JoinGame("g1", "tok-w", white)
	require.NoError(t, err)
	_, g, err := mgr.JoinGame("g1", "tok-b", black)
	require.NoError(t, err)

	g.State = domain.GameState{
		Board:       domain.InitialBoard(),
		Turn:        domain.White,
		Status:      domain.StatusWaitingForRoll,
		IsFirstMove: [2]bool{true, true},
	}

	require.NoError(t, g.Roll(domain.White))

	require.Nil(t, findMessage(white.Messages(), "TURN_SKIPPED"),
		"первый ход 6:6 не должен пропускаться — с головы снимаются две шашки")
	legal := findMessage(white.Messages(), "LEGAL_MOVES")
	require.NotNil(t, legal, "белым должны прийти легальные ходы первого хода 6:6")
	require.Contains(t, legal.Moves, protocol.MovePayload{From: 24, To: 18, Pip: 6},
		"единственный первый шаг дубля 6:6 — снять шашку с головы 24→18")
}

// TestGame_Resign_OpponentWinsKoks — игрок сдаётся: партия завершается, побеждает
// соперник с коксом (3 очка) независимо от позиции на доске. Оба игрока получают
// GAME_OVER, Status → Finished. Очередь не проверяется — сдаться можно в любой
// момент. Регрессия: RESIGN не был реализован вовсе (кнопка «Сдаться» молчала).
func TestGame_Resign_OpponentWinsKoks(t *testing.T) {
	cases := []struct {
		name     string
		resigner domain.Color
		wantWin  string
	}{
		{"белый сдаётся → побеждает чёрный", domain.White, "black"},
		{"чёрный сдаётся → побеждает белый", domain.Black, "white"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := game.NewManagerWithRand(bytes.NewReader([]byte{0, 0}))
			white := &mockConn{}
			black := &mockConn{}
			_, _, err := mgr.JoinGame("g1", "tok-w", white)
			require.NoError(t, err)
			_, g, err := mgr.JoinGame("g1", "tok-b", black)
			require.NoError(t, err)

			g.State = domain.GameState{
				Board:  domain.InitialBoard(),
				Turn:   domain.White,
				Status: domain.StatusWaitingForMove,
			}

			g.Resign(tc.resigner)

			require.Equal(t, domain.StatusFinished, g.State.Status, "партия должна завершиться")
			for _, conn := range []*mockConn{white, black} {
				over := findMessage(conn.Messages(), "GAME_OVER")
				require.NotNil(t, over, "оба игрока получают GAME_OVER")
				require.Equal(t, tc.wantWin, over.Winner, "побеждает соперник сдавшегося")
				require.Equal(t, "koks", over.Kind, "сдача = поражение с коксом")
			}
		})
	}
}
