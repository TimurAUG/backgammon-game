package game

import (
	"context"
	"sync"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/stretchr/testify/require"
)

// Card #22 — seat user ids + finished-game stats attribution.
// SPEC [AMENDED 2026-11-13 — card #24, source card #21].

type recordingStats struct {
	mu    sync.Mutex
	calls []statsCall
}

type statsCall struct {
	white, black string
	winner       domain.Color
}

func (r *recordingStats) ApplyFinishedGame(_ context.Context, whiteUserID, blackUserID string, winner domain.Color) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, statsCall{white: whiteUserID, black: blackUserID, winner: winner})
	return nil
}

func (r *recordingStats) Calls() []statsCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]statsCall, len(r.calls))
	copy(out, r.calls)
	return out
}

func TestManager_CreateJoin_RecordsSeatUserIDs(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	g, ok := m.storage.LoadGame(gameID)
	require.True(t, ok)
	require.Equal(t, [2]string{"user-w", "user-b"}, g.UserIDs())
}

func TestGame_Resign_AppliesStatsOnce(t *testing.T) {
	rec := &recordingStats{}
	m := NewManager()
	m.SetStatsRecorder(rec)
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	white := &recordingConn{}
	_, g, err := m.JoinGame(gameID, whiteTok, white)
	require.NoError(t, err)

	g.Resign(domain.White) // black wins
	require.True(t, g.StatsRecorded())
	calls := rec.Calls()
	require.Len(t, calls, 1)
	require.Equal(t, "user-w", calls[0].white)
	require.Equal(t, "user-b", calls[0].black)
	require.Equal(t, domain.Black, calls[0].winner)

	// Edge: second finish / resign on finished game must not increment again.
	g.Resign(domain.White)
	require.Len(t, rec.Calls(), 1)
}

func TestGame_Unfinished_DoesNotApplyStats(t *testing.T) {
	rec := &recordingStats{}
	m := NewManager()
	m.SetStatsRecorder(rec)
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)
	_, g, err := m.JoinGame(gameID, whiteTok, &recordingConn{})
	require.NoError(t, err)

	require.NotEqual(t, domain.StatusFinished, g.State.Status)
	require.False(t, g.StatsRecorded())
	require.Empty(t, rec.Calls())
}

func TestGame_HandleMove_Finished_AppliesStats(t *testing.T) {
	rec := &recordingStats{}
	m := NewManager()
	m.SetStatsRecorder(rec)
	gameID, whiteTok, err := m.CreateGame("Alice", "user-w")
	require.NoError(t, err)
	blackTok, err := m.JoinByID(gameID, "Bob", "user-b")
	require.NoError(t, err)

	_, g, err := m.JoinGame(gameID, whiteTok, &recordingConn{})
	require.NoError(t, err)
	_, _, err = m.JoinGame(gameID, blackTok, &recordingConn{})
	require.NoError(t, err)

	var b domain.Board
	b[0] = 1
	b[11] = -10
	g.State = domain.GameState{
		Board:    b,
		Turn:     domain.White,
		Dice:     domain.NewDice(1, 2),
		BorneOff: [2]uint8{14, 0},
		Status:   domain.StatusWaitingForMove,
	}
	// Restore seat attribution after State overwrite (State assignment doesn't clear userIDs).
	require.Equal(t, [2]string{"user-w", "user-b"}, g.UserIDs())

	require.NoError(t, g.HandleMove(domain.White, 1, 0))
	require.Equal(t, domain.StatusFinished, g.State.Status)
	require.True(t, g.StatsRecorded())
	calls := rec.Calls()
	require.Len(t, calls, 1)
	require.Equal(t, domain.White, calls[0].winner)
}

func TestGame_EmptyUserIDs_MarksRecordedWithoutCall(t *testing.T) {
	rec := &recordingStats{}
	m := NewManager()
	m.SetStatsRecorder(rec)
	gameID, whiteTok, err := m.CreateGame("Alice", "")
	require.NoError(t, err)
	_, err = m.JoinByID(gameID, "Bob", "")
	require.NoError(t, err)
	_, g, err := m.JoinGame(gameID, whiteTok, &recordingConn{})
	require.NoError(t, err)

	g.Resign(domain.White)
	require.True(t, g.StatsRecorded())
	require.Empty(t, rec.Calls(), "no seat user ids → no ApplyFinishedGame call")
}
