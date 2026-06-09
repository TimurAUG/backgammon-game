// Package ws предоставляет HTTP-handler, поднимающий WebSocket-канал
// для игровой партии.
package ws

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/coder/websocket"
)

// Handler — http.Handler, апгрейдящий запрос до WS и обрабатывающий
// сообщения клиента.
//
// На данный момент поддерживает только JOIN — приём подключения, регистрацию
// в менеджере и ответ STATE. Расширение до полного потока ROLL/MOVE/END_TURN
// — в последующих циклах #34+.
type Handler struct {
	mgr *game.Manager
}

// NewHandler собирает handler с заданным менеджером игр.
func NewHandler(mgr *game.Manager) *Handler {
	return &Handler{mgr: mgr}
}

// ServeHTTP принимает WS-соединение, читает первое сообщение и, если это
// JOIN, регистрирует клиента в игре и отправляет STATE.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusInternalError, "internal error")

	ctx := r.Context()
	_, data, err := conn.Read(ctx)
	if err != nil {
		return
	}

	var msg protocol.ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		writeMessage(ctx, conn, protocol.ServerMessage{
			Type: "ERROR", Code: "INVALID_STATE", Message: "bad json",
		})
		return
	}
	if msg.Type != "JOIN" {
		writeMessage(ctx, conn, protocol.ServerMessage{
			Type: "ERROR", Code: "INVALID_STATE", Message: "expected JOIN",
		})
		return
	}

	g := h.mgr.JoinGame(msg.GameID)
	writeMessage(ctx, conn, stateMessage(g.State))
	conn.Close(websocket.StatusNormalClosure, "")
}

func writeMessage(ctx context.Context, conn *websocket.Conn, msg protocol.ServerMessage) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}
	_ = conn.Write(ctx, websocket.MessageText, raw)
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
