package ws_test

import (
	"bytes"
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

// TestHandler_RollForFirst — оба клиента шлют ROLL_FOR_FIRST. Сервер,
// получив сигнал от обоих, делает броски через rng в порядке (white, black).
// При rng [4, 2] → white=5, black=3 → white побеждает, dice=(5, 3).
// Оба клиента получают STATE с turn=white, status=waitingForMove и Dice.
//
// TDD plan #34 (часть 2).
func TestHandler_RollForFirst(t *testing.T) {
	rng := bytes.NewReader([]byte{4, 2})
	mgr := game.NewManagerWithRand(rng)
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	_ = readMessage(t, ctx, conn1) // STATE при JOIN

	conn2 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")
	_ = readMessage(t, ctx, conn2) // STATE при JOIN
	_ = readMessage(t, ctx, conn1) // OPPONENT_JOINED

	rfr, err := json.Marshal(protocol.ClientMessage{Type: "ROLL_FOR_FIRST"})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, rfr))
	require.NoError(t, conn2.Write(ctx, websocket.MessageText, rfr))

	state1 := readMessage(t, ctx, conn1)
	state2 := readMessage(t, ctx, conn2)

	for _, s := range []protocol.ServerMessage{state1, state2} {
		require.Equal(t, "STATE", s.Type)
		require.Equal(t, "white", s.Turn)
		require.Equal(t, "waitingForMove", s.Status)
		require.NotNil(t, s.Dice)
		require.Equal(t, uint8(5), s.Dice.A)
		require.Equal(t, uint8(3), s.Dice.B)
		require.False(t, s.Dice.IsDouble)
		require.Equal(t, []uint8{5, 3}, s.Dice.Remaining)
	}
}

// TestHandler_LegalMovesAfterRollForFirst — после определения первого хода
// победитель (white при rng [4,2]) получает кроме STATE ещё и LEGAL_MOVES
// со списком одиночных ходов с учётом текущих пипсов.
//
// initial board, dice (5, 3): белый может только с 24 — 24→19 пипсом 5,
// 24→21 пипсом 3.
//
// TDD plan #34 (часть 3).
func TestHandler_LegalMovesAfterRollForFirst(t *testing.T) {
	rng := bytes.NewReader([]byte{4, 2})
	mgr := game.NewManagerWithRand(rng)
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	_ = readMessage(t, ctx, conn1) // STATE при JOIN

	conn2 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")
	_ = readMessage(t, ctx, conn2) // STATE при JOIN
	_ = readMessage(t, ctx, conn1) // OPPONENT_JOINED

	rfr, err := json.Marshal(protocol.ClientMessage{Type: "ROLL_FOR_FIRST"})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, rfr))
	require.NoError(t, conn2.Write(ctx, websocket.MessageText, rfr))

	_ = readMessage(t, ctx, conn1) // STATE после определения первого
	_ = readMessage(t, ctx, conn2) // STATE после определения первого

	legalMoves := readMessage(t, ctx, conn1)
	require.Equal(t, "LEGAL_MOVES", legalMoves.Type)
	require.ElementsMatch(t, []protocol.MovePayload{
		{From: 24, To: 19, Pip: 5},
		{From: 24, To: 21, Pip: 3},
	}, legalMoves.Moves)
}
