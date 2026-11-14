# Frontend — спецификация

TypeScript-клиент длинных нард. Подключается к Go-серверу по WebSocket, отображает доску, принимает ходы, реагирует на серверные обновления состояния. Сервер — единственный источник правды; клиент только рендерит `STATE` и шлёт действия.

Источники, на которые ссылается этот документ:
- [`SPEC.md`](SPEC.md) — игровые правила, общая архитектура.
- `.claude/skills/nardy-protocol/SKILL.md` — авторитативный референс WS-протокола.
- `.claude/skills/nardy-rules/SKILL.md` — правила длинных нард.
- `.claude/skills/nardy-tdd/SKILL.md` — TDD-дисциплина проекта.

При расхождении этих документов с FRONTEND_SPEC.md приоритет — за ними.

---

## 1. Стек и инструменты

- **TypeScript** строгий (`strict: true`).
- **Svelte 5** + **Vite** — фреймворк и сборка.
- **SVG** — рендер доски и шашек (см. § 6).
- **Vitest** + **@testing-library/svelte** + **jsdom** — тесты.
- **ESLint** + **Prettier** — линт и форматирование.
- Менеджер пакетов — **npm** (без доп. установок).
- Транспорт — нативный `WebSocket` (без обёрток).

Без UI-библиотек (никаких Tailwind/Material) в MVP — ванильный CSS. Если позже понадобится — добавим явным циклом.

## 2. Структура

```
web/
  index.html
  package.json
  tsconfig.json
  vite.config.ts
  svelte.config.js
  src/
    main.ts              — bootstrap
    App.svelte           — root: роутинг Connect/Game, проводка WSClient, оверлей уведомлений
    protocol/
      messages.ts        — TS-зеркало internal/protocol/messages.go
    transport/
      ws.ts              — WSClient: connect/send/onMessage/реконнект
    stores/
      game.svelte.ts          — board/turn/dice/borneOff/status/isFirstMove/allHome/legalMoves/gameOver/myColor/firstRoll/started
      connection.svelte.ts    — state: idle | connecting | connected | reconnecting | error
      notifications.svelte.ts — стек тостов (соперник присоединился / «Твой бросок»)
      chat.svelte.ts          — сообщения чата партии + счётчик непрочитанных + open
    screens/
      Connect.svelte     — создать игру / войти по приглашению / ручной ввод; persistence в localStorage
      Game.svelte        — Board + BearOffCounter + Dice + ActionBar + GameOver + Chat + баннеры (первый бросок, переподключение)
    components/
      Board.svelte       — SVG-доска, шашки и клик-режим ходов
      Dice.svelte        — два кубика + оставшиеся пипсы
      BearOffCounter.svelte — счётчик оставшихся к сбросу шашек (осталось/15) в фазе сброса, для обоих цветов
      ActionBar.svelte   — контекстные кнопки ROLL_FOR_FIRST/ROLL/END_TURN/RESIGN
      GameOver.svelte    — модалка результата
      Toast.svelte       — одна плашка уведомления (role=status/aria-live, авто-скрытие)
      Notifications.svelte — контейнер тостов из стора
      Chat.svelte          — сворачиваемая панель чата партии (лента + ввод + бейдж непрочитанных)
    lib/
      geometry.ts        — координаты SVG для пункта i и j-й шашки на пункте
      credentials.ts     — gameId+token в localStorage
      api.ts             — REST invite-клиент (createGame/joinGame)
      sound.ts           — playRollCue() короткий сигнал через Web Audio

[AMENDED 2026-11-11 — card #5, source card #1] `Connect.svelte` also hosts nick + password fields and «Log in» / «Register» actions on the same page (no new screen). `api.ts` covers register/login/me/logout and sends the session cookie (`credentials: 'include'`) on create/join; per-game seat token persistence in `credentials.ts` (`nardy.credentials`) stays as it is after a successful create/join.

[AMENDED 2026-11-12 — card #12, source card #9] `Connect.svelte` offers create-game and invite join only (plus nick/password from #4/#5); no visible manual `gameId`/`token` fallback. Seat-token `localStorage` after successful create/join and personal return `?game=&token=` stay.

[AMENDED 2026-11-12 — card #14, source card #10] Logged-in Connect shows a visible «Выйти» control next to the nick (before and after create). The Game toolbar includes «Выйти» alongside reconnect copy and «Сменить игру». `api.ts` `logout()` is the client call those controls use (`POST /api/logout` with `credentials: 'include'`).

[AMENDED 2026-11-13 — card #20, source card #18] Connect is three states on the same screen (Figma Вход / Вошли / Игра создана): login form only until `GET /api/me` succeeds; then logged-in lobby with «Создать игру» and «Выйти из аккаунта»; after create — invite link / «Копировать» / «Войти в игру». `ActionBar.svelte` exposes ROLL_FOR_FIRST / ROLL / RESIGN only — no visible END_TURN («Завершить ход»). Mid-game Game chrome has no return link, no header «Копировать», no «Выйти», no «Сменить игру». Account logout is only «Выйти из аккаунта» on the logged-in lobby (Вошли), via `api.ts` `logout()` (`POST /api/logout` with `credentials: 'include'`).

[AMENDED 2026-11-13 — card #24, source card #21] On Вошли, under «Выйти из аккаунта», Connect shows a «Статистика» block (same screen; no new route). `api.ts` `me()` parses `{ nick, played, wins, losses, win_percent }` from `GET /api/me` and the lobby renders those four numbers; the client does not invent or recompute them. The block is absent on Вход, on Игра создана, and on the board / Game chrome.
  tests/                 — общие тестовые утилиты, mock WebSocket, фикстуры STATE
```

