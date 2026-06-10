package ws_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/TimurAUG/backgammon-game/internal/transport/ws"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

// dialCounter гарантирует уникальный token при каждом вызове dialAndJoin —
// иначе два клиента в одной игре получали бы одинаковый token и второй
// случайно реконнектился бы вместо подключения как соперник.
var dialCounter atomic.Int32

// TestHandler_CrossOrigin_RejectedByDefault — по умолчанию проверка Origin
// строгая (coder/websocket): запрос с чужим Origin отклоняется. Self-host.
func TestHandler_CrossOrigin_RejectedByDefault(t *testing.T) {
	mgr := game.NewManager()
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {"http://evil.test"}},
	})

	require.Error(t, err, "чужой Origin должен отклоняться при строгой проверке")
}

// TestHandler_CrossOrigin_AllowedWhenConfigured — с OriginPatterns=["*"]
// (ALLOWED_ORIGINS=*) чужой Origin принимается. Нужно для self-host за
// туннелем/реверс-прокси, где Host ≠ Origin (Cloudflare Tunnel, Caddy и т.п.).
func TestHandler_CrossOrigin_AllowedWhenConfigured(t *testing.T) {
	mgr := game.NewManager()
	h := ws.NewHandler(mgr)
	h.OriginPatterns = []string{"*"}
	srv := httptest.NewServer(h)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {"http://evil.test"}},
	})

	require.NoError(t, err, "чужой Origin должен приниматься при ALLOWED_ORIGINS=*")
	conn.Close(websocket.StatusNormalClosure, "")
}

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

	conn := dialAndJoinWithToken(t, ctx, wsURL, "g1", "t1")
	defer conn.Close(websocket.StatusInternalError, "test cleanup")

	msg := readMessage(t, ctx, conn)
	require.Equal(t, "STATE", msg.Type)
	require.Equal(t, "white", msg.Turn)
	require.Equal(t, "waitingForRoll", msg.Status)
	require.Len(t, msg.Board, 24)
	require.Equal(t, int8(15), msg.Board[23], "15 белых на пункте 24")
	require.Equal(t, int8(-15), msg.Board[11], "15 чёрных на пункте 12")
}

// TestHandler_JoinTokenFallback — браузерный путь аутентификации: нативный
// WebSocket в браузере не умеет ставить Authorization-заголовок, поэтому
// сервер должен принимать upgrade без заголовка и брать токен из payload
// JOIN (как задокументировано в nardy-protocol).
//
// Подготовка к web#24 (FRONTEND_SPEC, Этап 8).
func TestHandler_JoinTokenFallback(t *testing.T) {
	mgr := game.NewManager()
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err, "upgrade без Authorization-заголовка должен проходить")
	defer conn.Close(websocket.StatusInternalError, "test cleanup")

	raw, err := json.Marshal(protocol.ClientMessage{Type: "JOIN", GameID: "g1", Token: "t1"})
	require.NoError(t, err)
	require.NoError(t, conn.Write(ctx, websocket.MessageText, raw))

	joined := readMessage(t, ctx, conn)
	require.Equal(t, "JOINED", joined.Type, "JOIN с token в payload должен регистрировать игрока")
	msg := readMessage(t, ctx, conn)
	require.Equal(t, "STATE", msg.Type, "после JOINED приходит снапшот STATE")
}

// TestHandler_UnauthorizedWithoutAnyToken — нет ни Authorization-заголовка,
// ни token в JOIN → ERROR UNAUTHORIZED по WS.
//
// Заменяет TestHandler_UnauthorizedWithoutHeader (#37): отказ 401 до upgrade
// оставлял браузерные клиенты вообще без пути аутентификации.
func TestHandler_UnauthorizedWithoutAnyToken(t *testing.T) {
	mgr := game.NewManager()
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer conn.Close(websocket.StatusInternalError, "test cleanup")

	raw, err := json.Marshal(protocol.ClientMessage{Type: "JOIN", GameID: "g1"})
	require.NoError(t, err)
	require.NoError(t, conn.Write(ctx, websocket.MessageText, raw))

	msg := readMessage(t, ctx, conn)
	require.Equal(t, "ERROR", msg.Type)
	require.Equal(t, "UNAUTHORIZED", msg.Code, "JOIN без токена должен отклоняться как UNAUTHORIZED")
}

