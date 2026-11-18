package auth

import (
	"context"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/stretchr/testify/require"
)

// Card #22 — ApplyFinishedGame / UserStats on MemoryStore.
// SPEC [AMENDED 2026-11-13 — card #24, source card #21].

func TestMemoryStore_UserStats_NewUser_Zeros(t *testing.T) {
	s := NewMemoryStore()
	u, err := s.CreateUser(context.Background(), "Ann", "password1")
	require.NoError(t, err)

	played, wins, losses, err := s.UserStats(context.Background(), u.ID)
	require.NoError(t, err)
	require.Equal(t, 0, played)
	require.Equal(t, 0, wins)
	require.Equal(t, 0, losses)
}

func TestMemoryStore_ApplyFinishedGame_WinnerAndLoser(t *testing.T) {
	s := NewMemoryStore()
	white, err := s.CreateUser(context.Background(), "White", "password1")
	require.NoError(t, err)
	black, err := s.CreateUser(context.Background(), "Black", "password1")
	require.NoError(t, err)

	require.NoError(t, s.ApplyFinishedGame(context.Background(), white.ID, black.ID, domain.White))

	p, w, l, err := s.UserStats(context.Background(), white.ID)
	require.NoError(t, err)
	require.Equal(t, 1, p)
	require.Equal(t, 1, w)
	require.Equal(t, 0, l)

	p, w, l, err = s.UserStats(context.Background(), black.ID)
	require.NoError(t, err)
	require.Equal(t, 1, p)
	require.Equal(t, 0, w)
	require.Equal(t, 1, l)
	require.Equal(t, p, w+l)
}

func TestMemoryStore_ApplyFinishedGame_EmptySeatSkipped(t *testing.T) {
	// Edge: empty user id seat skipped; other seat still counts.
	s := NewMemoryStore()
	white, err := s.CreateUser(context.Background(), "OnlyWhite", "password1")
	require.NoError(t, err)

	require.NoError(t, s.ApplyFinishedGame(context.Background(), white.ID, "", domain.White))
	p, w, l, err := s.UserStats(context.Background(), white.ID)
	require.NoError(t, err)
	require.Equal(t, 1, p)
	require.Equal(t, 1, w)
	require.Equal(t, 0, l)

	// white seated as black; white seat wins → this user loses.
	require.NoError(t, s.ApplyFinishedGame(context.Background(), "", white.ID, domain.White))
	p, w, l, err = s.UserStats(context.Background(), white.ID)
	require.NoError(t, err)
	require.Equal(t, 2, p)
	require.Equal(t, 1, w)
	require.Equal(t, 1, l)
}

func TestMemoryStore_ApplyFinishedGame_SameUserBothSeats_CountOnceWin(t *testing.T) {
	// Edge: same user id on both seats → one credit, winner-seat outcome.
	s := NewMemoryStore()
	u, err := s.CreateUser(context.Background(), "Self", "password1")
	require.NoError(t, err)

	require.NoError(t, s.ApplyFinishedGame(context.Background(), u.ID, u.ID, domain.White))
	p, w, l, err := s.UserStats(context.Background(), u.ID)
	require.NoError(t, err)
	require.Equal(t, 1, p)
	require.Equal(t, 1, w)
	require.Equal(t, 0, l)
}

func TestMemoryStore_ApplyFinishedGame_BothEmpty_NoOp(t *testing.T) {
	s := NewMemoryStore()
	require.NoError(t, s.ApplyFinishedGame(context.Background(), "", "", domain.White))
}

func TestWinPercent_Rounding(t *testing.T) {
	// Card #22 AC5 — nearest int of wins/played×100, halves away from zero.
	cases := []struct {
		wins, played, want int
	}{
		{0, 0, 0},
		{27, 42, 64},
		{1, 2, 50},
		{1, 3, 33},
		{2, 3, 67},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, WinPercent(tc.wins, tc.played),
			"wins=%d played=%d", tc.wins, tc.played)
	}
}
