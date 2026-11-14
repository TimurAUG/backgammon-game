package game

import (
	"encoding/json"
	"sync"
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
	// Pre-deploy / guest seats: empty nick → client shows waiting / guest label.
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

// Card #25 — JoinByID vs JoinGame must share one live *Game when Storage
// returns a fresh copy on every LoadGame (Postgres). Otherwise the host's
// active object keeps nicks[black]=="" while Postgres already has the joiner.
func TestManager_JoinByID_JoinGameRace_LiveNicksMatchStorage(t *testing.T) {
	// Card #25 edge: concurrent JoinByID / JoinGame must not split live vs LoadGame.
	for i := 0; i < 40; i++ {
		storage := newCopyingMemoryStorage()
		m := NewManagerWithStorage(storage, nil)
		gameID, whiteTok, err := m.CreateGame("Alice")
		require.NoError(t, err)

		var (
			wg   sync.WaitGroup
			live *Game
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = m.JoinByID(gameID, "Bob")
		}()
		go func() {
			defer wg.Done()
			_, g, err := m.JoinGame(gameID, whiteTok, &recordingConn{})
			require.NoError(t, err)
			live = g
		}()
		wg.Wait()

		require.NotNil(t, live)
		require.Equal(t, "Bob", live.nicks[domain.Black],
			"live active Game must carry joiner nick after JoinByID/JoinGame race")
		loaded, ok := storage.LoadGame(gameID)
		require.True(t, ok)
		require.Equal(t, "Bob", loaded.nicks[domain.Black],
			"storage must carry joiner nick")
	}
}

// Card #25 — host still on «Игра создана» (no WS) when joiner REST-joins:
// nick is in storage; host's first STATE after JoinGame shows joiner nick.
func TestManager_JoinByID_BeforeHostWS_HostFirstStateHasJoinerNick(t *testing.T) {
	// Card #25 edge: host enters after REST join → first STATE has joiner nick.
	storage := newCopyingMemoryStorage()
	m := NewManagerWithStorage(storage, nil)
	gameID, whiteTok, err := m.CreateGame("Alice")
	require.NoError(t, err)

	_, err = m.JoinByID(gameID, "Bob")
	require.NoError(t, err)

	white := &recordingConn{}
	_, g, err := m.JoinGame(gameID, whiteTok, white)
	require.NoError(t, err)
	require.Equal(t, "Bob", g.nicks[domain.Black])

	msg := StateMessage(g.State, g.Nicks())
	require.NotNil(t, msg.Nicks)
	require.Equal(t, "Bob", msg.Nicks.Black)
}

// copyingMemoryStorage mimics Postgres LoadGame: every load returns a new
// *Game copy so JoinByID / JoinGame can diverge without the registry fix.
type copyingMemoryStorage struct {
	mu    sync.Mutex
	games map[string]gameSnapshot
}

type gameSnapshot struct {
	id             string
	state          domain.GameState
	tokens         [2]string
	nicks          [2]string
	rolledForFirst [2]bool
	firstRoll      [2]uint8
	chat           []ChatMessage
}

func newCopyingMemoryStorage() *copyingMemoryStorage {
	return &copyingMemoryStorage{games: map[string]gameSnapshot{}}
}

func (s *copyingMemoryStorage) snapshotLocked(g *Game) gameSnapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	chat := append([]ChatMessage(nil), g.chat...)
	return gameSnapshot{
		id:             g.ID,
		state:          g.State,
		tokens:         g.tokens,
		nicks:          g.nicks,
		rolledForFirst: g.rolledForFirst,
		firstRoll:      g.firstRoll, // read under g.mu — do not call FirstRollValues (would re-lock)
		chat:           chat,
	}
}

func (s *copyingMemoryStorage) SaveGame(g *Game) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[g.ID] = s.snapshotLocked(g)
	return nil
}

func (s *copyingMemoryStorage) LoadGame(id string) (*Game, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.games[id]
	if !ok {
		return nil, false
	}
	chat := append([]ChatMessage(nil), snap.chat...)
	return &Game{
		ID:             snap.id,
		State:          snap.state,
		tokens:         snap.tokens,
		nicks:          snap.nicks,
		rolledForFirst: snap.rolledForFirst,
		firstRoll:      snap.firstRoll,
		chat:           chat,
	}, true
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
