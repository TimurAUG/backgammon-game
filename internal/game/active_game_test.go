package game

import (
	"sync"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/stretchr/testify/require"
)

// Card #32 — reclaim unfinished seat via session (active game API).
// SPEC.md §2 [NEW/AMENDED 2026-11-18 — card #34].

func TestManager_ActiveCredentials_AfterUnload_SameToken(t *testing.T) {
	// AC1 / AC2: unload keeps seat user id + token; resume returns same token; no win.
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	blackTok, err := m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	white := &recordingConn{}
	black := &recordingConn{}
	_, g, err := m.JoinGame(gameID, whiteTok, white)
	require.NoError(t, err)
	_, _, err = m.JoinGame(gameID, blackTok, black)
	require.NoError(t, err)

	m.Leave(gameID, domain.White, white)
	m.Leave(gameID, domain.Black, black)
	require.Equal(t, 0, m.ActiveCount())

	loaded, ok := m.storage.LoadGame(gameID)
	require.True(t, ok)
	require.NotEqual(t, domain.StatusFinished, loaded.State.Status)
	require.Equal(t, [2]string{"user-w", "user-b"}, loaded.UserIDs())
	require.Equal(t, whiteTok, loaded.Tokens()[domain.White])
	require.Equal(t, blackTok, loaded.Tokens()[domain.Black])
	require.False(t, loaded.StatsRecorded())
	require.Empty(t, white.Messages(), "unload must not award a win / GAME_OVER")

	gotID, gotTok, err := m.ActiveCredentials("user-w")
	require.NoError(t, err)
	require.Equal(t, gameID, gotID)
	require.Equal(t, whiteTok, gotTok)

	gotID, gotTok, err = m.ActiveCredentials("user-b")
	require.NoError(t, err)
	require.Equal(t, gameID, gotID)
	require.Equal(t, blackTok, gotTok)

	_ = g
}

func TestManager_ActiveCredentials_FinishedGame_NotFound(t *testing.T) {
	// AC4: finished (resign) is not active — credentials not returned.
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	_, g, err := m.JoinGame(gameID, whiteTok, &recordingConn{})
	require.NoError(t, err)
	g.Resign(domain.White)

	_, _, err = m.ActiveCredentials("user-w")
	require.ErrorIs(t, err, ErrNoActiveGame)
	_, _, err = m.ActiveCredentials("user-b")
	require.ErrorIs(t, err, ErrNoActiveGame)
}

func TestManager_ActiveCredentials_NoGame_NotFound(t *testing.T) {
	// AC3: no unfinished game → not found; no invented credentials.
	m := NewManager()
	gameID, tok, err := m.ActiveCredentials("nobody")
	require.ErrorIs(t, err, ErrNoActiveGame)
	require.Empty(t, gameID)
	require.Empty(t, tok)
}

func TestManager_ActiveCredentials_EmptyUserID_NotFound(t *testing.T) {
	// Edge: empty userID never matches seats (including empty seat columns).
	m := NewManager()
	_, _, err := m.CreateGame("guest-host", "")
	require.NoError(t, err)

	gameID, tok, err := m.ActiveCredentials("")
	require.ErrorIs(t, err, ErrNoActiveGame)
	require.Empty(t, gameID)
	require.Empty(t, tok)
}

func TestManager_ActiveCredentials_WaitingForOpponent_ReturnsWhiteToken(t *testing.T) {
	// AC7: host waiting for opponent — still reclaimable.
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)

	gotID, gotTok, err := m.ActiveCredentials("user-w")
	require.NoError(t, err)
	require.Equal(t, gameID, gotID)
	require.Equal(t, whiteTok, gotTok)
}

func TestManager_CreateGame_WhenActiveExists_ReturnsErr(t *testing.T) {
	// AC5: create while unfinished exists → conflict; no second seat.
	m := NewManager()
	firstID, firstTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)

	secondID, secondTok, err := m.CreateGame("Alice", "user-w")
	require.ErrorIs(t, err, ErrActiveGameExists)
	require.Empty(t, secondID)
	require.Empty(t, secondTok)

	gotID, gotTok, err := m.ActiveCredentials("user-w")
	require.NoError(t, err)
	require.Equal(t, firstID, gotID)
	require.Equal(t, firstTok, gotTok)
}

func TestManager_JoinByID_WhenOtherActiveExists_ReturnsErr(t *testing.T) {
	// AC6: join a different game while unfinished exists → refused.
	m := NewManager()
	ownID, ownTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	otherID, _, err := m.CreateGame("Bob", "user-b")
	require.NoError(t, err)

	tok, err := m.JoinByID(otherID, "Alice", "user-w")
	require.ErrorIs(t, err, ErrActiveGameExists)
	require.Empty(t, tok)

	gotID, gotTok, err := m.ActiveCredentials("user-w")
	require.NoError(t, err)
	require.Equal(t, ownID, gotID)
	require.Equal(t, ownTok, gotTok)
}

func TestManager_JoinByID_AlreadySeatedSameGame_ReturnsExistingToken(t *testing.T) {
	// Edge: join same unfinished game → existing seat token; do not take other color.
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)

	tok, err := m.JoinByID(gameID, "Alice", "user-w")
	require.NoError(t, err)
	require.Equal(t, whiteTok, tok)

	g, ok := m.storage.LoadGame(gameID)
	require.True(t, ok)
	require.Equal(t, whiteTok, g.Tokens()[domain.White])
	require.Empty(t, g.Tokens()[domain.Black], "must not occupy the other color")
	require.Equal(t, [2]string{"user-w", ""}, g.UserIDs())
}

func TestManager_CreateGame_ConcurrentSameUser_AtMostOneSeat(t *testing.T) {
	// Edge: concurrent create → ≤1 unfinished game; at most one success.
	m := NewManager()
	const n = 16
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		okCount  int
		errCount int
		ids      []string
	)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			id, tok, err := m.CreateGame("Alice", "user-w")
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				okCount++
				ids = append(ids, id)
				require.NotEmpty(t, tok)
				return
			}
			require.ErrorIs(t, err, ErrActiveGameExists)
			errCount++
		}()
	}
	wg.Wait()
	require.Equal(t, 1, okCount)
	require.Equal(t, n-1, errCount)
	require.Len(t, ids, 1)

	gotID, _, err := m.ActiveCredentials("user-w")
	require.NoError(t, err)
	require.Equal(t, ids[0], gotID)
}

func TestMemoryStorage_FindActiveGameByUserID(t *testing.T) {
	s := NewMemoryStorage()
	m := NewManagerWithStorage(s, nil)
	gameID, _, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)

	g, ok := s.FindActiveGameByUserID("user-w")
	require.True(t, ok)
	require.Equal(t, gameID, g.ID)

	_, ok = s.FindActiveGameByUserID("")
	require.False(t, ok)

	_, ok = s.FindActiveGameByUserID("nobody")
	require.False(t, ok)
}