// TestHandler_JoinSendsJoinedWithColor — после успешного JOIN сервер шлёт
// присоединившемуся JOINED с его цветом, до STATE: первый клиент — white,
// второй — black. STATE одинаков для обоих игроков, так что иначе клиенту
// неоткуда узнать, каким цветом он играет.
//
// Подготовка к web#24 (FRONTEND_SPEC, Этап 8).
func TestHandler_JoinSendsJoinedWithColor(t *testing.T) {
	mgr := game.NewManager()
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dialJoin := func(token string) *websocket.Conn {
		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		require.NoError(t, err)
		raw, err := json.Marshal(protocol.ClientMessage{Type: "JOIN", GameID: "g-joined", Token: token})
		require.NoError(t, err)
		require.NoError(t, conn.Write(ctx, websocket.MessageText, raw))
		return conn
	}
	// Анонимная структура вместо protocol.ServerMessage — фиксируем
	// JSON-контракт (имя поля color), а не Go-структуру.
	readTypeColor := func(conn *websocket.Conn) (string, string) {
		_, data, err := conn.Read(ctx)
		require.NoError(t, err)
		var parsed struct {
			Type  string `json:"type"`
			Color string `json:"color"`
		}
		require.NoError(t, json.Unmarshal(data, &parsed))
		return parsed.Type, parsed.Color
	}

	conn1 := dialJoin("t-first")
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	typ1, color1 := readTypeColor(conn1)
	require.Equal(t, "JOINED", typ1, "первое сообщение после JOIN — JOINED, до STATE")
	require.Equal(t, "white", color1, "первый подключившийся играет белыми")

	conn2 := dialJoin("t-second")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")
	typ2, color2 := readTypeColor(conn2)
	require.Equal(t, "JOINED", typ2, "второй клиент тоже получает JOINED до STATE")
	require.Equal(t, "black", color2, "второй подключившийся играет чёрными")
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

// dialAndJoin подключается с уникальным токеном в Authorization-header.
// Используется в тестах, где нужны два разных игрока.
func dialAndJoin(t *testing.T, ctx context.Context, wsURL, gameID string) *websocket.Conn {
	t.Helper()
	n := dialCounter.Add(1)
	return dialAndJoinWithToken(t, ctx, wsURL, gameID, "tok-"+strconv.Itoa(int(n))+"-"+gameID)
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
		require.Equal(t, []int{5, 3}, s.Dice.Remaining)
	}
}

// TestHandler_RollForFirst_SendsFirstRollValues — после определения первого
// хода сервер шлёт FIRST_ROLL с индивидуальными бросками обоих цветов (#2),
// чтобы клиент показал «кто сколько бросил». rng [4,2] → white=5, black=3.
func TestHandler_RollForFirst_SendsFirstRollValues(t *testing.T) {
	rng := bytes.NewReader([]byte{4, 2})
	mgr := game.NewManagerWithRand(rng)
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	_ = readMessage(t, ctx, conn1)
	conn2 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")
	_ = readMessage(t, ctx, conn2)
	_ = readMessage(t, ctx, conn1) // OPPONENT_JOINED

	rfr, err := json.Marshal(protocol.ClientMessage{Type: "ROLL_FOR_FIRST"})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, rfr))
	require.NoError(t, conn2.Write(ctx, websocket.MessageText, rfr))

	var fr *protocol.FirstRollPayload
	for i := 0; i < 5 && fr == nil; i++ {
		m := readMessage(t, ctx, conn1)
		if m.Type == "FIRST_ROLL" {
			fr = m.FirstRoll
		}
	}
	require.NotNil(t, fr, "должно прийти FIRST_ROLL")
	require.Equal(t, 5, fr.White)
	require.Equal(t, 3, fr.Black)
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

