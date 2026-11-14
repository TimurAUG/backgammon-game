package game

import (
	"bytes"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/stretchr/testify/require"
)

// Card #25 — opening die values persist and are recoverable after unload/rejoin
// so clients do not hang on waiting-first-roll when ROLL_FOR_FIRST no-ops.

func TestGame_RollForFirst_PersistsOpeningValues(t *testing.T) {
	// Card #25 AC3 / edge: opening values stored when decided.
	rng := bytes.NewReader([]byte{4, 2}) // white=5, black=3
	g := &Game{
		ID: "g1",
		State: domain.GameState{
			Board:       domain.InitialBoard(),
			Turn:        domain.White,
			Status:      domain.StatusWaitingForRoll,
			IsFirstMove: [2]bool{true, true},
		},
		rng: rng,
	}
	require.NoError(t, g.RollForFirst(domain.White))
	require.NoError(t, g.RollForFirst(domain.Black))

	vals := g.FirstRollValues()
	require.Equal(t, [2]uint8{5, 3}, vals)
	msg := g.FirstRollMessage()
	require.NotNil(t, msg)
	require.Equal(t, "FIRST_ROLL", msg.Type)
	require.NotNil(t, msg.FirstRoll)
	require.Equal(t, 5, msg.FirstRoll.White)
	require.Equal(t, 3, msg.FirstRoll.Black)
}

func TestGame_FirstRollMessage_NilBeforeOpening(t *testing.T) {
	g := &Game{ID: "g1"}
	require.Nil(t, g.FirstRollMessage())
}

func TestManager_RejoinAfterOpening_ResendsFirstRoll(t *testing.T) {
	// Card #25 edge: reconnect after opening decided → FIRST_ROLL recovered.
	storage := newCopyingMemoryStorage()
	rng := bytes.NewReader([]byte{4, 2})
	m := NewManagerWithStorage(storage, rng)

	gameID, whiteTok, err := m.CreateGame("Alice")
	require.NoError(t, err)
	blackTok, err := m.JoinByID(gameID, "Bob")
	require.NoError(t, err)

	white := &recordingConn{}
	black := &recordingConn{}
	_, gw, err := m.JoinGame(gameID, whiteTok, white)
	require.NoError(t, err)
	cb, gb, err := m.JoinGame(gameID, blackTok, black)
	require.NoError(t, err)
	require.Equal(t, domain.Black, cb)

	require.NoError(t, gw.RollForFirst(domain.White))
	require.NoError(t, gb.RollForFirst(domain.Black))
	require.Equal(t, [2]uint8{5, 3}, gw.FirstRollValues())

	// Unload both (simulates disconnect / page reload).
	m.Leave(gameID, domain.White, white)
	m.Leave(gameID, domain.Black, black)
	require.Equal(t, 0, m.ActiveCount())

	reconn := &recordingConn{}
	_, g2, err := m.JoinGame(gameID, whiteTok, reconn)
	require.NoError(t, err)
	require.Equal(t, [2]uint8{5, 3}, g2.FirstRollValues(),
		"opening values must survive unload via Storage")
	msg := g2.FirstRollMessage()
	require.NotNil(t, msg, "rejoin must be able to re-send FIRST_ROLL")
	require.Equal(t, 5, msg.FirstRoll.White)
	require.Equal(t, 3, msg.FirstRoll.Black)
}

func TestGame_NickOnlyState_DoesNotClearFirstRoll(t *testing.T) {
	// Card #25 edge: nick refresh must not wipe opening values on the Game.
	rng := bytes.NewReader([]byte{4, 2})
	g := &Game{
		ID: "g1",
		State: domain.GameState{
			Board:       domain.InitialBoard(),
			Turn:        domain.White,
			Status:      domain.StatusWaitingForRoll,
			IsFirstMove: [2]bool{true, true},
		},
		rng:   rng,
		nicks: [2]string{"Alice", ""},
	}
	white := &recordingConn{}
	g.conns[domain.White] = white
	require.NoError(t, g.RollForFirst(domain.White))
	require.NoError(t, g.RollForFirst(domain.Black))
	require.Equal(t, [2]uint8{5, 3}, g.FirstRollValues())

	g.mu.Lock()
	g.nicks[domain.Black] = "Bob"
	g.broadcastStateLocked()
	g.mu.Unlock()

	require.Equal(t, [2]uint8{5, 3}, g.FirstRollValues(),
		"STATE broadcast for nick update must not clear opening values")
	require.NotNil(t, findMsg(white.Messages(), "STATE"))
	require.NotNil(t, g.FirstRollMessage())
}

func findMsg(msgs []protocol.ServerMessage, typ string) *protocol.ServerMessage {
	for i := range msgs {
		if msgs[i].Type == typ {
			return &msgs[i]
		}
	}
	return nil
}
