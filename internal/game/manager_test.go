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
	b[0] = 1   // 1 белая на пункте 1 (последняя, всё остальное уже выкинуто)
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
