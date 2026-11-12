package rest_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TimurAUG/backgammon-game/internal/auth"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/TimurAUG/backgammon-game/internal/transport/rest"
	"github.com/stretchr/testify/require"
)

// Card #2 — auth REST: register, login, me, logout.

type nickBody struct {
	Nick string `json:"nick"`
}

func newAuthServer(t *testing.T) (*httptest.Server, *auth.MemoryStore) {
	t.Helper()
	store := auth.NewMemoryStore()
	mux := http.NewServeMux()
	rest.NewHandler(game.NewManager(), store).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, store
}

func postJSON(t *testing.T, url, raw string, cookies []*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(raw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func getMe(t *testing.T, srv *httptest.Server, cookies []*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/me", nil)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func sessionCookie(resp *http.Response) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == auth.CookieName {
			return c
		}
	}
	return nil
}

// Card #2 AC1 — register sets cookie and returns nick only (no password).
func TestRegister_NewNick_SetsCookieAndStoresHash(t *testing.T) {
	srv, store := newAuthServer(t)

	resp := postJSON(t, srv.URL+"/api/register", `{"nick":"Ann","password":"password1"}`, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "password1")
	var body nickBody
	require.NoError(t, json.Unmarshal(raw, &body))
	require.Equal(t, "Ann", body.Nick)

	c := sessionCookie(resp)
	require.NotNil(t, c)
	require.True(t, c.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, c.SameSite)
	require.Equal(t, "/", c.Path)
	require.Equal(t, int(auth.SessionTTL/time.Second), c.MaxAge)

	u, err := store.Authenticate(t.Context(), "Ann", "password1")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(u.PasswordHash, "$argon2id$"))
	require.NotContains(t, u.PasswordHash, "password1")
}

// Card #2 AC2 — duplicate nick any casing → 409.
func TestRegister_DuplicateNickAnyCase_Returns409(t *testing.T) {
	srv, _ := newAuthServer(t)
	r1 := postJSON(t, srv.URL+"/api/register", `{"nick":"Ann","password":"password1"}`, nil)
	r1.Body.Close()
	require.Equal(t, http.StatusOK, r1.StatusCode)

	r2 := postJSON(t, srv.URL+"/api/register", `{"nick":"ann","password":"password2"}`, nil)
	defer r2.Body.Close()
	require.Equal(t, http.StatusConflict, r2.StatusCode)
}

// Card #2 AC3 — login sets a new session; concurrent sessions allowed.
func TestLogin_ValidPair_SetsNewSessionCookie(t *testing.T) {
	srv, _ := newAuthServer(t)
	r1 := postJSON(t, srv.URL+"/api/register", `{"nick":"Bob","password":"password1"}`, nil)
	c1 := sessionCookie(r1)
	r1.Body.Close()
	require.NotNil(t, c1)

	r2 := postJSON(t, srv.URL+"/api/login", `{"nick":"bob","password":"password1"}`, nil)
	defer r2.Body.Close()
	require.Equal(t, http.StatusOK, r2.StatusCode)
	loginRaw, err := io.ReadAll(r2.Body)
	require.NoError(t, err)
	require.NotContains(t, string(loginRaw), "password1")
	c2 := sessionCookie(r2)
	require.NotNil(t, c2)
	require.NotEqual(t, c1.Value, c2.Value)

	me1 := getMe(t, srv, []*http.Cookie{c1})
	defer me1.Body.Close()
	require.Equal(t, http.StatusOK, me1.StatusCode)
	me2 := getMe(t, srv, []*http.Cookie{c2})
	defer me2.Body.Close()
	require.Equal(t, http.StatusOK, me2.StatusCode)
}

