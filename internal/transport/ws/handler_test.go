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

	msg := readMessage(t, ctx, conn)
	require.Equal(t, "STATE", msg.Type)
	require.Equal(t, "white", msg.Turn)
	require.Equal(t, "waitingForRoll", msg.Status)
	require.Len(t, msg.Board, 24)
	require.Equal(t, int8(15), msg.Board[23], "15 белых на пункте 24")
	require.Equal(t, int8(-15), msg.Board[11], "15 чёрных на пункте 12")
}

// TestHandler_SecondJoinNotifiesFirst — интеграционный тест на два клиента
// в одной игре: второй JOIN → второй получает STATE; первый клиент,
// уже ожидавший в read-loop, получает OPPONENT_JOINED.
//
// TDD plan #34 (часть 1).
func TestHandler_SecondJoinNotifiesFirst(t *testing.T) {
	mgr := game.NewManager()
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	state1 := readMessage(t, ctx, conn1)
	require.Equal(t, "STATE", state1.Type)

	conn2 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")
	state2 := readMessage(t, ctx, conn2)
	require.Equal(t, "STATE", state2.Type)

	opp := readMessage(t, ctx, conn1)
	require.Equal(t, "OPPONENT_JOINED", opp.Type)
}

func dialAndJoin(t *testing.T, ctx context.Context, wsURL, gameID string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	raw, err := json.Marshal(protocol.ClientMessage{Type: "JOIN", GameID: gameID})
	require.NoError(t, err)
	require.NoError(t, conn.Write(ctx, websocket.MessageText, raw))
	return conn
}

func readMessage(t *testing.T, ctx context.Context, conn *websocket.Conn) protocol.ServerMessage {
	t.Helper()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	var msg protocol.ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	return msg
}
