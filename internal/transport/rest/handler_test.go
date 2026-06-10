package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/transport/rest"
	"github.com/stretchr/testify/require"
)

// TDD plan #41 — REST invite: POST /api/games (создать), POST
// /api/games/{id}/join (войти по ссылке). Токены генерит сервер.

type credsBody struct {
	GameID string `json:"gameId"`
	Token  string `json:"token"`
}

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	rest.NewHandler(game.NewManager()).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func createGame(t *testing.T, srv *httptest.Server) credsBody {
	t.Helper()
	resp, err := http.Post(srv.URL+"/api/games", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

func join(t *testing.T, srv *httptest.Server, gameID string) *http.Response {
	t.Helper()
	resp, err := http.Post(srv.URL+"/api/games/"+gameID+"/join", "application/json", nil)
	require.NoError(t, err)
	return resp
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
