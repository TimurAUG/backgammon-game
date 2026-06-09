package ws_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/TimurAUG/backgammon-game/internal/transport/ws"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

// TestHandler_JoinReturnsState — интеграционный тест на минимальный поток:
//   1. Клиент открывает WS-соединение.
//   2. Шлёт JOIN с gameId.
//   3. Получает STATE с начальной доской.
//
// TDD plan #33.
func TestHandler_JoinReturnsState(t *testing.T) {
	mgr := game.NewManager()
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer conn.Close(websocket.StatusInternalError, "test cleanup")

	join, err := json.Marshal(protocol.ClientMessage{
		Type:   "JOIN",
		GameID: "g1",
		Token:  "t1",
	})
	require.NoError(t, err)
	require.NoError(t, conn.Write(ctx, websocket.MessageText, join))

	_, data, err := conn.Read(ctx)
	require.NoError(t, err)

	var msg protocol.ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	require.Equal(t, "STATE", msg.Type)
	require.Equal(t, "white", msg.Turn)
	require.Equal(t, "waitingForRoll", msg.Status)
	require.Len(t, msg.Board, 24)
	require.Equal(t, int8(15), msg.Board[23], "15 белых на пункте 24")
	require.Equal(t, int8(-15), msg.Board[11], "15 чёрных на пункте 12")
}
