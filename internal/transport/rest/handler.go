// Package rest предоставляет HTTP-хендлеры invite-флоу и auth REST:
// создание/вход в игру, register/login/me/logout. Токены игры (gameId/token)
// генерит сервер. Авторизация игры по WS остаётся прежней (seat token).
package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/TimurAUG/backgammon-game/internal/auth"
	"github.com/TimurAUG/backgammon-game/internal/game"
)

// meResponse is GET /api/me body (card #22 / SPEC #24).
type meResponse struct {
	Nick       string `json:"nick"`
	Played     int    `json:"played"`
	Wins       int    `json:"wins"`
	Losses     int    `json:"losses"`
	WinPercent int    `json:"win_percent"`
}

// Handler обслуживает REST-эндпойнты поверх game.Manager и auth.Store.
type Handler struct {
	mgr  *game.Manager
	auth auth.Store
	now  func() time.Time
}

// NewHandler собирает хендлер с менеджером игр и хранилищем пользователей.
// authStore may be nil only in tests that never hit auth routes; production
// always passes MemoryStore or PostgresStore.
func NewHandler(mgr *game.Manager, authStore auth.Store) *Handler {
	return &Handler{
		mgr:  mgr,
		auth: authStore,
		now:  time.Now,
	}
}

// Register вешает маршруты на mux (Go 1.22+ метод-паттерны). Используется и в
// проде (main.go), и в тестах — единый источник правды по путям.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/games", h.handleCreate)
	mux.HandleFunc("POST /api/games/{id}/join", h.handleJoin)
	mux.HandleFunc("POST /api/register", h.handleRegister)
	mux.HandleFunc("POST /api/login", h.handleLogin)
	mux.HandleFunc("GET /api/me", h.handleMe)
	mux.HandleFunc("POST /api/logout", h.handleLogout)
}

// credentialsResponse — тело ответа create/join: gameId + персональный token.
type credentialsResponse struct {
	GameID string `json:"gameId"`
	Token  string `json:"token"`
}

type authBody struct {
	Nick     string `json:"nick"`
	Password string `json:"password"`
}

type nickResponse struct {
	Nick string `json:"nick"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	nick, userID, ok := h.UserFromRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	gameID, token, err := h.mgr.CreateGame(nick, userID)
	if err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, credentialsResponse{GameID: gameID, Token: token})
}

func (h *Handler) handleJoin(w http.ResponseWriter, r *http.Request) {
	nick, userID, ok := h.UserFromRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	gameID := r.PathValue("id")
	token, err := h.mgr.JoinByID(gameID, nick, userID)
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

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeAuthBody(w, r)
	if !ok {
		return
	}
	u, err := h.auth.CreateUser(r.Context(), body.Nick, body.Password)
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	if err := h.issueSession(w, r, u.ID); err != nil {
		http.Error(w, "session failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, nickResponse{Nick: u.Nick})
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeAuthBody(w, r)
	if !ok {
		return
	}
	u, err := h.auth.Authenticate(r.Context(), body.Nick, body.Password)
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	if err := h.issueSession(w, r, u.ID); err != nil {
		http.Error(w, "session failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, nickResponse{Nick: u.Nick})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := h.userFromRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	played, wins, losses, err := h.auth.UserStats(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, meResponse{
		Nick:       u.Nick,
		Played:     played,
		Wins:       wins,
		Losses:     losses,
		WinPercent: auth.WinPercent(wins, played),
	})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if sid, ok := auth.SessionIDFromRequest(r); ok {
		_ = h.auth.DeleteSession(r.Context(), sid)
	}
	auth.ClearSessionCookie(w, r)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) issueSession(w http.ResponseWriter, r *http.Request, userID string) error {
	sess, err := h.auth.CreateSession(r.Context(), userID, auth.SessionExpiry(h.now()))
	if err != nil {
		return err
	}
	auth.SetSessionCookie(w, r, sess.ID)
	return nil
}

// UserFromRequest resolves the session cookie to a user. Exported for #3.
func (h *Handler) UserFromRequest(r *http.Request) (nick, userID string, ok bool) {
	u, ok := h.userFromRequest(r)
	if !ok {
		return "", "", false
	}
	return u.Nick, u.ID, true
}

func (h *Handler) userFromRequest(r *http.Request) (*auth.User, bool) {
	sid, ok := auth.SessionIDFromRequest(r)
	if !ok {
		return nil, false
	}
	u, err := h.auth.GetSessionUser(r.Context(), sid, h.now())
	if err != nil {
		return nil, false
	}
	return u, true
}

func decodeAuthBody(w http.ResponseWriter, r *http.Request) (authBody, bool) {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, 1<<20)
	dec := json.NewDecoder(limited)
	var body authBody
	if err := dec.Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return authBody{}, false
	}
	// Require both fields present (empty string after decode still fails validation).
	if body.Nick == "" || body.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return authBody{}, false
	}
	if err := auth.ValidateCredentials(auth.NormalizeNick(body.Nick), body.Password); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return authBody{}, false
	}
	return body, true
}

func writeAuthErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		http.Error(w, "bad request", http.StatusBadRequest)
	case errors.Is(err, auth.ErrNickTaken):
		http.Error(w, "nick taken", http.StatusConflict)
	case errors.Is(err, auth.ErrInvalidCredentials):
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
