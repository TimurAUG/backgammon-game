// Package ws предоставляет HTTP-handler, поднимающий WebSocket-канал
// для игровой партии.
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/coder/websocket"
)

// Handler — http.Handler, апгрейдящий запрос до WS и обрабатывающий
// сообщения клиента.
//
// На данный момент поддерживает:
//   - JOIN: регистрация в менеджере, ответ STATE, нотификация соперника
//     OPPONENT_JOINED, переход в read-loop.
//
// ROLL_FOR_FIRST / ROLL / MOVE / END_TURN / RESIGN — в следующих циклах #34+.
type Handler struct {
	mgr *game.Manager
}

// NewHandler собирает handler с заданным менеджером игр.
func NewHandler(mgr *game.Manager) *Handler {
	return &Handler{mgr: mgr}
}

// ServeHTTP принимает WS-соединение, читает первое сообщение JOIN, регистрирует
// клиента в игре и держит соединение в read-loop до закрытия.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusInternalError, "internal error")

	ctx := r.Context()
	pc := &playerConn{conn: conn, ctx: ctx}

	_, data, err := conn.Read(ctx)
	if err != nil {
		return
	}

	var msg protocol.ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		_ = pc.Send(protocol.ServerMessage{Type: "ERROR", Code: "INVALID_STATE", Message: "bad json"})
		return
	}
	if msg.Type != "JOIN" {
		_ = pc.Send(protocol.ServerMessage{Type: "ERROR", Code: "INVALID_STATE", Message: "expected JOIN"})
		return
	}

	color, g, err := h.mgr.JoinGame(msg.GameID, pc)
	if err != nil {
		code := "INVALID_STATE"
		if errors.Is(err, game.ErrRoomFull) {
			code = "ROOM_FULL"
		}
		_ = pc.Send(protocol.ServerMessage{Type: "ERROR", Code: code, Message: err.Error()})
		return
	}
	defer g.Detach(color)

	if err := pc.Send(stateMessage(g.State)); err != nil {
		return
	}
	if opp := g.Opponent(color); opp != nil {
		_ = opp.Send(protocol.ServerMessage{Type: "OPPONENT_JOINED"})
	}

	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			return
		}
		// Будущие циклы #34+: парсинг ROLL/MOVE/END_TURN и т.п.
	}
}

// playerConn — реализация game.Conn поверх *websocket.Conn. Сериализация
// записи через mu — иначе пересекающиеся Send из разных горутин повредят
// фрейм WS.
type playerConn struct {
	conn *websocket.Conn
	ctx  context.Context
	mu   sync.Mutex
}

// Send сериализует msg в JSON и отправляет одним текстовым WS-фреймом.
func (p *playerConn) Send(msg protocol.ServerMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.conn.Write(p.ctx, websocket.MessageText, raw)
}

func stateMessage(s domain.GameState) protocol.ServerMessage {
	board := make([]int8, len(s.Board))
	for i, v := range s.Board {
		board[i] = v
	}
	return protocol.ServerMessage{
		Type:   "STATE",
		Board:  board,
		Turn:   colorString(s.Turn),
		Status: "waitingForRoll",
	}
}

func colorString(c domain.Color) string {
	if c == domain.Black {
		return "black"
	}
	return "white"
}