// TestHandler_Move_AppliesAndBroadcasts — после ROLL_FOR_FIRST (rng[4,2] →
// white wins, dice 5:3) белый шлёт MOVE {from:24, to:19}. Сервер вычисляет
// pip=5, применяет Apply, рассылает STATE обоим, и LEGAL_MOVES только белому.
//
// Ожидание: после MOVE Board имеет 14 на пункте 24 и 1 на пункте 19;
// Remaining=[3]; HeadConsumed[White]=1.
//
// LEGAL_MOVES пуст: с пункта 24 правило головы запрещает (HeadConsumed=1,
// исключение работает только для дублей 6:6/4:4/3:3, а у нас 5:3); с пункта 19
// шаг 19→16 пипсом 3, на 16 пусто — легально.
//
// Проверим, что в LEGAL_MOVES есть 19→16 пипсом 3 и НЕТ 24→21 пипсом 3.
//
// TDD plan #34 (часть 4 — MOVE).
func TestHandler_Move_AppliesAndBroadcasts(t *testing.T) {
	rng := bytes.NewReader([]byte{4, 2})
	mgr := game.NewManagerWithRand(rng)
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	_ = readMessage(t, ctx, conn1)

	conn2 := dialAndJoin(t, ctx, wsURL, "g1")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")
	_ = readMessage(t, ctx, conn2)
	_ = readMessage(t, ctx, conn1)

	rfr, err := json.Marshal(protocol.ClientMessage{Type: "ROLL_FOR_FIRST"})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, rfr))
	require.NoError(t, conn2.Write(ctx, websocket.MessageText, rfr))
	_ = readMessage(t, ctx, conn1) // STATE
	_ = readMessage(t, ctx, conn2) // STATE
	_ = readMessage(t, ctx, conn1) // LEGAL_MOVES
	_ = readMessage(t, ctx, conn1) // FIRST_ROLL
	_ = readMessage(t, ctx, conn2) // FIRST_ROLL

	mv, err := json.Marshal(protocol.ClientMessage{Type: "MOVE", From: 24, To: 19})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, mv))

	state1 := readMessage(t, ctx, conn1)
	state2 := readMessage(t, ctx, conn2)
	for _, s := range []protocol.ServerMessage{state1, state2} {
		require.Equal(t, "STATE", s.Type)
		require.Len(t, s.Board, 24)
		require.Equal(t, int8(14), s.Board[23], "после хода 24→19 на 24 остаётся 14")
		require.Equal(t, int8(1), s.Board[18], "на 19 одна белая")
		require.Equal(t, []int{3}, s.Dice.Remaining)
	}

	legal := readMessage(t, ctx, conn1)
	require.Equal(t, "LEGAL_MOVES", legal.Type)
	require.ElementsMatch(t, []protocol.MovePayload{
		{From: 19, To: 16, Pip: 3},
	}, legal.Moves)
}

// TestHandler_EndTurn_RejectsWithPipsLeft — END_TURN при незавершённом ходе
// (есть пипс 3 и легальный ход 19→16) отклоняется с ERROR{MUST_USE_PIP}.
// State не меняется.
//
// TDD plan #34 (часть 5a).
func TestHandler_EndTurn_RejectsWithPipsLeft(t *testing.T) {
	rng := bytes.NewReader([]byte{4, 2})
	mgr := game.NewManagerWithRand(rng)
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1, conn2 := playUntilFirstMove(t, ctx, wsURL)
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")

	et, err := json.Marshal(protocol.ClientMessage{Type: "END_TURN"})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, et))

	msg := readMessage(t, ctx, conn1)
	require.Equal(t, "ERROR", msg.Type)
	require.Equal(t, "MUST_USE_PIP", msg.Code)
}