// Card #2 AC4 — wrong password and unknown nick → same 401 body.
func TestLogin_WrongOrUnknown_Same401(t *testing.T) {
	srv, _ := newAuthServer(t)
	reg := postJSON(t, srv.URL+"/api/register", `{"nick":"Carla","password":"password1"}`, nil)
	reg.Body.Close()

	wrong := postJSON(t, srv.URL+"/api/login", `{"nick":"Carla","password":"bad-password"}`, nil)
	defer wrong.Body.Close()
	unknown := postJSON(t, srv.URL+"/api/login", `{"nick":"nobody","password":"password1"}`, nil)
	defer unknown.Body.Close()

	require.Equal(t, http.StatusUnauthorized, wrong.StatusCode)
	require.Equal(t, http.StatusUnauthorized, unknown.StatusCode)
	wb, _ := io.ReadAll(wrong.Body)
	ub, _ := io.ReadAll(unknown.Body)
	require.Equal(t, string(wb), string(ub))
}

// Card #2 AC5 — valid cookie → 200 with nick only.
func TestMe_ValidCookie_ReturnsNick(t *testing.T) {
	srv, _ := newAuthServer(t)
	reg := postJSON(t, srv.URL+"/api/register", `{"nick":"Dan","password":"password1"}`, nil)
	c := sessionCookie(reg)
	reg.Body.Close()

	resp := getMe(t, srv, []*http.Cookie{c})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "password")
	var body nickBody
	require.NoError(t, json.Unmarshal(raw, &body))
	require.Equal(t, "Dan", body.Nick)
}

