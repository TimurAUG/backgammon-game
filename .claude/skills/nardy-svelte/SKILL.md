---
name: nardy-svelte
description: Конвенции Svelte 5 для проекта backgammon-game — runes ($state/$derived/$effect/$props), почему НЕ legacy stores из svelte/store, тестирование компонентов через @testing-library/svelte. Используй ПЕРЕД написанием или изменением любого .svelte-файла в /web/ или сторов в /web/src/stores/. Скилл нужен потому что Svelte 5 свежий: модели часто скатываются на API Svelte 4 (writable/readable stores, export let), который в этом проекте НЕ используется.
---

# Svelte 5 conventions — backgammon-game

Проект использует **Svelte 5 + runes**. Любая попытка применить API Svelte 4 (legacy stores, `export let`, `$:` reactive statements) — отвергается на ревью.

Связанные скиллы: `nardy-frontend` — структура `/web/` и TDD-дисциплина фронта.

## Runes (источник правды)

| Руна | Когда |
|---|---|
| `$state(initial)` | Любое **изменяемое** состояние компонента или модуля. |
| `$derived(expr)` | Производное значение от других `$state`. **Никаких** побочных эффектов. |
| `$effect(() => ...)` | Побочные эффекты (подписки, DOM-доступ). Очистка — `return () => ...`. |
| `$props()` | Объявление props компонента. Заменяет `export let`. |
| `$bindable()` | Двусторонний bind props (использовать редко). |

## Сторы — через runes, не через `svelte/store`

В проекте `/web/src/stores/*.ts` — это **модули с экспортированными `$state`-объектами**, не `writable()`/`readable()`.

```ts
// stores/game.ts
import type { Board, Color, Dice, ... } from '../protocol/messages'

export const gameState = $state({
  board: [] as Board,
  turn: 'white' as Color,
  dice: null as Dice | null,
  borneOff: { white: 0, black: 0 },
  status: 'waitingForRoll' as 'waitingForRoll' | 'waitingForMove' | 'finished',
  isFirstMove: { white: true, black: true },
  legalMoves: [] as Move[],
  gameOver: null as { winner: Color; kind: 'oin' | 'mars' | 'koks' } | null,
})

export function applyServerMessage(msg: ServerMessage) {
  if (msg.type === 'STATE') {
    gameState.board = msg.board
    gameState.turn = msg.turn
    // ...
  }
  // ...
}
```

В компоненте — просто читаем поля:

```svelte
<script lang="ts">
  import { gameState } from '../stores/game'
</script>

<p>Ход: {gameState.turn}</p>
```

Никаких `$gameStore` со знаком доллара — это синтаксис Svelte 4 для `writable()`. В runes — обычные поля объекта.

## Props — только через `$props()`

```svelte
<script lang="ts">
  let { from, to, pip }: { from: number; to: number; pip: number } = $props()
</script>
```

Не `export let from`. Не TypeScript-интерфейс отдельным узлом — destructuring с inline-типом.

## Реактивность с WebSocket

WSClient (`/web/src/transport/ws.ts`) — обычный TS-класс, **без** runes (модуль может не иметь компонентного контекста). При получении сообщения он зовёт `applyServerMessage(msg)` из стора. Стор внутри меняет `$state`-объект → Svelte автоматически перерендеривает зависимые компоненты.

```ts
// transport/ws.ts (фрагмент)
import { applyServerMessage } from '../stores/game'

socket.onmessage = (ev) => {
  const msg = parseServerMessage(ev.data)
  applyServerMessage(msg)
}
```

`$effect` в компонентах — только для DOM-побочек (например, фокус, scroll), не для подписки на сторы. Подписка автоматическая через чтение полей.

## События: делегирование и `event.currentTarget`

Svelte 5 **делегирует** большинство событий (`pointerdown`/`pointermove`/`pointerup`, `click`, `mousedown` и др.): вешает один слушатель на корень монтирования вместо листенера на каждом элементе. Следствие — в обработчике **`event.currentTarget` указывает не на тот узел**, где написан `onpointermove=...`, а на узел-делегат. Поэтому `event.currentTarget.getScreenCTM()` (и любой доступ к самому элементу через `currentTarget`) молча возвращает не то.

Нужен сам DOM-узел в обработчике — бери ссылку через `bind:this`, не через `event.currentTarget`:

```svelte
<script lang="ts">
  let boardEl: SVGSVGElement | undefined

  function onMove(e: PointerEvent) {
    // НЕ e.currentTarget — при делегировании это узел-делегат
    const ctm = boardEl?.getScreenCTM?.()
    // ...
  }
</script>

<svg bind:this={boardEl} onpointermove={onMove}>...</svg>
```

Поймано вживую на drag&drop (`Board.svelte`, этап 15): призрак не следовал за курсором, т.к. CTM брался из `event.currentTarget`. **jsdom этот класс багов не ловит** (нет `getScreenCTM`, у синтетического события нет координат) — проверять только в реальном браузере.

## Тестирование компонентов

- Библиотека: **`@testing-library/svelte`**, рантайм Vitest + jsdom.
- Файл рядом: `Board.test.ts` рядом с `Board.svelte`. Суффикс `.svelte.ts` — только для runes-исходников (например `stores/game.svelte.ts`), не для тестов.
- Импорт компонента и `render`:

```ts
import { render, screen } from '@testing-library/svelte'
import Board from './Board.svelte'

test('Board_rendersInitialPosition_15CheckersOnPoint24', () => {
  render(Board, { props: { board: initialBoardFixture() } })
  expect(screen.getAllByTestId('checker-24')).toHaveLength(15)
})
```

- **Семантические запросы** (`getByRole`, `getByLabelText`) приоритетнее `getByTestId`. `data-testid` — только когда семантического атрибута нет (SVG-элементы, например).
- События — через `fireEvent` или `userEvent`, не вручную через `dispatchEvent`.
- Сторы в тестах — **сбрасывать** в setup-блоке (`beforeEach`), не доверять порядку тестов.

## Антипаттерны

- `writable()`, `readable()`, `derived()` из `svelte/store` — **запрещены**. Только runes.
- `$:` reactive statements — Svelte 4 API, **запрещён**.
- `export let foo` — Svelte 4 API, **запрещён**. Только `$props()`.
- `onMount` для подписки на сторы — не нужно, реактивность сама.
- `bind:value` на сложных объектах — предпочитать явный handler.
- `event.currentTarget` для доступа к самому элементу в обработчике делегируемого события (pointer/click/mouse) — указывает на узел-делегат, а не на твой элемент. Бери `bind:this`.
- Inline `<style>` с глобальными селекторами — стилизация только локальная (`<style>` без `:global`), исключение — корневой layout.

## Когда `$effect` уместен

- Манипуляция DOM, недоступная декларативно (фокус, scroll, выделение текста).
- Подписки на внешние источники (resize, keyboard) с обязательной очисткой.

В большинстве случаев `$effect` — сигнал, что что-то делается неоптимально. Перед добавлением `$effect` — спросить себя, нельзя ли сделать через `$derived` или просто реактивное чтение.
