// Package static отдаёт собранный SPA из каталога с корректными заголовками
// кеширования: index.html ревалидируется на каждый заход, а хешированные
// Vite-ассеты кешируются навсегда.
package static

import "net/http"

// Handler оборачивает http.FileServer для каталога dir.
//
// TODO(green): добавить Cache-Control по пути запроса.
func Handler(dir string) http.Handler {
	return http.FileServer(http.Dir(dir))
}