`/web` живёт в этом репо. Это упрощает синхронизацию типов с `internal/protocol/messages.go` — при изменении схемы достаточно одного PR.

## 3. Поток данных

```
WebSocket ─► WSClient ─► messages.ts (parse) ─► gameStore (reducer-style)
                                                       │
                                            Svelte reactivity
                                                       ▼
                                          UI (Board/Dice/...) рендерит из store

UI событие ─► gameStore action ─► WSClient.send ─► WebSocket
```

Сторы держат **только** то, что прислал сервер. Никаких локальных правил игры — все валидации на бекенде. Подсветка легальных ходов = фильтр `legalMoves` из последнего `LEGAL_MOVES`.

## 4. Экраны

### Connect

Форма с двумя полями: `gameId` и `token`. Кнопка «Подключиться» → сохранение в `localStorage` → открытие WS с `Authorization: Bearer <token>` → автоматический `JOIN`.

При наличии сохранённых `gameId`+`token` в `localStorage` — пропускает форму и сразу подключается (см. § 5, реконнект).

[AMENDED 2026-11-11 — card #5, source card #1] On the **same** Connect page (current type and colors; no new screen, pattern, or mockup), the player also sees nick and password fields plus «Log in» and «Register» buttons. Register → `POST /api/register`; Log in → `POST /api/login`. Both rely on the browser storing the session cookie (HttpOnly); closing the tab must not clear that cookie. Create-game and join-game requests include the session cookie (`credentials: 'include'`). No email, real name, or other personal-data fields. Duplicate register surfaces as 409; failed login as the shared 401 — without inventing new layout patterns. Existing invite / manual gameId+token / seat-token localStorage behaviour remains.

[AMENDED 2026-11-12 — card #12, source card #9] Connect no longer shows a visible manual `gameId`/`token` form («или войти вручную», «ID игры», «Токен», «Подключиться»). Entry is create-game and invite join (`?game=`) only, plus nick/password Log in / Register from cards #4/#5. After successful create or join, seat-token persistence in `localStorage` (`nardy.credentials`) and personal return via `?game=&token=` remain; saved credentials still auto-connect when appropriate (§ 5).

[AMENDED 2026-11-12 — card #14, source card #10] When `GET /api/me` returns a nick, Connect shows that nick with a «Выйти» control next to it — both before create («Создать игру» visible) and after create (invite link / «Войти в игру» still showing the same nick). «Выйти» is not a new screen: current type and colors. On success it calls `POST /api/logout` with the session cookie, clears `nardy.credentials`, closes any open socket, and returns to the nick/password Log in / Register form (not the invite surface). A failed or non-OK logout keeps the nick and seat token and surfaces the existing Connect `.error` alert. Other browser sessions of the same nick stay valid.

[AMENDED 2026-11-13 — card #20, source card #18] Connect follows Figma Нарды as three states on the same page (current type and colors; no new route). **Вход** (no session): nick, password, Log in, Register only — no «Создать игру», no join. **Вошли** (`GET /api/me` returns nick): «Вы вошли как <ник>», «Создать игру», and directly under it same-size «Выйти из аккаунта» (replaces the #14 nick-row «Выйти» label). **Игра создана** (after successful create): «Игра создана. Отправь сопернику ссылку:», invite link, «Копировать», «Войти в игру». Create and invite join require the session cookie; without a session, even `?game=` or a saved seat token must not create/join or open the board — login first. Manual gameId/token form stays absent (#12). «Выйти из аккаунта» → `POST /api/logout`, clear `nardy.credentials`, close socket, show Вход. Failed logout keeps nick/seat and Connect `.error`. Other browser sessions of the same nick stay valid.

[AMENDED 2026-11-13 — card #24, source card #21] **Вошли** also shows, directly under «Выйти из аккаунта», a block titled «Статистика» with four rows (label left / value right, current Connect type and colors): «Сыграно игр» ← `played`, «Победы» ← `wins`, «Поражения» ← `losses`, «Процент побед» ← `win_percent` with a `%` suffix (server integer; client does not recompute — e.g. server `64` → «64%»). When the server reports zeros, show 0 / 0 / 0 / 0%. Numbers come only from a successful `GET /api/me` (`api.ts` `me()`); a failed session read must not render 0 / 0 / 0 / 0% as if that were the record. **Вход**, **Игра создана**, and the Game / board screens do not show the stats block. No separate account screen. Logout behaviour on Вошли is unchanged (#20).

### Game

- **Board** — SVG, 24 пункта (треугольники) + bar + bear-off трей. Рендерится из `gameStore.board`.
- **Dice** — два кубика; рядом — оставшиеся пипсы.
- **ActionBar** — кнопки появляются по контексту:
  - `ROLL_FOR_FIRST` — `status == "waitingForRoll"` И ещё не определён первый ход.
  - `ROLL` — `status == "waitingForRoll"` И мой ход.
  - `END_TURN` — `status == "waitingForMove"` И мой ход.
  - `RESIGN` — всегда, пока игра не закончена.
- Выбор шашки кликом → подсвечиваются клетки `to` из `legalMoves` для этого `from`. Клик по подсвеченной → отправка `MOVE`.

[AMENDED 2026-11-12 — card #17, source card #15] Game chrome also shows a seat-nick block (same screen; no new panel or pattern): line 1 is the current player's nick and their color word («Белые» / «Чёрные» from `myColor`); line 2 is the opponent's nick. Seat nicks come from `STATE.nicks { white, black }` in the game store — the client must not invent them. A seat with an empty nick string displays as «Гость». Board, Dice, ActionBar, first-roll / «Твой бросок» banners, and BearOffCounter keep color words. Toolbar controls («Копировать», «Сменить игру», reconnect banner) stay.

[AMENDED 2026-11-14 — card #25] Empty **opponent** nick on the Game toolbar shows italic «Ожидание игрока...» (not «Гость»). Own empty nick and Chat empty-nick author labels stay «Гость». A nick-only `STATE` must not clear `gameState.firstRoll`; after the opening is decided, reconnect recovers `FIRST_ROLL` from the server so ActionBar does not stick on `waiting-first-roll`.

[AMENDED 2026-11-12 — card #16 Verify] The seat-nick block is in the **top-left** of the Game toolbar (first chrome on the left), not top-right. Own line remains `{nick} · {Белые|Чёрные}`; opponent line remains nick or «Гость».

[AMENDED 2026-11-14 — card #25] Opponent line: nick or italic «Ожидание игрока...» when empty. Reconnect / «Копировать» follow the nick block; «Выйти» / «Сменить игру» stay on the right.

[AMENDED 2026-11-13 — card #20, source card #18] Mid-game Game chrome: seat-nick block (top-left, #16/#17) and reconnect status stay; no personal return link («Ссылка для возврата»), no header «Копировать», no «Выйти», no «Сменить игру». ActionBar shows «Бросить за первый ход» (`ROLL_FOR_FIRST`) and «Бросить кубики» (`ROLL`) as today, and «Сдаться» (`RESIGN`); it does not show «Завершить ход» (`END_TURN`) and does not replace it — automatic pass when there is no legal move stays. Chat and nicks stay. The only ways off the board mid-game are resign (loss by Кокс) or finishing by the existing win rules.

Drag&drop (этап 15) — перетаскивание шашек поверх клик-режима через Pointer Events; клики остаются.

### GameOver

Overlay поверх Game при `status == "finished"`. Показывает `winner` (мой/соперник) и `kind` (Оин/Марс/Кокс). Кнопка «Новая игра» — `POST /api/games`, сразу входит в новую сессию (новые `gameId`+token в creds и `?game=` в URL); законченная партия остаётся позади. При ошибке создания оверлей остаётся с сообщением; при 401 — в Connect без stale `?game=`. Mid-game «Сменить игру» по-прежнему только `endSession` → Connect (не create-and-enter).

[AMENDED 2026-11-12 — card #14, source card #10] Game chrome (toolbar): alongside the return link / reconnect copy, «Копировать», and «Сменить игру», the header includes «Выйти». «Сменить игру» remains a game switch only (`endSession` → Connect) and must not call logout or clear the account session. «Выйти» uses the same logout path as Connect: `POST /api/logout` with credentials, clear `nardy.credentials`, close the socket, show the nick/password Log in / Register form.

[AMENDED 2026-11-13 — card #20, source card #18] GameOver's only action is «Выйти» (not «Новая игра»). It ends the seat session (`endSession` / clear `nardy.credentials`, close socket) and returns to Вошли — still logged in as the same nick. It must not `POST /api/games`, must not create-and-enter, and must not account-logout. Outcome copy (won/lost, color, Оин/Марс/Кокс) is unchanged. Supersedes #6 Option B / TDD #22 create-and-enter. Mid-game «Сменить игру» and Game-toolbar «Выйти» are removed (see Game chrome amendment above).

## 5. Транспорт и резильентность

- При первом подключении: WSClient открывает сокет, ждёт `open`, шлёт `JOIN`.
- Реконнект — экспоненциальный backoff (1с / 2с / 4с / 8с, потолок 30с). При успешном повторном `open` — снова `JOIN` с теми же `gameId`+`token`, сервер вернёт текущий `STATE`.
- Токен живёт в `localStorage`. При `ERROR { code: "UNAUTHORIZED" }` — чистим и возвращаем в Connect.
- `connectionStore` отражает состояние сокета. UI блокирует ActionBar при `reconnecting`.

[NEW 2026-11-11 — card #5, source card #1] User session resilience: the login session cookie is independent of the per-game seat token in `localStorage`. Tab close does not end the user session; after reopening the app, `GET /api/me` with the cookie still returns the nick until expiry or logout. Create/join REST calls must send that cookie; WS still authenticates with the game seat token only.

[AMENDED 2026-11-13 — card #24, source card #21] After reopening the app, a successful `GET /api/me` returns nick plus `played`, `wins`, `losses`, and `win_percent` for the Вошли stats block until expiry or logout. `api.ts` `me()` is the only client source for those numbers. Create/join cookie and WS seat-token rules are unchanged.

[AMENDED 2026-11-12 — card #14, source card #10] The client exposes logout on every logged-in surface (Connect nick row and Game toolbar). Successful «Выйти» ends only this browser's user session (`POST /api/logout` with the cookie), clears the per-game seat credentials (`nardy.credentials`), closes the socket, and shows the nick/password Log in / Register form. Tab close is still not logout: the cookie survives until «Выйти» or expiry. Sibling sessions of the same nick in other browsers stay logged in. Logout with a missing or invalid cookie is idempotent (API 200) and still leaves the client on the login form; network / non-OK responses do not clear nick or seat token until logout succeeds.

[AMENDED 2026-11-13 — card #20, source card #18] Auth gate: App must not auto-start a seat session from saved `nardy.credentials` or `?game=&token=` unless `GET /api/me` shows a logged-in nick; otherwise show Вход. Refresh during a live game with both the session cookie and the seat token may still reconnect. Account logout is only «Выйти из аккаунта» on Вошли (not on the Game toolbar). GameOver «Выйти» clears the seat only so the next open is Вошли, not the finished game, and does not call `POST /api/logout`. Tab close still does not end the account session; sibling sessions of the same nick stay valid. Failed account logout keeps nick/seat until logout succeeds.

## 6. SVG-доска

Координатная система:
- Виртуальный размер `800 × 600` (viewBox), реальный — растяжимый.
- Пункты пронумерованы 1–24 как в `SPEC.md` § 1. Геометрия в `lib/geometry.ts`: чистые функции `pointAnchor(i) → {x, y, dir}` и `checkerAt(i, j) → {cx, cy}` (где `j` — индекс шашки в столбике).
- Кодировка `board[i]` (sm. § 4 в `nardy-protocol`): знак — цвет (`>0` белые, `<0` чёрные), модуль — количество шашек.

Покрытие тестами:
- Геометрия — чистая, тестируется напрямую (для пункта 1 шашки идут вверх, для пункта 24 — вниз, и т.п.).
- Рендер — компонентный тест: при заданном board[] проверяем число шашек на каждом пункте.

## 7. План TDD (порядок)

Каждый пункт — отдельный коммит: `red` → `green` → `refactor`. Для этапа с 3+ пунктами — группируем по парам (2 коммита на пару). Если пункт затрагивает больше 2 файлов — разбиваем на под-циклы `a/b/c`.

### Этап 0 — каркас (без TDD, glue/scaffolding)

Один коммит: `package.json`, `vite.config.ts`, `tsconfig.json`, `svelte.config.js`, `eslint.config.js`, `.prettierrc`, `vitest.config.ts`, `src/main.ts`, `src/App.svelte` с "hello".

### Этап 1 — типы протокола

1. `ClientMessage` сериализация: JOIN/ROLL/MOVE/END_TURN/RESIGN дают ожидаемый JSON.
2. `ServerMessage` парсинг: STATE с полным набором полей, LEGAL_MOVES, ERROR, GAME_OVER.

### Этап 2 — WSClient

3. `connect(url)` открывает сокет, `send(msg)` сериализует и шлёт.
4. `onMessage(cb)` парсит входящие через ServerMessage и зовёт callback.
5. Реконнект с экспоненциальным backoff после `close`.
6. Автоматический `JOIN` после `open`.

### Этап 3 — gameStore (reducer-style)

7. `STATE` обновляет board/turn/dice/borneOff/status/isFirstMove.
8. `LEGAL_MOVES` обновляет `legalMoves`; пустой массив = ход пропускается.
9. `GAME_OVER` устанавливает `gameOver { winner, kind }`, status → "finished".
10. `ERROR` не меняет state (только в connectionStore лог).

### Этап 4 — геометрия и доска

11. `lib/geometry.ts`: `pointAnchor(i)` для всех 24 пунктов.
12. `lib/geometry.ts`: `checkerAt(i, j)` — позиция j-й шашки в столбике.
13. `Board.svelte`: рендер 24 пунктов из `gameStore.board`.
14. `Board.svelte`: рендер шашек с правильным цветом и количеством.

### Этап 5 — кубики

15. `Dice.svelte`: рендер двух точек кубиков из `dice.a` и `dice.b`.
16. `Dice.svelte`: рендер `remaining` (при дубле — 4 одинаковых).

### Этап 6 — действия игрока

17. `ActionBar`: ROLL_FOR_FIRST показывается и шлёт сообщение.
18. `ActionBar`: ROLL/END_TURN/RESIGN — то же по своим условиям.
[AMENDED 2026-11-13 — card #20, source card #18] 18. `ActionBar`: ROLL/RESIGN — same by their conditions; END_TURN («Завершить ход») is not shown and is not replaced by another control.
19. `Board`: клик по своей шашке выделяет её, рендерит подсветку `to` из legalMoves.
20. `Board`: клик по подсвеченной клетке шлёт `MOVE { from, to }`.

### Этап 7 — конец игры

21. `GameOver.svelte` показывается при `gameOver != null` с winner+kind.
22. «Новая игра» создаёт новую игру (`POST /api/games`) и сразу входит в неё; URL/`gameId` — новой партии. (Amended by card #6 — was return-to-Connect only.)
[AMENDED 2026-11-13 — card #20, source card #18] 22. GameOver «Выйти» ends the seat session and returns to the logged-in lobby (Вошли); it does not `POST /api/games` or create-and-enter. (Supersedes card #6 create-and-enter.)

### Этап 8 — Connect и реконнект

23. `Connect.svelte`: форма gameId+token, сохранение в localStorage.
24. `App.svelte`: при наличии сохранённых — авто-подключение.
25. `UNAUTHORIZED` чистит localStorage и роутит в Connect.
26. `connectionStore.reconnecting` блокирует ActionBar.

### Этап 11 — уведомления о событиях

Тосты-уведомления (стек, авто-скрытие, `aria-live`) на ключевые события. Звук — только «Твой бросок» (синтез через Web Audio, без бинарных ассетов в репо).

31. `stores/notifications.svelte.ts`: модульный `$state` со списком тостов + `pushNotification`/`dismissNotification`/`resetNotifications` (по образцу `connection`).
32. `lib/sound.ts`: `playRollCue()` — короткий сигнал через Web Audio; конструктор `AudioContext` инжектируется, без поддержки/при ошибке — no-op.
33. `components/Toast.svelte` + `Notifications.svelte`: рендер тостов из стора, `role="status"`/`aria-live`, ручное закрытие, авто-скрытие по таймеру.
34. Проводка событий:
    - a. `OPPONENT_JOINED` → тост «Соперник присоединился» (в `App`, рядом с маршрутизацией сообщений; `resetNotifications` в `endSession`).
    - b. Переход «ожидается мой бросок» (`status == waitingForRoll` и мой ход на обычном ходу, либо стадия розыгрыша первого хода) → тост «Твой бросок» + `playRollCue()` (в `Game`, через `$effect`-детектор перехода `false→true`).

### Этап 14 — чат партии

Сворачиваемая панель чата на экране Game. Сервер — источник правды: рисуем только то, что пришло в `CHAT`/`CHAT_HISTORY`, без оптимистичного эха своего сообщения. Отправитель — цвет (`sender`); сопоставляем с `myColor` → «своё/чужое».

[AMENDED 2026-11-12 — card #17, source card #15] The **visible** chat author label is the seat nick from `STATE.nicks` for that message's `sender` color (empty nick → «Гость»), including after `CHAT_HISTORY` on reconnect — not the color words «Белые» / «Чёрные». Wire `sender` stays `"white"` / `"black"` and still drives mine/theirs layout (`mine` when `sender === myColor`). Chat does not carry a separate nick field; labels resolve from the store's `nicks` snapshot. Board / dice / first-roll banners remain color words.

35. `protocol/messages.ts`: `ClientMessage += {type:'CHAT', text}`; `ServerMessage += {type:'CHAT', sender, text}` и `{type:'CHAT_HISTORY', chat}`. Сериализация CHAT и парсинг CHAT/CHAT_HISTORY.
36. `stores/chat.svelte.ts`: `$state` `{messages, unread, open}`; `applyChat` (push; `++unread`, если панель закрыта), `applyChatHistory` (replace), `setChatOpen` (`open=true` обнуляет `unread`), `resetChat`.
37. `components/Chat.svelte`: рендер ленты из стора; своё сообщение (`sender == myColor`) и чужое различаются (класс/выравнивание).
38. `components/Chat.svelte`: поле ввода + отправка (Enter/кнопка) → `onAction({type:'CHAT', text})`, поле очищается; пустое/пробельное не шлётся.
39. `components/Chat.svelte`: сворачивание (кнопка-иконка) + бейдж `unread`; разворот зовёт `setChatOpen(true)`.
40. Проводка: `App` маршрутизирует `CHAT`→`applyChat` (+тост при закрытой панели), `CHAT_HISTORY`→`applyChatHistory`, `resetChat` в `endSession`; `NotificationKind += 'chat'`; `Game` рендерит `<Chat>`.
45. (session-добавка) `components/Chat.svelte`: автоскролл ленты к последнему сообщению — на новое сообщение и при развороте панели (`$effect` + `bind:this`, `scrollTop = scrollHeight`); скроллится только лента, не страница.
46. (session-добавка) `components/Board.svelte`: зона выкида переехала из-под доски — десктоп (`@media min-width:1024px`) слева от доски, мобила оверлеем на доске по центру. Drop-цель для drag и кнопка для клика — в одном `.bear-off-zone` поверх `.board-area` (`position:relative` + абсолютный оверлей). Логика/`data-testid` не тронуты — чистый CSS/layout (тесты выкида #20b/#43 — страховка).
47. (session-добавка) `components/Board.svelte`: метка-подсказка над каждой подсвеченной целью выбранной/захваченной шашки — бейдж (`move-hint-{to}`) с цифрой `pip` (значение кубика = дальность хода) из `legalMoves`. `$derived moveHints` (фильтр по `activeFrom`, выкид `to == 0` исключён), позиция `hintPos` у острия треугольника. Бейдж рисуется после шашек (поверх) + `pointer-events:none` (клик/drop проходит к цели). Работает и в клик-, и в drag-режиме через общий `activeFrom`.
48. (session-добавка) `protocol/messages.ts` + `stores/game.svelte.ts`: тип `ReachMove{from,path,pips}` и опц. поле `reach` в `LEGAL_MOVES`; стор хранит `reach` (дефолт `[]`, обнуляется при `LEGAL_MOVES` без поля).
49. (session-добавка) `components/Board.svelte`: подсветка ВСЕХ достижимых целей выбранной/захваченной шашки из `reach` — цвет по числу кубиков (классы `tier-N` на цели-треугольнике и бейдже; 1 зел/2 син/3 янтарь/4 фиолет), бейдж = суммарная дистанция (`dist`). Мини-легенда (`dice-legend`) при наличии составной цели. Фолбэк на одиночные `legalMoves`, если `reach` пуст.
50. (session-добавка) `components/Board.svelte`: выбор/дроп составной цели проигрывает `path` как цепочку `MOVE` (`commitMoveChain`) — протокол атомарный (по кубику за `MOVE`), сервер применяет по порядку; одиночная цель — один `MOVE` (как раньше).

52. (session-добавка) Счётчик оставшихся к сбросу шашек. Сквозная (бэк+фронт): `STATE` получил поле `allHome {white,black}` = доменный `AllInHome` (`internal/protocol/messages.go` + `game.StateMessage`); TS-зеркало в `messages.ts`, поле `allHome` в gameStore. Новый `components/BearOffCounter.svelte`: для цвета с `allHome[c]` показывает «Белые|Чёрные (15−borneOff)/15», плашка скрыта вне фазы сброса; проводка в `Game.svelte` (боковая колонка рядом с кубиками), для обоих игроков. Клиент не считает «все дома» сам — берёт готовый флаг из STATE (как `borneOff`/`isFirstMove`).

### Этап 15 — drag&drop шашек

Перетаскивание **поверх** клик-режима (клики остаются — § 4). Технология — **Pointer Events**: единый API для мыши и тача, работает с SVG. Без `setPointerCapture` (jsdom его не реализует, и захват сломал бы определение цели по элементу). Цель сброса определяется **DOM-элементом под `pointerup`**, а не координатами курсора — поэтому drop-логика тестируема в jsdom, где у события нет `clientX`. Координаты нужны лишь для «летящей» шашки-призрака — это визуал, его позицию не тестируем (§ «Что НЕ тестировать»). HTML5 native DnD отвергнут (плохо дружит с SVG: нет drag-image, слабый тач), сторонняя библиотека — против принципа «без UI-библиотек в MVP» (§ 1).

Группируем по парам: #41–#42 (ядро point→point), #43–#44 (выкид + призрак/отмена).

41. `Board`: `pointerdown` на своей шашке/пункте → старт перетаскивания (`dragFrom`), подсветка источника и легальных целей (как в клик-режиме). Чужая/пустая шашка или `myColor == null` → no-op.
42. `Board`: `pointerup` на легальной цели при активном drag → `onMove(from, to)` и сброс; на нелегальной точке или вне доски → отмена без хода.
43. `Board`: drop-зона выкида — `pointerup` на трее bear-off при легальном `to == 0` → `onMove(from, 0)`. Кнопка «Сбросить шашку» остаётся для клик-режима.
44. `Board`: «летящая» шашка следует за курсором (`pointermove`, перевод screen→viewBox через `getScreenCTM`, no-op при `null`); `pointercancel` → отмена. Тестируем факт призрака и отмены, не координаты.

## 8. Прогресс

Раздел заполняется по ходу работы. Маркируем закрытые этапы галочкой.

- ✅ Этап 0 — каркас (`/web/` собран на Vite 6 + Svelte 5 + TS 5 + Vitest 3; smoke-тест `App.test.ts` зелёный, `npm run check`/`lint`/`build` без ошибок).
- ✅ Этап 1 — типы протокола (`web/src/protocol/messages.ts`: discriminated unions `ClientMessage` и `ServerMessage`, `serializeClientMessage` + `parseServerMessage`; 17 тестов).
- ✅ Этап 2 — WSClient (`web/src/transport/ws.ts`: connect/send/onMessage + реконнект с backoff 1с→30с + auto-JOIN после каждого open; 13 тестов; общий `MockWebSocket` в `web/tests/`).
- ✅ Этап 3 — gameStore (`web/src/stores/game.svelte.ts`: $state-объект `gameState`, `applyServerMessage` reducer-style для STATE/LEGAL_MOVES/GAME_OVER/ERROR; 7 тестов).
- ✅ Этап 4 — геометрия и доска (`web/src/lib/geometry.ts`: `pointAnchor`/`checkerAt` + ViewBox-константы, 15 тестов; `web/src/components/Board.svelte`: SVG c 24 пунктами и шашками из `board: number[]`, 6 тестов).
- ✅ Этап 5 — кубики (`web/src/components/Dice.svelte`: рендер двух кубиков и `remaining`; `dice: Dice | null` → пусто при null; 7 тестов).
- ✅ Этап 6 — действия игрока (`ActionBar.svelte` с 4 кнопками ROLL_FOR_FIRST/ROLL/END_TURN/RESIGN — 9 тестов; `Board.svelte` расширен опциональными legalMoves/myColor/onMove + локальный selectedFrom-стейт + подсветка `selected`/`legal-target` — 10 новых тестов клик-режима).
- ✅ Этап 7 — конец игры (`GameOver.svelte` модалка с локализацией Белые/Чёрные + Оин/Марс/Кокс; «Вы победили»/«Вы проиграли» при наличии myColor; кнопка «Новая игра» → onNewGame; 10 тестов).
- ✅ Этап 8 — Connect и реконнект (`Connect.svelte` форма+localStorage; `App.svelte` — маршрутизация Connect↔Game, проводка WSClient, авто-подключение из localStorage, `ERROR{UNAUTHORIZED}` → чистка кредов и возврат в Connect; стор `connection.svelte.ts` + `WSClient.onStateChange` (connecting/connected/reconnecting) + `ActionBar.disabled` → `reconnecting` блокирует ActionBar; весь набор — 137 тестов).
- ✅ Этап 9 — invite-флоу (`lib/api.ts` createGame/joinGame; `Connect` «Создать игру» + ссылка-приглашение `?game=<id>` + вход по приглашению; `App` читает `?game=` из URL; Vite-прокси `/api`; ручной ввод — видимый фоллбэк. Бэкенд: SPEC #38–#41. Плюс session-добавка #27 — кнопка «Сменить игру». 147 тестов фронта).
- ✅ Этап 10 — личная ссылка для возврата (`App.svelte` читает `?game=<id>&token=<token>` → реконнект с приоритетом над сохранёнными кредами, сохраняет в localStorage и вычищает `token` из адресной строки через `history.replaceState`; `Game.svelte` показывает игроку его reconnect-ссылку с кнопкой копирования). Решает возврат в игру с другого устройства/браузера, где localStorage пуст. 167 тестов фронта.
- ✅ Этап 11 — уведомления о событиях (`stores/notifications.svelte.ts` — стек тостов push/dismiss/reset, push возвращает id; `lib/sound.ts` — `playRollCue()` коротким тоном через Web Audio с DI-конструктором, no-op без поддержки/при ошибке; `components/Toast.svelte` + `Notifications.svelte` — тосты с `role=status`/`aria-live`, ручным закрытием и авто-скрытием по таймеру, фиксированный оверлей в App; проводка: `App` ловит `OPPONENT_JOINED` → «Соперник присоединился», `Game` через `$effect`-детектор перехода «ожидается мой бросок» → «Твой бросок» + звук — и на обычном ходу, и на розыгрыше первого хода; флаг `started` в gameStore отсекает initial-снапшот между JOINED и первым STATE, чтобы не было ложного звона при возврате в игру). 189 тестов фронта (+22).
- ✅ Фикс invite vs saved creds — `App.svelte` авто-подключается по сохранённым кредам только когда нет приглашения или приглашение в свою же игру (`saved.gameId == inviteGameId`); приглашение в другую игру ведёт через Connect (свой токен), иначе игрок занимал чужой слот/уходил не в ту игру.
- ✅ Этап 13 (фронт-часть) — плашка «Переподключение…» в `Game.svelte` при `connection.state == 'reconnecting'` (`role=status`/`aria-live`): пауза при рестарте сервера читается как восстановление, а не зависание; авто-реконнект WSClient уже был. 193 теста фронта.
- ✅ Этап 14 — чат партии (#35–#40): типы `CHAT`/`CHAT_HISTORY` в `messages.ts`; стор `chat.svelte.ts` (`messages`/`unread`/`open`); `Chat.svelte` — сворачиваемая панель в углу (лента свои/чужие по `sender` vs `myColor`, ввод + Enter, бейдж непрочитанных, на узком экране — оверлей); проводка: `App` (`CHAT`→`applyChat` + тост чужого при свёрнутой панели, `CHAT_HISTORY`→`applyChatHistory`, `resetChat` в `endSession`), `NotificationKind += 'chat'`, `Game` рендерит `<Chat>`. Сервер — источник правды: без оптимистичного эха. 219 тестов фронта (+26).
- ✅ Session-добавка #45 — автоскролл чата: `Chat.svelte` через `$effect` + `bind:this` прокручивает ленту к низу (`scrollTop = scrollHeight`) при новом сообщении и при развороте панели — свежее сообщение видно сразу. Скроллится только лента (overflow-y), не страница. Зависимости эффекта: `chat.messages.length` и сам `messagesEl` ($state, чтобы перезапуск ловил пересоздание `<ul>` при развороте). Первый `$effect` в проекте. +2 теста (249 тестов фронта).
- ✅ Session-добавка #46 — зона выкида не под доской, а: десктоп (`@media min-width:1024px`) — слева от доски, мобила — оверлеем на доске по центру. В `Board.svelte` `<svg>` обёрнут в `.board-area` (`position:relative`), drop-цель (drag) и кнопка (клик) объединены в `.bear-off-zone` (абсолютный оверлей, позиция через CSS-брейкпоинт). Чистый CSS/layout: логика, `data-testid` и drag-детект сброса не менялись; страховка — существующие тесты выкида (#20b клик, #43 drag), 249 тестов фронта без изменений. Первый `@media` в проекте.
- ✅ Этап 15 — drag&drop шашек (#41–#44): `Board.svelte` на Pointer Events поверх клик-режима (клики остаются). `pointerdown` на своей шашке → `dragFrom` + подсветка целей (общий `activeFrom = dragFrom ?? selectedFrom`); `pointerup` на легальной цели → `commitMove(from, to)`, на нелегальной/голом SVG/`pointercancel` → отмена. Выкид перетаскиванием — drop-зона `bear-off-drop` (`to == 0`); кнопка «Сбросить шашку» оставлена для клик-режима. «Летящий» призрак следует за курсором (`pointermove` + `getScreenCTM`, no-op в jsdom). Без `setPointerCapture`: цель определяется элементом под `pointerup`, не координатами — тестируемо. 233 теста фронта (+14). Реализовано без правок бэкенда/протокола (шлём тот же `MOVE`).
- ✅ Session-добавка #47 — метки-подсказки pip над целями: над каждой подсвеченной целью выбранной/захваченной шашки — бейдж `move-hint-{to}` (зелёный кружок с белой цифрой) со значением кубика `pip` из `legalMoves` (на какую дальность идёт ход). `$derived moveHints` фильтрует `legalMoves` по общему `activeFrom` (работает и для клика, и для drag) и исключает выкид (`to == 0` — это отдельная зона, не треугольник); `hintPos` ставит бейдж у острия треугольника. Бейдж рендерится после шашек (z-order → поверх) с непрозрачным фоном — цифра читается даже на занятой своими цели; `pointer-events:none` пропускает клик/drop к пункту-цели под меткой. Только фронт, без правок протокола. 254 теста фронта (+5).
- ✅ Session-добавка #48–#50 — подсказки прогресса хода: сервер шлёт `reach` в `LEGAL_MOVES` (составные ходы одной шашкой, бэкенд SPEC #50–#51); `Board` подсвечивает все достижимые цели цветом по числу кубиков (`tier-N`) с суммарной дистанцией на бейдже и мини-легендой; выбор/дроп составной цели шлёт цепочку `MOVE` по `path`. 265 тестов фронта (+12).
- ✅ Session-добавка #51 — индикатор ожидания соперника в розыгрыше первого хода: `ROLL_FOR_FIRST` — «сигнал готовности», сервер молчит, пока не пришли оба (FIRST_ROLL шлётся только после второго), поэтому после клика «ничего не происходило». `ActionBar` держит локальный `$state`-флаг `sentRollForFirst` (ставится в `act` при клике, гейтит показ вместе с `inFirstRollPhase`): кнопка `action-roll-for-first` сменяется индикатором `waiting-first-roll` («Ожидание другого игрока…» + спиннер, `role=status`). Флаг не сбрасывается вручную — в партии `rolledForFirst` монотонно `false→true`, а новая партия размонтирует `ActionBar` (App: `client→null`); `disabled` (реконнект) не даёт ложно «ждать», т.к. бросок не ушёл. Только фронт, без правок протокола. 270 тестов фронта (+5).
- ✅ Session-добавка #52 — счётчик оставшихся к сбросу шашек. Сквозная фича: сервер в `STATE` шлёт `allHome {white,black}` = значение доменного `AllInHome` (`protocol/messages.go` — `AllHomePayload`; `game.StateMessage` заполняет из `AllInHome(board, White/Black)`). Фронт зеркалит поле в `messages.ts` (обязательное, как `borneOff`) и хранит в gameStore (initial `{false,false}`, apply в STATE). `components/BearOffCounter.svelte` рисует «осталось/15» = `(15−borneOff[c])/15` для каждого цвета, у кого `allHome[c]` (все шашки в доме), и не рендерит ничего вне фазы сброса; проводка в `Game.svelte` (боковая колонка рядом с кубиками). Показывается для обоих игроков. Клиент не дублирует правило «все дома» — берёт готовый флаг из STATE. Бэк +1 тест (`state_message`), фронт +8 тестов: стор +2, `BearOffCounter` +4, проводка `Game` +2 (STATE-парсинг-тест расширен полем `allHome`). 278 тестов фронта.

## 9. Открытые вопросы

1. ~~**STATE неполный на бекенде.**~~ ✅ **Закрыто.** `ServerMessage` получил `BorneOffPayload` и `IsFirstMovePayload`; `game.StateMessage` заполняет оба поля из доменного состояния. Этап 3 фронта (gameStore) разблокирован.
2. **`ROOM_FULL` не задокументирован в `nardy-protocol`.** Бекенд реально шлёт его (`internal/transport/ws/handler.go:76` при отказе третьему клиенту), но в скилле этот код не упомянут. Поправить отдельным docs-циклом: добавить `ROOM_FULL` в таблицу кодов ошибок в `nardy-protocol/SKILL.md`, после — расширить `ErrorCode` в `web/src/protocol/messages.ts`.
2. ~~**REST для invite-флоу.**~~ ✅ **Закрыто.** `POST /api/games` (создать) и `POST /api/games/{id}/join` (вход по ссылке) генерят `gameId`/`token` на сервере (crypto/rand); токен — только в теле ответа, не в URL. Connect получил «Создать игру» + ссылку-приглашение и вход по `?game=`; ручной ввод оставлен фоллбэком. Бэкенд SPEC #38–#41, фронт #28–#30.
3. ~~**Drag&drop.**~~ ✅ **Закрыто (этап 15).** **Pointer Events** поверх клик-режима. HTML5 native DnD и сторонняя библиотека отвергнуты (SVG-несовместимость / принцип «без UI-библиотек»). Drop-цель определяется DOM-элементом под `pointerup` — тестируемо в jsdom без `PointerEvent`/координат.
4. **i18n.** Пока — только русский, без обёрток. Когда добавим английский — ввести `t()` и каталоги.
5. **Стилизация.** Старт — ванильный CSS. При росте UI пересмотреть.