// TestHandler_AutoEndTurnOnEmptyLegalMoves — после исчерпания всех
// легальных ходов сервер автоматически передаёт ход сопернику. Клиент
// не должен слать END_TURN явно.
//
// Сценарий: MOVE 24→19, MOVE 19→16. После второго MOVE LegalMoves пуст,
// поэтому вместо LEGAL_MOVES сервер шлёт второй STATE с turn=black,
// status=waitingForRoll.
//
// TDD plan #34 (часть 5b + 9 — auto-END_TURN).
func TestHandler_AutoEndTurnOnEmptyLegalMoves(t *testing.T) {
	rng := bytes.NewReader([]byte{4, 2})
	mgr := game.NewManagerWithRand(rng)
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1, conn2 := playUntilFirstMove(t, ctx, wsURL)
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")

	mv, err := json.Marshal(protocol.ClientMessage{Type: "MOVE", From: 19, To: 16})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, mv))
	_ = readMessage(t, ctx, conn1) // STATE после Apply
	_ = readMessage(t, ctx, conn2) // STATE после Apply

	// LegalMoves пуст → auto-END_TURN: ещё один STATE с переключённым Turn,
	// без LEGAL_MOVES между ними.
	state1 := readMessage(t, ctx, conn1)
	state2 := readMessage(t, ctx, conn2)
	for _, s := range []protocol.ServerMessage{state1, state2} {
		require.Equal(t, "STATE", s.Type)
		require.Equal(t, "black", s.Turn)
		require.Equal(t, "waitingForRoll", s.Status)
		require.Nil(t, s.Dice, "Dice должен быть сброшен после auto-END_TURN")
	}
}

// TestHandler_Roll_StartsBlackTurn — после END_TURN белого, чёрный шлёт ROLL.
// rng-байты [1, 3] → бросок (2, 4). Оба получают STATE с turn=black,
// status=waitingForMove, dice=(2,4). Чёрный также получает LEGAL_MOVES
// с двумя ходами с головы (правило головы разрешает первый ход с головы).
//
// TDD plan #34 (часть 7 — ROLL).
func TestHandler_Roll_StartsBlackTurn(t *testing.T) {
	// 4 байта: 2 на ROLL_FOR_FIRST (white=5,black=3), 2 на ROLL чёрного (2,4).
	rng := bytes.NewReader([]byte{4, 2, 1, 3})
	mgr := game.NewManagerWithRand(rng)
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1, conn2 := playUntilFirstMove(t, ctx, wsURL)
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")

	// Завершить ход белого: MOVE 19→16 пипсом 3 → auto-END_TURN
	// (LegalMoves пуст).
	mv, err := json.Marshal(protocol.ClientMessage{Type: "MOVE", From: 19, To: 16})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, mv))
	_ = readMessage(t, ctx, conn1) // STATE после Apply
	_ = readMessage(t, ctx, conn2) // STATE после Apply
	_ = readMessage(t, ctx, conn1) // STATE auto-END_TURN
	_ = readMessage(t, ctx, conn2) // STATE auto-END_TURN

	// Black: ROLL
	roll, err := json.Marshal(protocol.ClientMessage{Type: "ROLL"})
	require.NoError(t, err)
	require.NoError(t, conn2.Write(ctx, websocket.MessageText, roll))

	state1 := readMessage(t, ctx, conn1)
	state2 := readMessage(t, ctx, conn2)
	for _, s := range []protocol.ServerMessage{state1, state2} {
		require.Equal(t, "STATE", s.Type)
		require.Equal(t, "black", s.Turn)
		require.Equal(t, "waitingForMove", s.Status)
		require.NotNil(t, s.Dice)
		require.Equal(t, uint8(2), s.Dice.A)
		require.Equal(t, uint8(4), s.Dice.B)
		require.Equal(t, []int{2, 4}, s.Dice.Remaining)
	}

	legal := readMessage(t, ctx, conn2)
	require.Equal(t, "LEGAL_MOVES", legal.Type)
	require.ElementsMatch(t, []protocol.MovePayload{
		{From: 12, To: 10, Pip: 2},
		{From: 12, To: 8, Pip: 4},
	}, legal.Moves)
}

