package rest_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/auth"
	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/TimurAUG/backgammon-game/internal/transport/rest"
	"github.com/stretchr/testify/require"
)

// Card #22 — create/join seat user ids + /api/me after finish.
// SPEC [AMENDED 2026-11-13 — card #24, source card #21].

type discardConn struct{}

func (discardConn) Send(protocol.ServerMessage) error { return nil }

func TestCreateJoin_WithSession_RecordsSeatUserIDs(t *testing.T) {
	store := auth.NewMemoryStore()
	mgr := game.NewManager()
	mgr.SetStatsRecorder(store)
	mux := http.NewServeMux()
	rest.NewHandler(mgr, store).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	whiteCookie := registerCookie(t, srv, "WhiteP")
	blackCookie := registerCookie(t, srv, "BlackP")

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{whiteCookie})
	defer createResp.Body.Close()
	require.Equal(t, http.StatusOK, createResp.StatusCode)
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{blackCookie})
	defer joinResp.Body.Close()
	require.Equal(t, http.StatusOK, joinResp.StatusCode)

	_, g, err := mgr.JoinGame(created.GameID, created.Token, discardConn{})
	require.NoError(t, err)
	uids := g.UserIDs()
	require.NotEmpty(t, uids[domain.White])
	require.NotEmpty(t, uids[domain.Black])
	require.NotEqual(t, uids[domain.White], uids[domain.Black])
}

func TestMe_AfterResignWinAndLoss(t *testing.T) {
	store := auth.NewMemoryStore()
	mgr := game.NewManager()
	mgr.SetStatsRecorder(store)
	mux := http.NewServeMux()
	rest.NewHandler(mgr, store).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	whiteCookie := registerCookie(t, srv, "WStat")
	blackCookie := registerCookie(t, srv, "BStat")

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{whiteCookie})
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()

	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{blackCookie})
	joinResp.Body.Close()

	_, g, err := mgr.JoinGame(created.GameID, created.Token, discardConn{})
	require.NoError(t, err)
	g.Resign(domain.White) // black wins — white loss

	meW := getMe(t, srv, []*http.Cookie{whiteCookie})
	defer meW.Body.Close()
	require.Equal(t, http.StatusOK, meW.StatusCode)
	var whiteMe meBody
	require.NoError(t, json.NewDecoder(meW.Body).Decode(&whiteMe))
	require.Equal(t, 1, whiteMe.Played)
	require.Equal(t, 0, whiteMe.Wins)
	require.Equal(t, 1, whiteMe.Losses)
	require.Equal(t, 0, whiteMe.WinPercent)
	require.Equal(t, whiteMe.Played, whiteMe.Wins+whiteMe.Losses)

	meB := getMe(t, srv, []*http.Cookie{blackCookie})
	defer meB.Body.Close()
	var blackMe meBody
	require.NoError(t, json.NewDecoder(meB.Body).Decode(&blackMe))
	require.Equal(t, 1, blackMe.Played)
	require.Equal(t, 1, blackMe.Wins)
	require.Equal(t, 0, blackMe.Losses)
	require.Equal(t, 100, blackMe.WinPercent)
}

func TestMe_WinPercent_AfterSeededStats(t *testing.T) {
	store := auth.NewMemoryStore()
	mgr := game.NewManager()
	mgr.SetStatsRecorder(store)
	mux := http.NewServeMux()
	rest.NewHandler(mgr, store).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cookie := registerCookie(t, srv, "Pct")
	u, err := store.Authenticate(t.Context(), "Pct", "password1")
	require.NoError(t, err)
	// 27 wins then 15 losses → 42 played → 64%.
	for i := 0; i < 27; i++ {
		require.NoError(t, store.ApplyFinishedGame(t.Context(), u.ID, "other", domain.White))
	}
	for i := 0; i < 15; i++ {
		require.NoError(t, store.ApplyFinishedGame(t.Context(), u.ID, "other", domain.Black))
	}

	resp := getMe(t, srv, []*http.Cookie{cookie})
	defer resp.Body.Close()
	var body meBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, 42, body.Played)
	require.Equal(t, 27, body.Wins)
	require.Equal(t, 15, body.Losses)
	require.Equal(t, 64, body.WinPercent)
	require.Equal(t, body.Played, body.Wins+body.Losses)
}

func TestMe_Logout_DoesNotClearRecord(t *testing.T) {
	store := auth.NewMemoryStore()
	mgr := game.NewManager()
	mgr.SetStatsRecorder(store)
	mux := http.NewServeMux()
	rest.NewHandler(mgr, store).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cookie := registerCookie(t, srv, "Keep")
	u, err := store.Authenticate(t.Context(), "Keep", "password1")
	require.NoError(t, err)
	require.NoError(t, store.ApplyFinishedGame(t.Context(), u.ID, "opp", domain.White))

	out := postJSON(t, srv.URL+"/api/logout", `{}`, []*http.Cookie{cookie})
	out.Body.Close()

	login := postJSON(t, srv.URL+"/api/login", `{"nick":"Keep","password":"password1"}`, nil)
	newCookie := sessionCookie(login)
	login.Body.Close()
	require.NotNil(t, newCookie)

	resp := getMe(t, srv, []*http.Cookie{newCookie})
	defer resp.Body.Close()
	var body meBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, 1, body.Played)
	require.Equal(t, 1, body.Wins)
}

func TestMe_UnknownCookie_NotZeroStatsBody(t *testing.T) {
	srv, _ := newAuthServer(t)
	resp := getMe(t, srv, []*http.Cookie{{Name: auth.CookieName, Value: "forged-session-id"}})
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"played":0`)
}
