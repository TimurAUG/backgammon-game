// Package ws предоставляет HTTP-handler, поднимающий WebSocket-канал
// для игровой партии.
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/coder/websocket"
)

// Handler — http.Handler, апгрейдящий запрос до WS и обрабатывающий
// сообщения клиента.
//
// Поддерживает:
//   - JOIN: регистрация в менеджере, ответ STATE, нотификация соперника
//     OPPONENT_JOINED, переход в read-loop.
//   - ROLL_FOR_FIRST: сигнал готовности на определение первого хода.
//
// ROLL / MOVE / END_TURN / RESIGN — в следующих циклах #34+.
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

	if err := pc.Send(game.StateMessage(g.State)); err != nil {
		return
	}
	if opp := g.Opponent(color); opp != nil {
		_ = opp.Send(protocol.ServerMessage{Type: "OPPONENT_JOINED"})
	}

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var in protocol.ClientMessage
		if err := json.Unmarshal(data, &in); err != nil {
			_ = pc.Send(protocol.ServerMessage{Type: "ERROR", Code: "INVALID_STATE", Message: "bad json"})
			continue
		}
		switch in.Type {
		case "ROLL_FOR_FIRST":
			_ = g.RollForFirst(color)
		case "MOVE":
			if err := g.HandleMove(color, in.From, in.To); err != nil {
				_ = pc.Send(protocol.ServerMessage{
					Type: "ERROR", Code: "INVALID_MOVE", Message: err.Error(),
				})
			}
		case "END_TURN":
			if err := g.EndTurn(color); err != nil {
				code := "INVALID_STATE"
				switch {
				case errors.Is(err, game.ErrMustUsePip):
					code = "MUST_USE_PIP"
				case errors.Is(err, game.ErrNotYourTurn):
					code = "NOT_YOUR_TURN"
				case errors.Is(err, game.ErrRuleOfSix):
					code = "RULE_OF_SIX"
				}
				_ = pc.Send(protocol.ServerMessage{
					Type: "ERROR", Code: code, Message: err.Error(),
				})
			}
		default:
			_ = pc.Send(protocol.ServerMessage{
				Type: "ERROR", Code: "INVALID_STATE",
				Message: "unsupported message: " + in.Type,
			})
		}
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
