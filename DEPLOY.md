# Деплой на Fly.io

Одно приложение: Go-сервер отдаёт собранный SPA (`web/dist`) **и** `/ws` + `/api`
с одного origin. Fly терминирует TLS → клиент автоматически ходит по `https`/`wss`
(фронт выбирает схему по `location.protocol`). Same-origin снимает вопросы CORS и
проверки Origin у WebSocket.

## Локальная проверка образа (не нужен Fly)

```sh
docker build -t nardy .
docker run --rm -p 8080:8080 nardy
# открыть http://localhost:8080 — SPA, /ws и /api на одном порту
```

## Без карты: self-host + Cloudflare Tunnel

Если платёжка PaaS не принимает карту — публикуем со своей машины через
бесплатный временный туннель (без аккаунта и карты):

```sh
brew install cloudflared
docker run --rm -p 8080:8080 -e ALLOWED_ORIGINS='*' nardy
# в другом терминале:
cloudflared tunnel --url http://localhost:8080
# → https://<random>.trycloudflare.com — отдаёшь другу
```

`ALLOWED_ORIGINS='*'` обязателен за туннелем/реверс-прокси: там `Host` ≠ `Origin`,
и строгая проверка WebSocket иначе вернёт 403. Туннель даёт HTTPS → `wss` работает.
URL временный (меняется при перезапуске); для постоянного — именованный туннель
(бесплатный Cloudflare-аккаунт) или VPS с Caddy/доменом.

## Первый деплой

```sh
brew install flyctl              # или https://fly.io/docs/flyctl/install/
fly auth login
fly apps create <имя>            # затем впиши <имя> в app в fly.toml
fly deploy                       # билдит Dockerfile и катит
```

`fly launch` тоже подойдёт — он сам сгенерит `fly.toml` и обнаружит `Dockerfile`.

## Хранилище

По умолчанию **in-memory** — `DATABASE_URL` не задан, партии живут в памяти и
теряются при рестарте машины. Для персистентности:

```sh
fly postgres create
fly postgres attach <pg-app>     # выставит секрет DATABASE_URL
```

Сервер сам поднимет схему при старте (`InitSchema`).

## Заметки

- `force_https = true` — http→https редирект; публичный эндпойнт `wss://<app>.fly.dev/ws`.
- `auto_stop_machines` усыпляет простаивающую машину; активные WS-соединения держат
  её живой. Для гарантии «без холодного старта посреди партии» подними
  `min_machines_running = 1` (дороже).
