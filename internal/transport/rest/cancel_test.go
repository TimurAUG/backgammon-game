package rest_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/auth"
	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/transport/rest"
	"github.com/stretchr/testify/require"
)

// Card #29 / SPEC.md §2 [NEW 2026-11-18 — card #31]: DELETE /api/games/{id}.

func deleteJSON(t *testing.T, url string, cookies []*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func newCancelServer(t *testing.T) (*httptest.Server, *game.Manager, *auth.MemoryStore) {
	t.Helper()
	store := auth.NewMemoryStore()
	mgr := game.NewManager()
	mgr.SetStatsRecorder(store)
	mux := http.NewServeMux()
	rest.NewHandler(mgr, store).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, mgr, store
}

func TestHandler_Cancel_Creator_Returns204_AndJoin404(t *testing.T) {
	srv, _, _ := newCancelServer(t)
	cookie := registerCookie(t, srv, "HostA")

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{cookie})
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()

	del := deleteJSON(t, srv.URL+"/api/games/"+created.GameID, []*http.Cookie{cookie})
	defer del.Body.Close()
	require.Equal(t, http.StatusNoContent, del.StatusCode)
	body, err := io.ReadAll(del.Body)
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(string(body)))

	joiner := registerCookie(t, srv, "JoinA")
	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{joiner})
	defer joinResp.Body.Close()
	require.Equal(t, http.StatusNotFound, joinResp.StatusCode)
}

func TestHandler_Cancel_AfterJoin_Still204_AndJoin404(t *testing.T) {
	srv, _, _ := newCancelServer(t)
	host := registerCookie(t, srv, "HostB")
	opp := registerCookie(t, srv, "OppB")

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{host})
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()

	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{opp})
	joinResp.Body.Close()
	require.Equal(t, http.StatusOK, joinResp.StatusCode)

	del := deleteJSON(t, srv.URL+"/api/games/"+created.GameID, []*http.Cookie{host})
	defer del.Body.Close()
	require.Equal(t, http.StatusNoContent, del.StatusCode)

	again := registerCookie(t, srv, "LateB")
	late := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{again})
	defer late.Body.Close()
	require.Equal(t, http.StatusNotFound, late.StatusCode)
}

func TestHandler_Cancel_DoesNotIncrementMeStats(t *testing.T) {
	srv, _, _ := newCancelServer(t)
	host := registerCookie(t, srv, "HostC")
	opp := registerCookie(t, srv, "OppC")

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{host})
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()
	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{opp})
	joinResp.Body.Close()

	del := deleteJSON(t, srv.URL+"/api/games/"+created.GameID, []*http.Cookie{host})
	del.Body.Close()
	require.Equal(t, http.StatusNoContent, del.StatusCode)

	for _, c := range []*http.Cookie{host, opp} {
		me := getMe(t, srv, []*http.Cookie{c})
		var body meBody
		require.NoError(t, json.NewDecoder(me.Body).Decode(&body))
		me.Body.Close()
		require.Equal(t, 0, body.Played)
		require.Equal(t, 0, body.Wins)
		require.Equal(t, 0, body.Losses)
	}
}

func TestHandler_Cancel_NoSession_Returns401(t *testing.T) {
	srv, _, _ := newCancelServer(t)
	host := registerCookie(t, srv, "HostD")
	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{host})
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()

	del := deleteJSON(t, srv.URL+"/api/games/"+created.GameID, nil)
	defer del.Body.Close()
	require.Equal(t, http.StatusUnauthorized, del.StatusCode)

	// Game still joinable — not deleted.
	opp := registerCookie(t, srv, "OppD")
	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{opp})
	defer joinResp.Body.Close()
	require.Equal(t, http.StatusOK, joinResp.StatusCode)
}

func TestHandler_Cancel_Joiner_Returns403_GameRemains(t *testing.T) {
	srv, _, _ := newCancelServer(t)
	host := registerCookie(t, srv, "HostE")
	opp := registerCookie(t, srv, "OppE")

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{host})
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()
	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{opp})
	joinResp.Body.Close()

	del := deleteJSON(t, srv.URL+"/api/games/"+created.GameID, []*http.Cookie{opp})
	defer del.Body.Close()
	require.Equal(t, http.StatusForbidden, del.StatusCode)

	// Still full (host + joiner) — third gets 409, proving game remains.
	third := registerCookie(t, srv, "ThirdE")
	late := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{third})
	defer late.Body.Close()
	require.Equal(t, http.StatusConflict, late.StatusCode)
}

func TestHandler_Cancel_UnknownID_Returns404(t *testing.T) {
	srv, _, _ := newCancelServer(t)
	cookie := registerCookie(t, srv, "HostF")
	del := deleteJSON(t, srv.URL+"/api/games/no-such-id", []*http.Cookie{cookie})
	defer del.Body.Close()
	require.Equal(t, http.StatusNotFound, del.StatusCode)
}

func TestHandler_Cancel_SecondCall_Returns404(t *testing.T) {
	srv, _, _ := newCancelServer(t)
	cookie := registerCookie(t, srv, "HostG")
	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{cookie})
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()

	del1 := deleteJSON(t, srv.URL+"/api/games/"+created.GameID, []*http.Cookie{cookie})
	del1.Body.Close()
	require.Equal(t, http.StatusNoContent, del1.StatusCode)

	del2 := deleteJSON(t, srv.URL+"/api/games/"+created.GameID, []*http.Cookie{cookie})
	defer del2.Body.Close()
	require.Equal(t, http.StatusNotFound, del2.StatusCode)
}

func TestHandler_Cancel_FinishedGame_Returns409(t *testing.T) {
	srv, mgr, _ := newCancelServer(t)
	host := registerCookie(t, srv, "HostH")
	opp := registerCookie(t, srv, "OppH")

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{host})
	var created credsBody
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	createResp.Body.Close()
	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{opp})
	var joined credsBody
	require.NoError(t, json.NewDecoder(joinResp.Body).Decode(&joined))
	joinResp.Body.Close()

	_, g, err := mgr.JoinGame(created.GameID, created.Token, discardConn{})
	require.NoError(t, err)
	_, _, err = mgr.JoinGame(created.GameID, joined.Token, discardConn{})
	require.NoError(t, err)
	g.Resign(domain.White)

	meBefore := getMe(t, srv, []*http.Cookie{host})
	var before meBody
	require.NoError(t, json.NewDecoder(meBefore.Body).Decode(&before))
	meBefore.Body.Close()
	require.Equal(t, 1, before.Played)

	del := deleteJSON(t, srv.URL+"/api/games/"+created.GameID, []*http.Cookie{host})
	defer del.Body.Close()
	require.Equal(t, http.StatusConflict, del.StatusCode)

	meAfter := getMe(t, srv, []*http.Cookie{host})
	var after meBody
	require.NoError(t, json.NewDecoder(meAfter.Body).Decode(&after))
	meAfter.Body.Close()
	require.Equal(t, before.Played, after.Played)
	require.Equal(t, before.Wins, after.Wins)
	require.Equal(t, before.Losses, after.Losses)
}
