package game

import (
	"encoding/json"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/stretchr/testify/require"
)

// Card #16 / SPEC #17 amend — create/join record session nick on the seat;
// every STATE carries nicks {white,black}; JoinByID updates the live active
// Game and broadcasts STATE so a waiting opponent sees the joiner nick.

func TestManager_CreateGame_RecordsWhiteNick(t *testing.T) {
	m := NewManager()

	gameID, _, err := m.CreateGame("Alice")
	require.NoError(t, err)

	g, ok := m.storage.LoadGame(gameID)
	require.True(t, ok)
	require.Equal(t, "Alice", g.nicks[domain.White])
	require.Empty(t, g.nicks[domain.Black])
}

func TestManager_JoinByID_RecordsBlackNick(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame("Alice")
	require.NoError(t, err)

	_, err = m.JoinByID(gameID, "Bob")
	require.NoError(t, err)

	g, ok := m.storage.LoadGame(gameID)
	require.True(t, ok)
	require.Equal(t, "Alice", g.nicks[domain.White])
	require.Equal(t, "Bob", g.nicks[domain.Black])
}

func TestManager_JoinByID_ActiveGame_UpdatesLiveNicks(t *testing.T) {
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice")
	require.NoError(t, err)

	white := &recordingConn{}
	_, gLive, err := m.JoinGame(gameID, whiteTok, white)
	require.NoError(t, err)
	require.Equal(t, "Alice", gLive.nicks[domain.White])
	require.Empty(t, gLive.nicks[domain.Black])

	_, err = m.JoinByID(gameID, "Bob")
	require.NoError(t, err)

	require.Equal(t, "Bob", gLive.nicks[domain.Black],
		"JoinByID must update the live active Game, not only a LoadGame copy")
}

func TestManager_JoinByID_ConnectedOpponent_ReceivesStateWithJoinerNick(t *testing.T) {
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice")
	require.NoError(t, err)

	white := &recordingConn{}
	_, _, err = m.JoinGame(gameID, whiteTok, white)
	require.NoError(t, err)
	white.reset()

	_, err = m.JoinByID(gameID, "Bob")
	require.NoError(t, err)

	msgs := white.Messages()
	require.NotEmpty(t, msgs, "waiting white must receive a fresh STATE after join")
	var state *protocol.ServerMessage
	for i := range msgs {
		if msgs[i].Type == "STATE" {
			state = &msgs[i]
		}
	}
	require.NotNil(t, state, "broadcast must include STATE")
	require.NotNil(t, state.Nicks)
	require.Equal(t, "Alice", state.Nicks.White)
	require.Equal(t, "Bob", state.Nicks.Black)
}

func TestStateMessage_IncludesNicks(t *testing.T) {
	s := domain.GameState{
		Board:       domain.InitialBoard(),
		Turn:        domain.White,
		Status:      domain.StatusWaitingForRoll,
		IsFirstMove: [2]bool{true, true},
	}
	nicks := [2]string{"Alice", "Bob"}

	raw, err := json.Marshal(StateMessage(s, nicks))
	require.NoError(t, err)

	var parsed struct {
		Nicks *struct {
			White string `json:"white"`
			Black string `json:"black"`
		} `json:"nicks"`
	}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	require.NotNil(t, parsed.Nicks, "STATE must include nicks")
	require.Equal(t, "Alice", parsed.Nicks.White)
	require.Equal(t, "Bob", parsed.Nicks.Black)
}

func TestStateMessage_EmptyNicks_SerializesEmptyStrings(t *testing.T) {
	// Pre-deploy / guest seats: empty nick → client shows «Гость».
	s := domain.GameState{
		Board:  domain.InitialBoard(),
		Turn:   domain.White,
		Status: domain.StatusWaitingForRoll,
	}
	raw, err := json.Marshal(StateMessage(s, [2]string{"", ""}))
	require.NoError(t, err)

	var parsed struct {
		Nicks *struct {
			White string `json:"white"`
			Black string `json:"black"`
		} `json:"nicks"`
	}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	require.NotNil(t, parsed.Nicks)
	require.Equal(t, "", parsed.Nicks.White)
	require.Equal(t, "", parsed.Nicks.Black)
}

// recordingConn is a Conn that records outbound ServerMessages.
type recordingConn struct {
	msgs []protocol.ServerMessage
}

func (c *recordingConn) Send(msg protocol.ServerMessage) error {
	c.msgs = append(c.msgs, msg)
	return nil
}

func (c *recordingConn) Messages() []protocol.ServerMessage {
	out := make([]protocol.ServerMessage, len(c.msgs))
	copy(out, c.msgs)
	return out
}

func (c *recordingConn) reset() {
	c.msgs = nil
}
