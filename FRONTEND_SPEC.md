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
    App.svelte           — root, выбирает Connect или Game по connectionStore
    protocol/
      messages.ts        — TS-зеркало internal/protocol/messages.go
    transport/
      ws.ts              — WSClient: connect/send/onMessage/реконнект
    stores/
      game.ts            — board, turn, dice, borneOff, status, isFirstMove, legalMoves, gameOver
      connection.ts      — state: idle | connecting | connected | reconnecting | error
    screens/
      Connect.svelte     — форма ввода gameId + token, persistence в localStorage
      Game.svelte        — Board + Dice + ActionBar + GameOver overlay
    components/
      Board.svelte       — SVG-доска
      Checker.svelte     — одна шашка
      Dice.svelte        — два кубика + оставшиеся пипсы
      ActionBar.svelte   — контекстные кнопки ROLL_FOR_FIRST/ROLL/END_TURN/RESIGN
      GameOver.svelte    — модалка результата
    lib/
      geometry.ts        — координаты SVG для пункта i и j-й шашки на пункте
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

Drag&drop — после клик-режима, отдельным циклом. MVP — клики.

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

## 8. Прогресс

Раздел заполняется по ходу работы. Маркируем закрытые этапы галочкой.

- ⬜ Этап 0 — каркас
- ⬜ Этап 1 — типы протокола
- ⬜ Этап 2 — WSClient
- ⬜ Этап 3 — gameStore
- ⬜ Этап 4 — геометрия и доска
- ⬜ Этап 5 — кубики
- ⬜ Этап 6 — действия игрока
- ⬜ Этап 7 — конец игры
- ⬜ Этап 8 — Connect и реконнект

## 9. Открытые вопросы

1. ~~**STATE неполный на бекенде.**~~ ✅ **Закрыто.** `ServerMessage` получил `BorneOffPayload` и `IsFirstMovePayload`; `game.StateMessage` заполняет оба поля из доменного состояния. Этап 3 фронта (gameStore) разблокирован.
2. **REST для invite-флоу.** Сейчас `gameId` и `token` вводятся вручную в Connect. Когда появится REST (создание игры, генерация приглашения) — Connect заменится на «Создать игру» / «Войти по ссылке».
3. **Drag&drop.** В MVP не входит — только клики. Решить, делать ли HTML5 drag, pointer events или библиотеку, когда возьмём цикл.
4. **i18n.** Пока — только русский, без обёрток. Когда добавим английский — ввести `t()` и каталоги.
5. **Стилизация.** Старт — ванильный CSS. При росте UI пересмотреть.
