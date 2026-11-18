package game

import (
	"sync"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/stretchr/testify/require"
)

// Card #29 / SPEC.md §2 [NEW 2026-11-18 — card #31]: creator cancel deletes
// an unfinished game without finishing or recording stats.

func TestManager_CancelGame_RemovesFromStorage(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)

	require.NoError(t, m.CancelGame(gameID, "user-w"))

	_, ok := m.storage.LoadGame(gameID)
	require.False(t, ok, "cancelled game must be gone from storage")
	require.Equal(t, 0, m.ActiveCount())
}

func TestManager_CancelGame_AfterJoin_RemovesGame(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	require.NoError(t, m.CancelGame(gameID, "user-w"))
	_, err = m.JoinByID(gameID, "Carol", "user-c")
	require.ErrorIs(t, err, ErrGameNotFound)
}

func TestManager_CancelGame_NotifiesLiveConns_GAME_NOT_FOUND(t *testing.T) {
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	blackTok, err := m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	white := &recordingConn{}
	black := &recordingConn{}
	_, _, err = m.JoinGame(gameID, whiteTok, white)
	require.NoError(t, err)
	_, _, err = m.JoinGame(gameID, blackTok, black)
	require.NoError(t, err)
	white.reset()
	black.reset()

	require.NoError(t, m.CancelGame(gameID, "user-w"))

	for _, c := range []*recordingConn{white, black} {
		msgs := c.Messages()
		require.NotEmpty(t, msgs, "live seat must get GAME_NOT_FOUND")
		require.Equal(t, "ERROR", msgs[0].Type)
		require.Equal(t, "GAME_NOT_FOUND", msgs[0].Code)
		for _, msg := range msgs {
			require.NotEqual(t, "GAME_OVER", msg.Type, "cancel must not finish the game")
		}
	}
	require.Equal(t, 0, m.ActiveCount())
}

func TestManager_CancelGame_DoesNotRecordStats(t *testing.T) {
	rec := &recordingStats{}
	m := NewManager()
	m.SetStatsRecorder(rec)
	gameID, _, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	require.NoError(t, m.CancelGame(gameID, "user-w"))
	require.Empty(t, rec.Calls(), "cancel must not call ApplyFinishedGame")
}

func TestManager_CancelGame_NotCreator_ReturnsErrNotCreator(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	err = m.CancelGame(gameID, "user-b")
	require.ErrorIs(t, err, ErrNotCreator)
	_, ok := m.storage.LoadGame(gameID)
	require.True(t, ok, "game must remain when caller is not creator")
}

func TestManager_CancelGame_EmptyWhiteUserID_ReturnsErrNotCreator(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame("Alice", "")
	require.NoError(t, err)

	err = m.CancelGame(gameID, "user-w")
	require.ErrorIs(t, err, ErrNotCreator)
	_, ok := m.storage.LoadGame(gameID)
	require.True(t, ok)
}

func TestManager_CancelGame_UnknownID_ReturnsErrGameNotFound(t *testing.T) {
	m := NewManager()
	err := m.CancelGame("missing", "user-w")
	require.ErrorIs(t, err, ErrGameNotFound)
}

func TestManager_CancelGame_SecondCall_ReturnsErrGameNotFound(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)

	require.NoError(t, m.CancelGame(gameID, "user-w"))
	err = m.CancelGame(gameID, "user-w")
	require.ErrorIs(t, err, ErrGameNotFound)
}

func TestManager_CancelGame_Finished_ReturnsErrGameFinished(t *testing.T) {
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)
	_, g, err := m.JoinGame(gameID, whiteTok, &recordingConn{})
	require.NoError(t, err)
	g.Resign(domain.White)
	require.Equal(t, domain.StatusFinished, g.State.Status)

	err = m.CancelGame(gameID, "user-w")
	require.ErrorIs(t, err, ErrGameFinished)
	_, ok := m.storage.LoadGame(gameID)
	require.True(t, ok, "finished game row must stay")
}

func TestManager_JoinGame_MissingID_ReturnsErrGameNotFound(t *testing.T) {
	m := NewManager()
	_, _, err := m.JoinGame("no-such", "tok", &recordingConn{})
	require.ErrorIs(t, err, ErrGameNotFound)
	require.Equal(t, 0, m.ActiveCount())
}

func TestManager_CancelGame_ConcurrentDoubleDelete_OneWins(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- m.CancelGame(gameID, "user-w")
		}()
	}
	wg.Wait()
	close(errs)

	var okCount, notFound int
	for err := range errs {
		if err == nil {
			okCount++
		} else if err == ErrGameNotFound {
			notFound++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	require.Equal(t, 1, okCount)
	require.Equal(t, 1, notFound)
	_, ok := m.storage.LoadGame(gameID)
	require.False(t, ok)
}

func TestManager_CancelGame_DoesNotEmitGAME_OVER(t *testing.T) {
	m := NewManager()
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	white := &recordingConn{}
	_, g, err := m.JoinGame(gameID, whiteTok, white)
	require.NoError(t, err)
	white.reset()

	require.NoError(t, m.CancelGame(gameID, "user-w"))
	require.NotEqual(t, domain.StatusFinished, g.State.Status)
	msgs := white.Messages()
	require.Len(t, msgs, 1)
	require.Equal(t, "ERROR", msgs[0].Type)
	require.Equal(t, "GAME_NOT_FOUND", msgs[0].Code)
	require.NotEqual(t, "GAME_OVER", msgs[0].Type)
}

// Card #29 / Code Review edge: WS still holds live *Game after CancelGame.
// Resign / Roll / HandleMove / EndTurn must not resurrect the row, record
// stats, emit GAME_OVER, or finish the in-memory state.
func TestManager_CancelGame_LiveSeatActions_DoNotResurrectOrRecordStats(t *testing.T) {
	rec := &recordingStats{}
	m := NewManager()
	m.SetStatsRecorder(rec)
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
	white.reset()
	black.reset()

	require.NoError(t, m.CancelGame(gameID, "user-w"))

	// Resign is the critical zombie path: finishes + maybePersist SaveGame + stats.
	g.Resign(domain.White)
	_, ok := m.storage.LoadGame(gameID)
	require.False(t, ok, "post-cancel Resign must not SaveGame-resurrect the deleted id")
	require.Empty(t, rec.Calls(), "post-cancel Resign must not ApplyFinishedGame")
	require.NotEqual(t, domain.StatusFinished, g.State.Status,
		"post-cancel Resign must not finish the live *Game")

	require.ErrorIs(t, g.Roll(domain.White), ErrGameNotFound)
	require.ErrorIs(t, g.HandleMove(domain.White, 24, 23), ErrGameNotFound)
	require.ErrorIs(t, g.EndTurn(domain.White), ErrGameNotFound)
	require.ErrorIs(t, g.RollForFirst(domain.White), ErrGameNotFound)

	_, ok = m.storage.LoadGame(gameID)
	require.False(t, ok, "post-cancel mutate paths must not resurrect the deleted id")
	require.Empty(t, rec.Calls())
	for _, c := range []*recordingConn{white, black} {
		for _, msg := range c.Messages() {
			require.NotEqual(t, "GAME_OVER", msg.Type)
		}
	}
}
