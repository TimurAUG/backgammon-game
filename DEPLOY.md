# Деплой

Способы: **ngrok** (без карты, со своей машины — проверено, см. ниже), **Fly.io**
(если карту принимают), **VPS** (CIS-провайдеры под рос. карты). Образ один и тот же.

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

## Без карты: self-host + туннель (проверено — ngrok)

PaaS не принял карту? Публикуем игру со своей машины через туннель.
**Рабочий вариант в фильтрующей сети — ngrok** (тащит и HTTP, и WebSocket;
бесплатный аккаунт по email, без карты).

Предусловие: образ собран — `docker build -t nardy .`

```sh
# 1) Освободить :8080, если занят dev-сервером:
lsof -ti tcp:8080 | xargs kill 2>/dev/null

# 2) Прод-контейнер. ALLOWED_ORIGINS='*' ОБЯЗАТЕЛЕН за туннелем (см. ниже):
docker run --rm -p 8080:8080 -e ALLOWED_ORIGINS='*' nardy

# 3) В другом терминале — ngrok:
brew install ngrok
ngrok config add-authtoken <токен с dashboard.ngrok.com>
ngrok http 8080
# → Forwarding https://<random>.ngrok-free.dev
```

Открываешь выданный `https`-адрес → ngrok покажет страницу-предупреждение →
**«Visit Site»** (один раз на браузер) → **«Создать игру»** → копируешь
ссылку-приглашение (тот же ngrok-адрес + `?game=...`) и отдаёшь другу. Он
открывает → «Visit Site» → «Войти в игру» → играете.

**Важно:**
- `ALLOWED_ORIGINS='*'` обязателен за любым туннелем/реверс-прокси: там `Host` ≠
  `Origin`, и строгая проверка WebSocket иначе вернёт 403 (`fetch`-API проходит,
  а `/ws` — нет, и игра «висит»).
- Контейнер **и** ngrok должны быть запущены, пока играете.
- На free-плане URL меняется при перезапуске ngrok. Стабильный — статический
  домен ngrok: `ngrok http --url=<твой>.ngrok-free.app 8080` (берётся в дашборде).
- Страницу-предупреждение ngrok показывает только на навигацию; `fetch`-запросы
  SPA проходят, так что создание/вход в игру работают после одного клика.

**Что НЕ сработало в фильтрующей сети (не трать время):**
- **Cloudflare quick tunnel** (`cloudflared --url`) — провайдер режет POST
  создания туннеля (DPI): `unexpected EOF`, хотя сам Cloudflare доступен.
- **pinggy free** (`ssh -p443 a.pinggy.io`) — отдаёт HTTP, но **не проксирует
  WebSocket** (upgrade висит/таймаутит) → игра не идёт.

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
