// Package static отдаёт собранный SPA из каталога с корректными заголовками
// кеширования: index.html ревалидируется на каждый заход, а хешированные
// Vite-ассеты кешируются навсегда.
package static

import (
	"net/http"
	"strings"
)

// Handler оборачивает http.FileServer для каталога dir, добавляя заголовки
// Cache-Control в зависимости от пути запроса.
func Handler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCacheControl(w.Header(), r.URL.Path)
		fs.ServeHTTP(w, r)
	})
}

// setCacheControl выставляет политику кеширования по пути запроса:
//   - /assets/* — хешированные Vite-бандлы: имя файла меняется при изменении
//     контента, поэтому кешируем навсегда (immutable);
//   - всё остальное (index.html и SPA-роуты) — no-cache: браузер каждый раз
//     ревалидирует точку входа, чтобы подхватить свежие имена бандлов.
func setCacheControl(h http.Header, path string) {
	if strings.HasPrefix(path, "/assets/") {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	h.Set("Cache-Control", "no-cache")
}
