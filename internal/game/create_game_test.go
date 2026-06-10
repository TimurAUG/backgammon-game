package game

import (
	"bytes"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestManager_CreateGame_PersistsGameWithCreatorToken — CreateGame генерит
// gameID + creator-token, создаёт игру с начальной доской, резервирует
// слот White под токен создателя (Black свободен), сохраняет в storage.
// TDD plan #39.
func TestManager_CreateGame_PersistsGameWithCreatorToken(t *testing.T) {
	m := NewManager()
	// 8 байт → gameID, затем 16 байт → token (детерминированный ids-reader).
	m.ids = bytes.NewReader(append(
		[]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11},
		bytes.Repeat([]byte{0x22}, 16)...,
	))

	gameID, token, err := m.CreateGame()

	require.NoError(t, err)
	require.Equal(t, "aabbccddeeff0011", gameID)
	require.Len(t, token, 32, "16 байт → 32 hex-символа")

	g, ok := m.storage.LoadGame(gameID)
	require.True(t, ok, "игра должна быть сохранена")
	require.Equal(t, token, g.tokens[domain.White], "creator-токен в слоте White")
	require.Empty(t, g.tokens[domain.Black], "слот Black свободен до join")
}