// Card #2 AC6 — no cookie → 401.
func TestMe_NoCookie_Returns401(t *testing.T) {
	srv, _ := newAuthServer(t)
	resp := getMe(t, srv, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Card #2 AC7 / edge — logout ends one session; sibling remains.
func TestLogout_EndsOnlyCurrentSession(t *testing.T) {
	srv, _ := newAuthServer(t)
	r1 := postJSON(t, srv.URL+"/api/register", `{"nick":"Eve","password":"password1"}`, nil)
	c1 := sessionCookie(r1)
	r1.Body.Close()
	r2 := postJSON(t, srv.URL+"/api/login", `{"nick":"Eve","password":"password1"}`, nil)
	c2 := sessionCookie(r2)
	r2.Body.Close()

	out := postJSON(t, srv.URL+"/api/logout", `{}`, []*http.Cookie{c1})
	defer out.Body.Close()
	require.Equal(t, http.StatusOK, out.StatusCode)
	cleared := sessionCookie(out)
	require.NotNil(t, cleared)
	require.True(t, cleared.MaxAge < 0 || cleared.Value == "")

	me1 := getMe(t, srv, []*http.Cookie{c1})
	defer me1.Body.Close()
	require.Equal(t, http.StatusUnauthorized, me1.StatusCode)

	me2 := getMe(t, srv, []*http.Cookie{c2})
	defer me2.Body.Close()
	require.Equal(t, http.StatusOK, me2.StatusCode)
}

// Card #2 edge — logout with no/invalid cookie → 200 + clear Set-Cookie.
func TestLogout_NoCookie_Idempotent200(t *testing.T) {
	srv, _ := newAuthServer(t)
	resp := postJSON(t, srv.URL+"/api/logout", `{}`, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	c := sessionCookie(resp)
	require.NotNil(t, c)
	require.True(t, c.MaxAge < 0 || c.Value == "")
}

// Card #2 edge — logout with junk/invalid session cookie → 200 + clear Set-Cookie.
func TestLogout_InvalidCookie_Idempotent200(t *testing.T) {
	srv, _ := newAuthServer(t)
	junk := &http.Cookie{Name: auth.CookieName, Value: "not-a-real-session"}
	resp := postJSON(t, srv.URL+"/api/logout", `{}`, []*http.Cookie{junk})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	c := sessionCookie(resp)
	require.NotNil(t, c)
	require.True(t, c.MaxAge < 0 || c.Value == "")
}

// Card #2 edge (Code Review) — forged/unknown session cookie → /api/me 401.
func TestMe_UnknownCookie_Returns401(t *testing.T) {
	srv, _ := newAuthServer(t)
	resp := getMe(t, srv, []*http.Cookie{{Name: auth.CookieName, Value: "forged-session-id"}})
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Card #2 AC1 — Secure cookie when X-Forwarded-Proto is https; plain HTTP is not Secure.
func TestRegister_CookieSecure_FollowsHTTPS(t *testing.T) {
	store := auth.NewMemoryStore()
	mux := http.NewServeMux()
	rest.NewHandler(game.NewManager(), store).Register(mux)

	t.Run("X-Forwarded-Proto https → Secure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{"nick":"HttpsAnn","password":"password1"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		c := sessionCookie(rec.Result())
		require.NotNil(t, c)
		require.True(t, c.Secure)
	})

	t.Run("plain HTTP → not Secure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{"nick":"PlainAnn","password":"password1"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		c := sessionCookie(rec.Result())
		require.NotNil(t, c)
		require.False(t, c.Secure)
	})
}

// Card #2 edge (Code Review) — padded nick stores and returns trimmed nick.
func TestRegister_PaddedNick_StoresAndReturnsTrimmed(t *testing.T) {
	srv, store := newAuthServer(t)
	resp := postJSON(t, srv.URL+"/api/register", `{"nick":"  Ann  ","password":"password1"}`, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var body nickBody
	require.NoError(t, json.Unmarshal(raw, &body))
	require.Equal(t, "Ann", body.Nick)

	u, err := store.Authenticate(t.Context(), "Ann", "password1")
	require.NoError(t, err)
	require.Equal(t, "Ann", u.Nick)
}

// Card #2 edge — expired session → /api/me 401.
func TestMe_ExpiredSession_Returns401(t *testing.T) {
	store := auth.NewMemoryStore()
	mux := http.NewServeMux()
	h := rest.NewHandler(game.NewManager(), store)
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	u, err := store.CreateUser(t.Context(), "Frank", "password1")
	require.NoError(t, err)
	past := time.Now().UTC().Add(-time.Minute)
	sess, err := store.CreateSession(t.Context(), u.ID, past)
	require.NoError(t, err)

	resp := getMe(t, srv, []*http.Cookie{{Name: auth.CookieName, Value: sess.ID}})
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Card #2 edge — validation / bad JSON → 400, no user created.
func TestRegister_InvalidInput_Returns400(t *testing.T) {
	srv, store := newAuthServer(t)
	cases := []struct {
		name string
		raw  string
	}{
		{"empty body", ``},
		{"non-json", `not-json`},
		{"missing nick", `{"password":"password1"}`},
		{"missing password", `{"nick":"Ann"}`},
		{"short nick", `{"nick":"ab","password":"password1"}`},
		{"long nick", `{"nick":"` + strings.Repeat("a", 21) + `","password":"password1"}`},
		{"short password", `{"nick":"Ann","password":"short"}`},
		{"long password", `{"nick":"Ann","password":"` + strings.Repeat("x", 129) + `"}`},
		{"whitespace nick", `{"nick":"   ","password":"password1"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := postJSON(t, srv.URL+"/api/register", tc.raw, nil)
			defer resp.Body.Close()
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
			_, err := store.Authenticate(t.Context(), "Ann", "password1")
			require.ErrorIs(t, err, auth.ErrInvalidCredentials)
		})
	}
}

// --- Card #3 — create/join require a valid session cookie ---

// saveCountingStorage wraps Storage and counts SaveGame calls so create-401
// tests can assert no game was persisted (ActiveCount stays 0 even after a
// successful CreateGame — create only SaveGames, it never inserts into active).
type saveCountingStorage struct {
	inner game.Storage
	mu    sync.Mutex
	saves int
}

func (s *saveCountingStorage) LoadGame(id string) (*game.Game, bool) {
	return s.inner.LoadGame(id)
}

func (s *saveCountingStorage) SaveGame(g *game.Game) error {
	s.mu.Lock()
	s.saves++
	s.mu.Unlock()
	return s.inner.SaveGame(g)
}

func (s *saveCountingStorage) SaveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saves
}

func newGateServer(t *testing.T) (*httptest.Server, *saveCountingStorage, *auth.MemoryStore) {
	t.Helper()
	store := auth.NewMemoryStore()
	counting := &saveCountingStorage{inner: game.NewMemoryStorage()}
	mgr := game.NewManagerWithStorage(counting, nil)
	mux := http.NewServeMux()
	rest.NewHandler(mgr, store).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, counting, store
}

func registerCookie(t *testing.T, srv *httptest.Server, nick string) *http.Cookie {
	t.Helper()
	resp := postJSON(t, srv.URL+"/api/register", `{"nick":"`+nick+`","password":"password1"}`, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	c := sessionCookie(resp)
	require.NotNil(t, c)
	return c
}

func decodeCreds(t *testing.T, resp *http.Response) credsBody {
	t.Helper()
	var body credsBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

// Card #3 AC1 / SPEC [NEW 2026-11-11 — card #5] — no session → create 401, no game.
func TestCreate_NoSession_Returns401(t *testing.T) {
	srv, storage, _ := newGateServer(t)

	resp := postJSON(t, srv.URL+"/api/games", `{}`, nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, 0, storage.SaveCount(), "no game must be persisted without a session")
}

// Card #3 AC2 — no session → join 401; seat remains free for an authenticated join.
func TestJoin_NoSession_Returns401(t *testing.T) {
	srv, _, _ := newGateServer(t)
	cookie := registerCookie(t, srv, "CreatorA")
	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{cookie})
	created := decodeCreds(t, createResp)
	createResp.Body.Close()
	require.Equal(t, http.StatusOK, createResp.StatusCode)

	anon := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, nil)
	defer anon.Body.Close()
	require.Equal(t, http.StatusUnauthorized, anon.StatusCode)

	joiner := registerCookie(t, srv, "JoinerA")
	okJoin := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{joiner})
	defer okJoin.Body.Close()
	require.Equal(t, http.StatusOK, okJoin.StatusCode, "seat must still be free after anonymous 401")
}

// Card #3 AC3 — valid session → create returns gameId + seat token.
func TestCreate_WithSession_ReturnsGameIDAndToken(t *testing.T) {
	srv, _, _ := newGateServer(t)
	cookie := registerCookie(t, srv, "CreatorB")

	resp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{cookie})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decodeCreds(t, resp)
	require.NotEmpty(t, body.GameID)
	require.NotEmpty(t, body.Token)
}

// Card #3 AC3 — valid session → join returns a distinct seat token.
func TestJoin_WithSession_ReturnsToken(t *testing.T) {
	srv, _, _ := newGateServer(t)
	creator := registerCookie(t, srv, "CreatorC")
	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{creator})
	created := decodeCreds(t, createResp)
	createResp.Body.Close()

	joiner := registerCookie(t, srv, "JoinerC")
	resp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{joiner})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decodeCreds(t, resp)
	require.Equal(t, created.GameID, body.GameID)
	require.NotEmpty(t, body.Token)
	require.NotEqual(t, created.Token, body.Token)
}

// Card #3 edge — expired session cookie → 401 on create/join.
func TestCreateJoin_ExpiredSession_Returns401(t *testing.T) {
	srv, storage, store := newGateServer(t)
	u, err := store.CreateUser(t.Context(), "ExpiredUser", "password1")
	require.NoError(t, err)
	past := time.Now().UTC().Add(-time.Minute)
	sess, err := store.CreateSession(t.Context(), u.ID, past)
	require.NoError(t, err)
	expired := []*http.Cookie{{Name: auth.CookieName, Value: sess.ID}}

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, expired)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, createResp.StatusCode)
	require.Equal(t, 0, storage.SaveCount(), "no game must be persisted with an expired session")

	// Seed a real game with a fresh session, then join with expired.
	fresh := registerCookie(t, srv, "SeedExpired")
	seed := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{fresh})
	created := decodeCreds(t, seed)
	seed.Body.Close()

	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, expired)
	defer joinResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, joinResp.StatusCode)
}

// Card #3 edge — unknown/forged session cookie → 401 on create/join.
func TestCreateJoin_UnknownSession_Returns401(t *testing.T) {
	srv, storage, _ := newGateServer(t)
	junk := []*http.Cookie{{Name: auth.CookieName, Value: "forged-session-id"}}

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, junk)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, createResp.StatusCode)
	require.Equal(t, 0, storage.SaveCount(), "no game must be persisted with an unknown session")

	fresh := registerCookie(t, srv, "SeedUnknown")
	seed := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{fresh})
	created := decodeCreds(t, seed)
	seed.Body.Close()

	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, junk)
	defer joinResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, joinResp.StatusCode)
}

// Card #3 edge — empty nardy_session value → treated as absent → 401.
func TestCreateJoin_EmptySessionCookie_Returns401(t *testing.T) {
	srv, storage, _ := newGateServer(t)
	empty := []*http.Cookie{{Name: auth.CookieName, Value: ""}}

	createResp := postJSON(t, srv.URL+"/api/games", `{}`, empty)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, createResp.StatusCode)
	require.Equal(t, 0, storage.SaveCount(), "no game must be persisted with an empty session cookie")

	fresh := registerCookie(t, srv, "SeedEmpty")
	seed := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{fresh})
	created := decodeCreds(t, seed)
	seed.Body.Close()

	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, empty)
	defer joinResp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, joinResp.StatusCode)
}

// Card #3 edge — valid session + unknown gameId → 404 (not 401).
func TestJoin_WithSession_UnknownGame_Returns404(t *testing.T) {
	srv, _, _ := newGateServer(t)
	cookie := registerCookie(t, srv, "Joiner404")

	resp := postJSON(t, srv.URL+"/api/games/no-such-game/join", `{}`, []*http.Cookie{cookie})
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// Card #3 edge — valid session + full room → 409 (not 401).
func TestJoin_WithSession_FullGame_Returns409(t *testing.T) {
	srv, _, _ := newGateServer(t)
	creator := registerCookie(t, srv, "CreatorFull")
	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{creator})
	created := decodeCreds(t, createResp)
	createResp.Body.Close()

	joiner1 := registerCookie(t, srv, "JoinerFull1")
	r1 := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{joiner1})
	r1.Body.Close()
	require.Equal(t, http.StatusOK, r1.StatusCode)

	joiner2 := registerCookie(t, srv, "JoinerFull2")
	r2 := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{joiner2})
	defer r2.Body.Close()
	require.Equal(t, http.StatusConflict, r2.StatusCode)
}

// Card #16 — create/join record the session nick on the seat.
func TestCreateJoin_WithSession_RecordsSeatNicks(t *testing.T) {
	srv, storage, _ := newGateServer(t)
	creator := registerCookie(t, srv, "AliceNick")
	createResp := postJSON(t, srv.URL+"/api/games", `{}`, []*http.Cookie{creator})
	created := decodeCreds(t, createResp)
	createResp.Body.Close()
	require.Equal(t, http.StatusOK, createResp.StatusCode)

	g, ok := storage.LoadGame(created.GameID)
	require.True(t, ok)
	require.Equal(t, [2]string{"AliceNick", ""}, g.Nicks())

	joiner := registerCookie(t, srv, "BobNick")
	joinResp := postJSON(t, srv.URL+"/api/games/"+created.GameID+"/join", `{}`, []*http.Cookie{joiner})
	defer joinResp.Body.Close()
	require.Equal(t, http.StatusOK, joinResp.StatusCode)

	g2, ok := storage.LoadGame(created.GameID)
	require.True(t, ok)
	require.Equal(t, [2]string{"AliceNick", "BobNick"}, g2.Nicks())
}
