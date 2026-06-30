package game_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/stretchr/testify/require"
)

// TestLegalMovesMessageFor_IncludesReachChains: сообщение LEGAL_MOVES несёт не
// только одиночные шаги (moves), но и составные цели одной шашки (reach) — куда
// шашка дойдёт несколькими кубиками. Белая на 13, кубики 2 и 4, дорожка пуста →
// в reach есть цель 7 (13→…→7) на два кубика, а одиночные шаги остаются в moves.
// Плюс проверяем wire-контракт: ключи reach/path/pips в JSON.
//
// SPEC #51.
func TestLegalMovesMessageFor_IncludesReachChains(t *testing.T) {
	mgr := game.NewManagerWithRand(bytes.NewReader([]byte{0, 0}))
	white := &mockConn{}
	black := &mockConn{}
	_, _, err := mgr.JoinGame("g1", "", white)
	require.NoError(t, err)
	_, g, err := mgr.JoinGame("g1", "", black)
	require.NoError(t, err)

	var b domain.Board
	b[12] = 1 // белая на 13, дорожка 11/9/7 пуста
	g.State = domain.GameState{
		Board:  b,
		Turn:   domain.White,
		Dice:   domain.NewDice(2, 4),
		Status: domain.StatusWaitingForMove,
	}

	msg := g.LegalMovesMessageFor(domain.White)
	require.NotNil(t, msg)
	require.Equal(t, "LEGAL_MOVES", msg.Type)
	require.NotEmpty(t, msg.Moves, "одиночные шаги остаются в moves")

	var combined *protocol.ReachPayload
	for i := range msg.Reach {
		r := &msg.Reach[i]
		if len(r.Path) > 0 && r.Path[len(r.Path)-1] == 7 {
			combined = r
		}
	}
	require.NotNil(t, combined, "reach должен содержать составную цель 7")
	require.Equal(t, uint8(13), combined.From)
	require.Len(t, combined.Pips, 2, "до 7 идём двумя кубиками")
	require.Len(t, combined.Path, 2)
	require.Equal(t, 7, combined.Path[len(combined.Path)-1])

	raw, err := json.Marshal(msg)
	require.NoError(t, err)
	for _, key := range []string{`"reach"`, `"path"`, `"pips"`} {
		require.True(t, strings.Contains(string(raw), key), "JSON LEGAL_MOVES должен содержать "+key)
	}
}
