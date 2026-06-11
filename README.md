# backgammon-game

Онлайн-игра в **длинные нарды** на двоих по приглашению (классические русские правила: оин/марс/кокс, исключение для дублей `6:6`/`4:4`/`3:3` на первом ходу). Сервер — единственный источник правды; клиент получает полное состояние после каждого валидного действия.

Полный стек в одном репозитории: бэкенд на **Go**, фронтенд на **TypeScript + Svelte 5**, реал-тайм через WebSocket. Один образ: Go-сервер отдаёт собранный SPA **и** `/ws` + `/api` с одного origin.

## Статус

MVP готов и собирается в один Docker-образ.

- **Backend** — доменное ядро + транспорт (WS/REST/static) + persistence + auth. TDD-план [SPEC.md](SPEC.md) §6 закрыт целиком (пункты #1–#41, включая REST invite-флоу).
- **Frontend** — все этапы [FRONTEND_SPEC.md](FRONTEND_SPEC.md) §7 закрыты (0–10): доска, кубики, действия, конец игры, Connect, реконнект, invite-флоу, личная ссылка для возврата.
- **Hosting** — `Dockerfile` (многостадийный), `fly.toml`, инструкции под Fly.io / ngrok / VPS — см. [DEPLOY.md](DEPLOY.md).

Ещё не сделано:
- Rate limit на REST (см. [SPEC.md](SPEC.md) §8).
- Drag&drop на доске (MVP — только клики).
- i18n (пока только русский).
- Логирование там, где сейчас `_ = ...`.

## Стек

**Backend**
- Go **1.26**
- WebSocket: `github.com/coder/websocket`
- Postgres: `github.com/jackc/pgx/v5`
- Тесты: `testing` + `github.com/stretchr/testify` + `testcontainers-go` (Docker нужен для интеграционных тестов Postgres)
- Источник случайности для кубиков: `crypto/rand` (rejection sampling)

**Frontend** (`web/`)
- TypeScript (strict) + **Svelte 5** (runes) + **Vite 6**
- Рендер доски — SVG, без UI-библиотек (ванильный CSS)
- Тесты: **Vitest 3** + `@testing-library/svelte` + jsdom
- Транспорт — нативный `WebSocket`, без обёрток

## Структура

```
cmd/server/                — точка входа (выбор Storage по DATABASE_URL, проводка ws/rest/static)
internal/domain/           — чистая игровая логика (доска, кубики, правила, исход); только stdlib
internal/game/             — оркестратор сессий (Manager, Storage-интерфейс, in-memory + Postgres)
internal/protocol/         — типы WS-сообщений (зеркалятся фронтом)
internal/transport/ws/     — WS handshake + read-loop, auth, проверка Origin
internal/transport/rest/   — REST invite-флоу (POST /api/games, /api/games/{id}/join)
internal/transport/static/ — раздача SPA с Cache-Control (no-cache для index.html, immutable для ассетов)
web/                       — SPA на Svelte 5 / TS / Vite (см. FRONTEND_SPEC.md)
```

Доменный пакет не зависит ни от чего, кроме stdlib. Все правила нард — там. Подробности — в [SPEC.md](SPEC.md) §2–3.

## Запуск

### Локально (dev): Go-бэкенд + Vite-dev отдельно

```sh
# 1) Бэкенд на :8080 (in-memory storage, STATIC_DIR не задан → статику отдаёт Vite)
go run ./cmd/server -addr :8080

# 2) В другом терминале — фронт; Vite проксирует /ws и /api на :8080
cd web
npm install
npm run dev        # открыть выданный http://localhost:5173
```

С Postgres (схема создаётся автоматически при старте):

```sh
DATABASE_URL=postgres://user:pass@localhost:5432/nardy go run ./cmd/server -addr :8080
```

### Один образ (prod-like): Go отдаёт собранный SPA + /ws + /api

```sh
docker build -t nardy .
docker run --rm -p 8080:8080 nardy
# открыть http://localhost:8080
```

Переменные окружения:

| Переменная | Назначение |
|---|---|
| `DATABASE_URL` | Postgres DSN. Не задан → in-memory (партии теряются при рестарте). |
| `STATIC_DIR` | Каталог собранного SPA. Задан → Go отдаёт статику с того же origin (в образе = `/app/web`). |
| `ALLOWED_ORIGINS` | Список Origin через запятую, ослабляет same-origin проверку WS. За туннелем/реверс-прокси нужен `*` (см. [DEPLOY.md](DEPLOY.md)). |

Деплой (Fly.io / ngrok / VPS) — [DEPLOY.md](DEPLOY.md).

## Тесты

```sh
# Backend: доменные + транспортные + интеграционные
go test ./...
go test -cover ./...
go test ./internal/domain/...   # только домен, без Docker

# Frontend
cd web
npm test                        # vitest run
npm run check && npm run lint   # типы + линт
```

Интеграционные тесты `internal/game` (Postgres через testcontainers) **требуют запущенный Docker**.

## Поток партии

Игрок A создаёт игру (`POST /api/games`) → получает `gameId`, личный `token` и ссылку-приглашение `?game=<id>`. Игрок B открывает ссылку, входит (`POST /api/games/{id}/join`), оба подключаются по WS и делают `JOIN`. Дальше — `ROLL_FOR_FIRST` → `ROLL` → `MOVE`… Подробности — [SPEC.md](SPEC.md) §5.

## WebSocket-протокол

Полная таблица сообщений — [SPEC.md](SPEC.md) §4. Авторизация — Bearer-токен в `Authorization`-заголовке WS-хендшейка. Краткая шпаргалка:

**Клиент → сервер:** `JOIN`, `ROLL_FOR_FIRST`, `ROLL`, `MOVE`, `END_TURN`, `RESIGN`.

**Сервер → клиент:** `STATE` (полный снапшот), `LEGAL_MOVES`, `OPPONENT_JOINED`, `OPPONENT_LEFT`, `GAME_OVER`, `ERROR`.

**Коды ошибок (`ERROR.code`):** `UNAUTHORIZED`, `ROOM_FULL`, `NOT_YOUR_TURN`, `INVALID_STATE`, `MUST_USE_PIP`, `RULE_OF_SIX`.

## Документация

- [SPEC.md](SPEC.md) — игровые правила, архитектура, WS-протокол, TDD-план + прогресс.
- [FRONTEND_SPEC.md](FRONTEND_SPEC.md) — стек, экраны, поток данных и TDD-план фронтенда.
- [DEPLOY.md](DEPLOY.md) — Docker, Fly.io, self-host через туннель.
- `.claude/skills/nardy-rules/SKILL.md` — авторитативный референс правил.
- `.claude/skills/nardy-tdd/SKILL.md` — TDD-дисциплина бэкенда.
- `.claude/skills/nardy-frontend/SKILL.md` — архитектура фронтенда и его TDD.
- `.claude/skills/nardy-svelte/SKILL.md` — конвенции Svelte 5.
- `.claude/skills/nardy-protocol/SKILL.md` — детали WS-протокола.
