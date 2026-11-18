package rest_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/auth"
	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
	"github.com/TimurAUG/backgammon-game/internal/transport/rest"
	"github.com/stretchr/testify/require"
)

// Card #32 — GET /api/games/active + create/join conflict guards.
// SPEC.md §2 [NEW/AMENDED 2026-11-18 — card #34].

var activeNickSeq atomic.Int32

type restGameConn struct{}

func (restGameConn) Send(_ protocol.ServerMessage) error { return nil }

func newActiveServer(t *testing.T) (*httptest.Server, *game.Manager, *auth.MemoryStore) {
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

func registerActive(t *testing.T, srv *httptest.Server, nick string) *http.Cookie {
	t.Helper()
	resp := postJSON(t, srv.URL+"/api/register", `{"nick":"`+nick+`","password":"password1"}`, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	c := sessionCookie(resp)
	require.NotNil(t, c)
	return c
}

func uniqueNick(prefix string) string {
	return prefix + strconv.Itoa(int(activeNickSeq.Add(1)))
}

func getActive(t *testing.T, srv *httptest.Server, cookies []*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/games/active", nil)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func createGameWithCookie(t *testing.T, srv *httptest.Server, cookie *http.Cookie) credsBody {
	t.Helper()
	resp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{cookie})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.NotEmpty(t, body.GameID)
	require.NotEmpty(t, body.Token)
	return body
}

func TestHandler_ActiveGame_ReturnsExistingSeatCredentials(t *testing.T) {
	// AC2: resume returns existing {gameId,token}; does not mint a second token.
	srv, _, _ := newActiveServer(t)
	cookie := registerActive(t, srv, uniqueNick("alice"))
	created := createGameWithCookie(t, srv, cookie)

	resp := getActive(t, srv, []*http.Cookie{cookie})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, created.GameID, body.GameID)
	require.Equal(t, created.Token, body.Token)
}

func TestHandler_ActiveGame_NoGame_Returns404(t *testing.T) {
	// AC3: no unfinished game → 404; no invented credentials.
	srv, _, _ := newActiveServer(t)
	cookie := registerActive(t, srv, uniqueNick("lonely"))

	resp := getActive(t, srv, []*http.Cookie{cookie})
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "gameId")
	require.NotContains(t, string(raw), "token")
}

func TestHandler_ActiveGame_NoSession_Returns401(t *testing.T) {
	// Edge: unauthenticated → 401; no credentials minted.
	srv, _, _ := newActiveServer(t)

	resp := getActive(t, srv, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "gameId")
	require.NotContains(t, string(raw), "token")
}

func TestHandler_ActiveGame_WaitingForOpponent_ReturnsWhiteCredentials(t *testing.T) {
	// AC7: host waiting for opponent reclaimable.
	srv, _, _ := newActiveServer(t)
	cookie := registerActive(t, srv, uniqueNick("host"))
	created := createGameWithCookie(t, srv, cookie)

	resp := getActive(t, srv, []*http.Cookie{cookie})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, created.GameID, body.GameID)
	require.Equal(t, created.Token, body.Token)
}

func TestHandler_ActiveGame_AfterResign_Returns404(t *testing.T) {
	// AC4: finished (resign) is not active.
	srv, mgr, _ := newActiveServer(t)
	alice := registerActive(t, srv, uniqueNick("finA"))
	bob := registerActive(t, srv, uniqueNick("finB"))
	created := createGameWithCookie(t, srv, alice)
	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{bob})
	defer joinResp.Body.Close()
	require.Equal(t, http.StatusOK, joinResp.StatusCode)

	_, g, err := mgr.JoinGame(created.GameID, created.Token, restGameConn{})
	require.NoError(t, err)
	g.Resign(domain.White)

	resp := getActive(t, srv, []*http.Cookie{alice})
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandler_CreateGame_WhenActiveExists_Returns409(t *testing.T) {
	// AC5: POST /api/games while active → 409 conflict; no new seat.
	srv, _, _ := newActiveServer(t)
	cookie := registerActive(t, srv, uniqueNick("busy"))
	first := createGameWithCookie(t, srv, cookie)

	resp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{cookie})
	defer resp.Body.Close()
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, strings.TrimSpace(string(raw)), "conflict")

	active := getActive(t, srv, []*http.Cookie{cookie})
	defer active.Body.Close()
	var body credsBody
	require.NoError(t, json.NewDecoder(active.Body).Decode(&body))
	require.Equal(t, first.GameID, body.GameID)
	require.Equal(t, first.Token, body.Token)
}

