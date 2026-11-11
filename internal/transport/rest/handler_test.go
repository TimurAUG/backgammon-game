package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/auth"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/transport/rest"
	"github.com/stretchr/testify/require"
)

// TDD plan #41 — REST invite: POST /api/games (создать), POST
// /api/games/{id}/join (войти по ссылке). Токены генерит сервер.
// Card #3 — create/join helpers attach a valid nardy_session cookie.

type credsBody struct {
	GameID string `json:"gameId"`
	Token  string `json:"token"`
}

var inviteNickSeq atomic.Int32

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	rest.NewHandler(game.NewManager(), auth.NewMemoryStore()).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func inviteSession(t *testing.T, srv *httptest.Server) *http.Cookie {
	t.Helper()
	nick := "invite" + strconv.Itoa(int(inviteNickSeq.Add(1)))
	resp := postJSON(t, srv.URL+"/api/register", `{"nick":"`+nick+`","password":"password1"}`, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	c := sessionCookie(resp)
	require.NotNil(t, c)
	return c
}

func createGame(t *testing.T, srv *httptest.Server) credsBody {
	t.Helper()
	cookie := inviteSession(t, srv)
	resp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{cookie})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

func join(t *testing.T, srv *httptest.Server, gameID string) *http.Response {
	t.Helper()
	cookie := inviteSession(t, srv)
	return postJSON(t, srv.URL+"/api/games/"+gameID+"/join", `{}`, []*http.Cookie{cookie})
}

func TestHandler_CreateGame_ReturnsGameIDAndToken(t *testing.T) {
	srv := newServer(t)

	body := createGame(t, srv)

	require.NotEmpty(t, body.GameID, "ответ содержит gameId")
	require.NotEmpty(t, body.Token, "ответ содержит token")
}

func TestHandler_Join_ExistingGame_ReturnsToken(t *testing.T) {
	srv := newServer(t)
	created := createGame(t, srv)

	resp := join(t, srv, created.GameID)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, created.GameID, body.GameID)
	require.NotEmpty(t, body.Token)
	require.NotEqual(t, created.Token, body.Token, "у второго игрока свой токен")
}

func TestHandler_Join_UnknownGame_Returns404(t *testing.T) {
	srv := newServer(t)

	resp := join(t, srv, "no-such-game")
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandler_Join_FullGame_Returns409(t *testing.T) {
	srv := newServer(t)
	created := createGame(t, srv)
	r1 := join(t, srv, created.GameID) // занимает Black
	r1.Body.Close()
	require.Equal(t, http.StatusOK, r1.StatusCode)

	r2 := join(t, srv, created.GameID) // третий — мест нет
	defer r2.Body.Close()

	require.Equal(t, http.StatusConflict, r2.StatusCode)
}

// проверка, что роутинг отбивает не-POST (sanity на метод-паттерн).
func TestHandler_CreateGame_RejectsGet(t *testing.T) {
	srv := newServer(t)

	resp, err := http.Get(srv.URL + "/api/games")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