// TestHandler_Reconnect — клиент с тем же gameId и token подключается
// повторно (например, после обрыва соединения). Сервер должен вернуть его
// в тот же слот и прислать STATE.
//
// Сценарий:
//  1. conn1 JOIN(g1, t1) → STATE как white.
//  2. conn2 JOIN(g1, t1) → STATE как white (реконнект на тот же слот).
//  3. conn3 JOIN(g1, t2) → STATE как black; conn2 (новый white) получает
//     OPPONENT_JOINED.
//
// TDD plan #35.
func TestHandler_Reconnect(t *testing.T) {
	mgr := game.NewManager()
	srv := httptest.NewServer(ws.NewHandler(mgr))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn1 := dialAndJoinWithToken(t, ctx, wsURL, "g1", "t1")
	defer conn1.Close(websocket.StatusInternalError, "test cleanup")
	state1 := readMessage(t, ctx, conn1)
	require.Equal(t, "STATE", state1.Type)
	require.Equal(t, int8(15), state1.Board[23], "initial: 15 белых на 24")

	conn2 := dialAndJoinWithToken(t, ctx, wsURL, "g1", "t1")
	defer conn2.Close(websocket.StatusInternalError, "test cleanup")
	state2 := readMessage(t, ctx, conn2)
	require.Equal(t, "STATE", state2.Type)
	require.Equal(t, int8(15), state2.Board[23], "реконнект: тот же initial board")

	conn3 := dialAndJoinWithToken(t, ctx, wsURL, "g1", "t2")
	defer conn3.Close(websocket.StatusInternalError, "test cleanup")
	state3 := readMessage(t, ctx, conn3)
	require.Equal(t, "STATE", state3.Type)

	opp := readMessage(t, ctx, conn2)
	require.Equal(t, "OPPONENT_JOINED", opp.Type,
		"новый white-conn (после реконнекта) должен получить OPPONENT_JOINED при входе black")
}

// dialAndJoinWithToken подключается с явно заданным токеном в
// Authorization: Bearer-заголовке. JOIN-сообщение токен не несёт
// (по SPEC #37 заголовок — основной путь для не-браузерных клиентов).
//
// Съедает ответное JOINED, чтобы тесты дальше читали с STATE — так
// сценарии, которым цвет не важен, не зашумляются лишним readMessage.
func dialAndJoinWithToken(t *testing.T, ctx context.Context, wsURL, gameID, token string) *websocket.Conn {
	t.Helper()
	opts := &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": []string{"Bearer " + token},
		},
	}
	conn, _, err := websocket.Dial(ctx, wsURL, opts)
	require.NoError(t, err)
	raw, err := json.Marshal(protocol.ClientMessage{Type: "JOIN", GameID: gameID})
	require.NoError(t, err)
	require.NoError(t, conn.Write(ctx, websocket.MessageText, raw))

	joined := readMessage(t, ctx, conn)
	require.Equal(t, "JOINED", joined.Type, "после JOIN первым приходит JOINED")
	return conn
}

// playUntilFirstMove подключает двух клиентов, проходит JOIN/ROLL_FOR_FIRST
// и выполняет первый ход 24→19 пипсом 5. Возвращает соединения готовые к
// следующему сообщению белого. Соответствует rng[4,2].
func playUntilFirstMove(t *testing.T, ctx context.Context, wsURL string) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	conn1 := dialAndJoin(t, ctx, wsURL, "g1")
	_ = readMessage(t, ctx, conn1)
	conn2 := dialAndJoin(t, ctx, wsURL, "g1")
	_ = readMessage(t, ctx, conn2)
	_ = readMessage(t, ctx, conn1) // OPPONENT_JOINED

	rfr, err := json.Marshal(protocol.ClientMessage{Type: "ROLL_FOR_FIRST"})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, rfr))
	require.NoError(t, conn2.Write(ctx, websocket.MessageText, rfr))
	_ = readMessage(t, ctx, conn1) // STATE
	_ = readMessage(t, ctx, conn2) // STATE
	_ = readMessage(t, ctx, conn1) // LEGAL_MOVES
	_ = readMessage(t, ctx, conn1) // FIRST_ROLL
	_ = readMessage(t, ctx, conn2) // FIRST_ROLL

	mv, err := json.Marshal(protocol.ClientMessage{Type: "MOVE", From: 24, To: 19})
	require.NoError(t, err)
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, mv))
	_ = readMessage(t, ctx, conn1) // STATE
	_ = readMessage(t, ctx, conn2) // STATE
	_ = readMessage(t, ctx, conn1) // LEGAL_MOVES
	return conn1, conn2
}
