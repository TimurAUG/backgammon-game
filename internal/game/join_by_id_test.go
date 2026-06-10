package game

import (
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestManager_JoinByID_ReservesSecondSlot — второй игрок входит по gameID:
// сервер генерит token и кладёт в свободный слот Black. TDD plan #40.
func TestManager_JoinByID_ReservesSecondSlot(t *testing.T) {
	m := NewManager()
	gameID, creatorToken, err := m.CreateGame()
	require.NoError(t, err)

	token, err := m.JoinByID(gameID)

	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEqual(t, creatorToken, token, "у игроков разные токены")

	g, _ := m.storage.LoadGame(gameID)
	require.Equal(t, creatorToken, g.tokens[domain.White])
	require.Equal(t, token, g.tokens[domain.Black], "join-токен в слоте Black")
}

// TestManager_JoinByID_NotFound — вход в несуществующую игру → ErrGameNotFound.
func TestManager_JoinByID_NotFound_ReturnsErrGameNotFound(t *testing.T) {
	m := NewManager()

	_, err := m.JoinByID("no-such-game")

	require.ErrorIs(t, err, ErrGameNotFound)
}

// TestManager_JoinByID_Full — третий вход (оба слота заняты) → ErrRoomFull.
func TestManager_JoinByID_Full_ReturnsErrRoomFull(t *testing.T) {
	m := NewManager()
	gameID, _, err := m.CreateGame()
	require.NoError(t, err)
	_, err = m.JoinByID(gameID) // занимает Black
	require.NoError(t, err)

	_, err = m.JoinByID(gameID) // мест больше нет

	require.ErrorIs(t, err, ErrRoomFull)
}
