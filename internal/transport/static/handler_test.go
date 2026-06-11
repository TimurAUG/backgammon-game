package static

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// staticDir создаёт временный каталог с index.html и хешированным ассетом —
// имитация вывода `vite build`.
func staticDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "index-abc123.js"), []byte("0"), 0o644))
	return dir
}

func cacheControl(t *testing.T, h http.Handler, path string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, http.StatusOK, rec.Code, "ожидался 200 на %s", path)
	return rec.Header().Get("Cache-Control")
}

// TestHandler_IndexHTML_NoCache — точка входа SPA всегда ревалидируется,
// иначе старый index.html из кеша ссылается на устаревшие бандлы.
func TestHandler_IndexHTML_NoCache(t *testing.T) {
	h := Handler(staticDir(t))
	require.Equal(t, "no-cache", cacheControl(t, h, "/"),
		"SPA-точка входа (/) должна отдаваться с Cache-Control: no-cache")
}

// TestHandler_HashedAsset_Immutable — имя ассета содержит хеш контента, поэтому
// файл можно кешировать навсегда: при изменении кода меняется имя.
func TestHandler_HashedAsset_Immutable(t *testing.T) {
	h := Handler(staticDir(t))
	require.Equal(t, "public, max-age=31536000, immutable",
		cacheControl(t, h, "/assets/index-abc123.js"),
		"хешированный ассет должен кешироваться как immutable")
}
