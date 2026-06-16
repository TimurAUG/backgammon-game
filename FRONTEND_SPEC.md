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
      game.svelte.ts          — board/turn/dice/borneOff/status/isFirstMove/legalMoves/gameOver/myColor/firstRoll/started
      connection.svelte.ts    — state: idle | connecting | connected | reconnecting | error
      notifications.svelte.ts — стек тостов (соперник присоединился / «Твой бросок»)
      chat.svelte.ts          — сообщения чата партии + счётчик непрочитанных + open
    screens/
      Connect.svelte     — создать игру / войти по приглашению / ручной ввод; persistence в localStorage
      Game.svelte        — Board + Dice + ActionBar + GameOver + баннеры (первый бросок, переподключение)
    components/
      Board.svelte       — SVG-доска, шашки и клик-режим ходов
      Dice.svelte        — два кубика + оставшиеся пипсы
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

### Game

- **Board** — SVG, 24 пункта (треугольники) + bar + bear-off трей. Рендерится из `gameStore.board`.
- **Dice** — два кубика; рядом — оставшиеся пипсы.
- **ActionBar** — кнопки появляются по контексту:
  - `ROLL_FOR_FIRST` — `status == "waitingForRoll"` И ещё не определён первый ход.
  - `ROLL` — `status == "waitingForRoll"` И мой ход.
  - `END_TURN` — `status == "waitingForMove"` И мой ход.
  - `RESIGN` — всегда, пока игра не закончена.
- Выбор шашки кликом → подсвечиваются клетки `to` из `legalMoves` для этого `from`. Клик по подсвеченной → отправка `MOVE`.

Drag&drop (этап 15) — перетаскивание шашек поверх клик-режима через Pointer Events; клики остаются.

### GameOver

Overlay поверх Game при `status == "finished"`. Показывает `winner` (мой/соперник) и `kind` (Оин/Марс/Кокс). Кнопка «Новая игра» — сбрасывает `gameId` и возвращает в Connect.

## 5. Транспорт и резильентность

- При первом подключении: WSClient открывает сокет, ждёт `open`, шлёт `JOIN`.
- Реконнект — экспоненциальный backoff (1с / 2с / 4с / 8с, потолок 30с). При успешном повторном `open` — снова `JOIN` с теми же `gameId`+`token`, сервер вернёт текущий `STATE`.
- Токен живёт в `localStorage`. При `ERROR { code: "UNAUTHORIZED" }` — чистим и возвращаем в Connect.
- `connectionStore` отражает состояние сокета. UI блокирует ActionBar при `reconnecting`.

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
19. `Board`: клик по своей шашке выделяет её, рендерит подсветку `to` из legalMoves.
20. `Board`: клик по подсвеченной клетке шлёт `MOVE { from, to }`.

### Этап 7 — конец игры

21. `GameOver.svelte` показывается при `gameOver != null` с winner+kind.
22. «Новая игра» очищает `gameId` и возвращает в Connect.

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

35. `protocol/messages.ts`: `ClientMessage += {type:'CHAT', text}`; `ServerMessage += {type:'CHAT', sender, text}` и `{type:'CHAT_HISTORY', chat}`. Сериализация CHAT и парсинг CHAT/CHAT_HISTORY.
36. `stores/chat.svelte.ts`: `$state` `{messages, unread, open}`; `applyChat` (push; `++unread`, если панель закрыта), `applyChatHistory` (replace), `setChatOpen` (`open=true` обнуляет `unread`), `resetChat`.
37. `components/Chat.svelte`: рендер ленты из стора; своё сообщение (`sender == myColor`) и чужое различаются (класс/выравнивание).
38. `components/Chat.svelte`: поле ввода + отправка (Enter/кнопка) → `onAction({type:'CHAT', text})`, поле очищается; пустое/пробельное не шлётся.
39. `components/Chat.svelte`: сворачивание (кнопка-иконка) + бейдж `unread`; разворот зовёт `setChatOpen(true)`.
40. Проводка: `App` маршрутизирует `CHAT`→`applyChat` (+тост при закрытой панели), `CHAT_HISTORY`→`applyChatHistory`, `resetChat` в `endSession`; `NotificationKind += 'chat'`; `Game` рендерит `<Chat>`.
45. (session-добавка) `components/Chat.svelte`: автоскролл ленты к последнему сообщению — на новое сообщение и при развороте панели (`$effect` + `bind:this`, `scrollTop = scrollHeight`); скроллится только лента, не страница.

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
- ✅ Этап 15 — drag&drop шашек (#41–#44): `Board.svelte` на Pointer Events поверх клик-режима (клики остаются). `pointerdown` на своей шашке → `dragFrom` + подсветка целей (общий `activeFrom = dragFrom ?? selectedFrom`); `pointerup` на легальной цели → `commitMove(from, to)`, на нелегальной/голом SVG/`pointercancel` → отмена. Выкид перетаскиванием — drop-зона `bear-off-drop` (`to == 0`); кнопка «Сбросить шашку» оставлена для клик-режима. «Летящий» призрак следует за курсором (`pointermove` + `getScreenCTM`, no-op в jsdom). Без `setPointerCapture`: цель определяется элементом под `pointerup`, не координатами — тестируемо. 233 теста фронта (+14). Реализовано без правок бэкенда/протокола (шлём тот же `MOVE`).

## 9. Открытые вопросы

1. ~~**STATE неполный на бекенде.**~~ ✅ **Закрыто.** `ServerMessage` получил `BorneOffPayload` и `IsFirstMovePayload`; `game.StateMessage` заполняет оба поля из доменного состояния. Этап 3 фронта (gameStore) разблокирован.
2. **`ROOM_FULL` не задокументирован в `nardy-protocol`.** Бекенд реально шлёт его (`internal/transport/ws/handler.go:76` при отказе третьему клиенту), но в скилле этот код не упомянут. Поправить отдельным docs-циклом: добавить `ROOM_FULL` в таблицу кодов ошибок в `nardy-protocol/SKILL.md`, после — расширить `ErrorCode` в `web/src/protocol/messages.ts`.
2. ~~**REST для invite-флоу.**~~ ✅ **Закрыто.** `POST /api/games` (создать) и `POST /api/games/{id}/join` (вход по ссылке) генерят `gameId`/`token` на сервере (crypto/rand); токен — только в теле ответа, не в URL. Connect получил «Создать игру» + ссылку-приглашение и вход по `?game=`; ручной ввод оставлен фоллбэком. Бэкенд SPEC #38–#41, фронт #28–#30.
3. ~~**Drag&drop.**~~ ✅ **Закрыто (этап 15).** **Pointer Events** поверх клик-режима. HTML5 native DnD и сторонняя библиотека отвергнуты (SVG-несовместимость / принцип «без UI-библиотек»). Drop-цель определяется DOM-элементом под `pointerup` — тестируемо в jsdom без `PointerEvent`/координат.
4. **i18n.** Пока — только русский, без обёрток. Когда добавим английский — ввести `t()` и каталоги.
5. **Стилизация.** Старт — ванильный CSS. При росте UI пересмотреть.
