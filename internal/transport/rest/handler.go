// Package rest предоставляет HTTP-хендлеры invite-флоу: создание игры и
// вход по приглашению. Токены (gameId/token) генерит сервер — клиент их
// больше не придумывает. Авторизация игры по WS остаётся прежней (token).
package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/TimurAUG/backgammon-game/internal/game"
)

// Handler обслуживает REST-эндпойнты поверх game.Manager.
type Handler struct {
	mgr *game.Manager
}

// NewHandler собирает хендлер с заданным менеджером игр.
func NewHandler(mgr *game.Manager) *Handler {
	return &Handler{mgr: mgr}
}

// Register вешает маршруты на mux (Go 1.22+ метод-паттерны). Используется и в
// проде (main.go), и в тестах — единый источник правды по путям.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/games", h.handleCreate)
	mux.HandleFunc("POST /api/games/{id}/join", h.handleJoin)
}

// credentialsResponse — тело ответа create/join: gameId + персональный token.
type credentialsResponse struct {
	GameID string `json:"gameId"`
	Token  string `json:"token"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, _ *http.Request) {
	gameID, token, err := h.mgr.CreateGame()
	if err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, credentialsResponse{GameID: gameID, Token: token})
}

func (h *Handler) handleJoin(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")
	token, err := h.mgr.JoinByID(gameID)
	if err != nil {
		switch {
		case errors.Is(err, game.ErrGameNotFound):
			http.Error(w, "game not found", http.StatusNotFound)
		case errors.Is(err, game.ErrRoomFull):
			http.Error(w, "room full", http.StatusConflict)
		default:
			http.Error(w, "join failed", http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, credentialsResponse{GameID: gameID, Token: token})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