func TestHandler_Join_WhenOtherActiveExists_Returns409(t *testing.T) {
	// AC6: join different game while active → 409; existing seat stays.
	srv, _, _ := newActiveServer(t)
	alice := registerActive(t, srv, uniqueNick("alice"))
	bob := registerActive(t, srv, uniqueNick("bob"))
	own := createGameWithCookie(t, srv, alice)
	other := createGameWithCookie(t, srv, bob)

	resp := postJSON(t, srv.URL+"/api/games/"+other.GameID+"/join", `{}`, []*http.Cookie{alice})
	defer resp.Body.Close()
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, strings.TrimSpace(string(raw)), "conflict")

	active := getActive(t, srv, []*http.Cookie{alice})
	defer active.Body.Close()
	var body credsBody
	require.NoError(t, json.NewDecoder(active.Body).Decode(&body))
	require.Equal(t, own.GameID, body.GameID)
	require.Equal(t, own.Token, body.Token)
}

func TestHandler_Join_AlreadySeatedSameGame_ReturnsExistingToken(t *testing.T) {
	// Edge: join same unfinished game → existing {gameId,token}.
	srv, _, _ := newActiveServer(t)
	cookie := registerActive(t, srv, uniqueNick("host"))
	created := createGameWithCookie(t, srv, cookie)

	resp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{cookie})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, created.GameID, body.GameID)
	require.Equal(t, created.Token, body.Token)
}

func TestHandler_Logout_ThenActiveGame_StillReturnsCredentials(t *testing.T) {
	// Edge: logout ends session only; seat remains; next login can reclaim.
	srv, _, _ := newActiveServer(t)
	nick := uniqueNick("logout")
	cookie := registerActive(t, srv, nick)
	created := createGameWithCookie(t, srv, cookie)

	logout := postJSON(t, srv.URL+"/api/logout", `{}`, []*http.Cookie{cookie})
	logout.Body.Close()
	require.Equal(t, http.StatusOK, logout.StatusCode)

	login := postJSON(t, srv.URL+"/api/login", `{"nick":"`+nick+`","password":"password1"}`, nil)
	defer login.Body.Close()
	require.Equal(t, http.StatusOK, login.StatusCode)
	newCookie := sessionCookie(login)
	require.NotNil(t, newCookie)

	resp := getActive(t, srv, []*http.Cookie{newCookie})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, created.GameID, body.GameID)
	require.Equal(t, created.Token, body.Token)
}

func TestHandler_CreateGame_ConcurrentSameUser_AtMostOne200(t *testing.T) {
	// Edge: concurrent POST /api/games → at most one 200; others 409.
	srv, _, _ := newActiveServer(t)
	cookie := registerActive(t, srv, uniqueNick("race"))

	const n = 12
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		okN       int
		conflictN int
	)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			resp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{cookie})
			defer resp.Body.Close()
			mu.Lock()
			defer mu.Unlock()
			switch resp.StatusCode {
			case http.StatusOK:
				okN++
			case http.StatusConflict:
				conflictN++
			default:
				t.Errorf("unexpected status %d", resp.StatusCode)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, okN)
	require.Equal(t, n-1, conflictN)
}

func TestHandler_ActiveGame_TwoSessions_SameCredentials(t *testing.T) {
	// Edge: two concurrent sessions reclaim the same seat credentials.
	srv, _, _ := newActiveServer(t)
	nick := uniqueNick("dual")
	cookie1 := registerActive(t, srv, nick)
	created := createGameWithCookie(t, srv, cookie1)

	login := postJSON(t, srv.URL+"/api/login", `{"nick":"`+nick+`","password":"password1"}`, nil)
	defer login.Body.Close()
	require.Equal(t, http.StatusOK, login.StatusCode)
	cookie2 := sessionCookie(login)
	require.NotNil(t, cookie2)

	r1 := getActive(t, srv, []*http.Cookie{cookie1})
	defer r1.Body.Close()
	r2 := getActive(t, srv, []*http.Cookie{cookie2})
	defer r2.Body.Close()
	require.Equal(t, http.StatusOK, r1.StatusCode)
	require.Equal(t, http.StatusOK, r2.StatusCode)

	var b1, b2 credsBody
	require.NoError(t, json.NewDecoder(r1.Body).Decode(&b1))
	require.NoError(t, json.NewDecoder(r2.Body).Decode(&b2))
	require.Equal(t, created, b1)
	require.Equal(t, created, b2)
}

func TestHandler_ActiveGame_OtherUser_CannotStealSeat(t *testing.T) {
	// Edge: another user does not receive seated player's token via resume.
	srv, _, _ := newActiveServer(t)
	alice := registerActive(t, srv, uniqueNick("own"))
	intruder := registerActive(t, srv, uniqueNick("intr"))
	created := createGameWithCookie(t, srv, alice)

	resp := getActive(t, srv, []*http.Cookie{intruder})
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	aliceResp := getActive(t, srv, []*http.Cookie{alice})
	defer aliceResp.Body.Close()
	var body credsBody
	require.NoError(t, json.NewDecoder(aliceResp.Body).Decode(&body))
	require.Equal(t, created, body)
}
