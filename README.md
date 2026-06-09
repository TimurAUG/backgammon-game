# backgammon-game

Онлайн-игра в **длинные нарды** (классические русские правила: оин/марс/кокс, исключение для дублей `6:6`/`4:4`/`3:3` на первом ходу). Сервер — единственный источник правды; клиент получает полное состояние после каждого валидного действия.

Бэкенд на Go, реал-тайм через WebSocket. Фронтенд (TypeScript) — следующая фаза.

## Статус

Backend MVP готов: доменное ядро + транспорт + persistence + auth. Весь TDD-план из [SPEC.md](SPEC.md) (пункты #1–#37) закрыт.

Что ещё не сделано:
- Frontend (TypeScript).
- Hosting: `Dockerfile`, `fly.toml`.
- REST endpoint для генерации/проверки токенов (invite-флоу).
- Rate limit, WSS/HTTPS (см. раздел 8 `SPEC.md`).
- Логирование там, где сейчас `_ = ...`.

## Стек

- Go **1.26+**
- WebSocket: `github.com/coder/websocket`
- Postgres: `github.com/jackc/pgx/v5`
- Тесты: `testing` + `github.com/stretchr/testify` + `testcontainers-go` (требует Docker для интеграционных тестов Postgres)
- Источник случайности для кубиков: `crypto/rand` (rejection sampling)

## Структура

```
cmd/server/                — точка входа
internal/domain/           — чистая игровая логика (доска, кубики, правила, исход)
internal/game/             — оркестратор сессий (Manager, Storage, Postgres-реализация)
internal/protocol/         — типы WS-сообщений
internal/transport/ws/     — HTTP-handler + WS read-loop
```

Доменный пакет не зависит ни от чего, кроме stdlib. Все правила нард — там. Подробности — в [SPEC.md](SPEC.md), разделы 2–3.

## Запуск

```sh
# in-memory storage (DATABASE_URL не задан)
go run ./cmd/server -addr :8080

# с Postgres (схема создаётся автоматически при старте)
DATABASE_URL=postgres://user:pass@localhost:5432/nardy go run ./cmd/server -addr :8080
```

Подключение клиента (любой непустой Bearer-токен сейчас принимается; auth-флоу через REST — не реализован):

```sh
wscat -c ws://localhost:8080/ws -H "Authorization: Bearer любая-строка"
> {"type":"JOIN","gameId":"g1"}
```

## Тесты

```sh
# доменные + транспортные тесты
go test ./...

# с покрытием
go test -cover ./...

# только домен (без Docker)
go test ./internal/domain/...
```

Интеграционные тесты `internal/game` (Postgres через testcontainers) **требуют запущенный Docker**.

## WebSocket-протокол

Полная таблица сообщений — [SPEC.md](SPEC.md), раздел 4. Краткая шпаргалка:

**Клиент → сервер:** `JOIN`, `ROLL_FOR_FIRST`, `ROLL`, `MOVE`, `END_TURN`, `RESIGN`.

**Сервер → клиент:** `STATE` (полный снапшот), `LEGAL_MOVES`, `OPPONENT_JOINED`, `OPPONENT_LEFT`, `GAME_OVER`, `ERROR`.

## Документация

- [SPEC.md](SPEC.md) — игровые правила, архитектура, WS-протокол, TDD-план + прогресс.
- `.claude/skills/nardy-rules/SKILL.md` — авторитативный референс правил.
- `.claude/skills/nardy-tdd/SKILL.md` — TDD-дисциплина проекта.
- `.claude/skills/nardy-protocol/SKILL.md` — детали WS-протокола.
