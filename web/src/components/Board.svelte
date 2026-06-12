<script lang="ts">
  import type { Color, Move } from '../protocol/messages'

  import {
    BAR_WIDTH,
    BAR_X,
    CHECKER_RADIUS,
    COLUMN_WIDTH,
    VIEWBOX_HEIGHT,
    VIEWBOX_WIDTH,
    checkerAt,
    pointAnchor,
  } from '../lib/geometry'

  interface Props {
    board: number[]
    legalMoves?: Move[]
    myColor?: Color | null
    onMove?: (from: number, to: number) => void
  }

  let {
    board,
    legalMoves = [],
    myColor = null,
    onMove = () => {},
  }: Props = $props()

  const POINTS: number[] = Array.from({ length: 24 }, (_, i) => i + 1)
  const TRIANGLE_HEIGHT = 320
  const TRIANGLE_HALF_BASE = COLUMN_WIDTH * 0.4

  let selectedFrom = $state<number | null>(null)
  // Источник перетаскивания (#41). Отдельно от selectedFrom: drag и клик-режим
  // не должны мешать друг другу (тап = pointerdown+up без движения порождает
  // ещё и click). Подсветку ведём по activeFrom — что активно сейчас.
  // Перетаскивание активируется только после заметного движения (порог) — до
  // этого pointerdown лишь запоминает кандидата. Тап без движения остаётся
  // кликом (клик-режим), поэтому клики работают как раньше. #41
  let dragFrom = $state<number | null>(null)
  let pendingDrag: { point: number; x: number; y: number } | null = null
  const DRAG_THRESHOLD = 6 // px в экранных координатах
  // Позиция «летящей» шашки-призрака в координатах viewBox (#44).
  let dragX = $state(0)
  let dragY = $state(0)
  // Ссылка на сам <svg>. Через event.currentTarget нельзя: Svelte 5 делегирует
  // pointermove, и там currentTarget — не этот узел (баг найден вживую).
  let boardEl: SVGSVGElement | undefined
  const activeFrom = $derived(dragFrom ?? selectedFrom)

  // Перспектива: белый видит доску повёрнутой на 180°, чтобы его шашки были
  // слева (у чёрного/наблюдателя — как есть). #1.
  const flipped = $derived(myColor === 'white')

  function trianglePoints(point: number, flip: boolean): string {
    const a = pointAnchor(point, flip)
    const tipY = a.direction === 'up' ? a.y - TRIANGLE_HEIGHT : a.y + TRIANGLE_HEIGHT
    return `${a.x - TRIANGLE_HALF_BASE},${a.y} ${a.x + TRIANGLE_HALF_BASE},${a.y} ${a.x},${tipY}`
  }

  function checkerCount(point: number): number {
    return Math.abs(board[point - 1] ?? 0)
  }

  function checkerColor(point: number): 'white' | 'black' {
    return (board[point - 1] ?? 0) > 0 ? 'white' : 'black'
  }

  function isMyChecker(point: number): boolean {
    if (myColor === null) return false
    const v = board[point - 1] ?? 0
    return myColor === 'white' ? v > 0 : v < 0
  }

  function isLegalTarget(point: number): boolean {
    if (activeFrom === null) return false
    return legalMoves.some((m) => m.from === activeFrom && m.to === point)
  }

  function handlePointClick(point: number): void {
    if (myColor === null) return
    if (selectedFrom !== null && isLegalTarget(point)) {
      commitMove(selectedFrom, point)
      return
    }
    if (isMyChecker(point)) {
      selectedFrom = selectedFrom === point ? null : point
      return
    }
    // клик по пустому/чужому пункту → снять выделение (отмена выбора)
    selectedFrom = null
  }

  // Drag&drop (#41–#42). Без setPointerCapture (jsdom его не реализует) — цель
  // сброса определяем по элементу под pointerup, а не по координатам курсора.
  // pointerdown лишь запоминает кандидата; drag стартует в moveDrag по порогу.
  function startDrag(point: number, event: PointerEvent): void {
    if (myColor === null || !isMyChecker(point)) {
      pendingDrag = null
      return
    }
    pendingDrag = { point, x: event.clientX, y: event.clientY }
  }

  // Призрак следует за курсором. Drag активируется здесь — после движения за
  // порог (тап без движения остаётся кликом). screen→viewBox через CTM; в jsdom
  // getScreenCTM/координат нет → порог считаем пройденным, позицию не двигаем.
  function moveDrag(event: PointerEvent): void {
    if (dragFrom === null) {
      if (pendingDrag === null) return
      const hasCoords =
        typeof event.clientX === 'number' && typeof pendingDrag.x === 'number'
      if (hasCoords) {
        const moved = Math.hypot(event.clientX - pendingDrag.x, event.clientY - pendingDrag.y)
        if (moved < DRAG_THRESHOLD) return
      }
      dragFrom = pendingDrag.point
    }
    const ctm = boardEl?.getScreenCTM?.()
    if (!ctm) return
    const local = new DOMPoint(event.clientX, event.clientY).matrixTransform(ctm.inverse())
    dragX = local.x
    dragY = local.y
  }

  function dropOn(point: number): void {
    pendingDrag = null
    if (dragFrom === null) return // тап без движения → разберётся клик-режим
    if (isLegalTarget(point)) commitMove(dragFrom, point)
    else cancelDrag()
  }

  // Отмена перетаскивания (pointerup вне цели / pointercancel). selectedFrom НЕ
  // трогаем — иначе тап по выделенной шашке снимал бы выбор раньше click и
  // ломал отмену повторным кликом.
  function cancelDrag(): void {
    pendingDrag = null
    dragFrom = null
  }

  // Завершение хода из любого режима (клик или drag): сброс выбора и
  // перетаскивания + onMove. Источник правды — сервер, ждём STATE.
  function commitMove(from: number, to: number): void {
    pendingDrag = null
    dragFrom = null
    selectedFrom = null
    onMove(from, to)
  }

  // Выкид: у пункта 0 на доске нет, поэтому отдельный контрол. Появляется,
  // когда у выделенной шашки есть легальный выкид (to === 0).
  const bearOffAvailable = $derived(
    selectedFrom !== null && legalMoves.some((m) => m.from === selectedFrom && m.to === 0),
  )

  function handleBearOff(): void {
    if (selectedFrom !== null) commitMove(selectedFrom, 0)
  }

  // Выкид перетаскиванием (#43): drop-зона видна, пока тащим шашку с легальным
  // выкидом; on pointerup на ней → onMove(from, 0).
  const dragBearOffAvailable = $derived(
    dragFrom !== null && legalMoves.some((m) => m.from === dragFrom && m.to === 0),
  )

  function dropBearOff(): void {
    if (dragFrom !== null) commitMove(dragFrom, 0)
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<svg
  bind:this={boardEl}
  class="board"
  viewBox={`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`}
  xmlns="http://www.w3.org/2000/svg"
  onpointermove={moveDrag}
  onpointerup={cancelDrag}
  onpointercancel={cancelDrag}
>
  <rect class="bar" x={BAR_X} y="0" width={BAR_WIDTH} height={VIEWBOX_HEIGHT} />
  {#each POINTS as point (point)}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <polygon
      class="point {point % 2 === 0 ? 'even' : 'odd'}"
      class:selected={selectedFrom === point || dragFrom === point}
      class:legal-target={isLegalTarget(point)}
      data-testid="point-{point}"
      points={trianglePoints(point, flipped)}
      onclick={() => handlePointClick(point)}
      onpointerdown={(e) => startDrag(point, e)}
      onpointerup={() => dropOn(point)}
    />
  {/each}

  {#each POINTS as point (point)}
    {#each Array.from({ length: checkerCount(point) }, (_, j) => j) as j (j)}
      {@const pos = checkerAt(point, j, checkerCount(point), flipped)}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <circle
        class="checker {checkerColor(point)}"
        data-testid="checker-{point}-{j}"
        cx={pos.cx}
        cy={pos.cy}
        r={pos.r}
        onclick={() => handlePointClick(point)}
        onpointerdown={(e) => startDrag(point, e)}
        onpointerup={() => dropOn(point)}
      />
    {/each}
  {/each}

  {#if dragFrom !== null}
    <!-- Летящая шашка-призрак (#44). pointer-events=none — чтобы не перехватывать
         pointerup у цели под курсором. -->
    <circle
      class="checker {myColor === 'white' ? 'white' : 'black'} drag-ghost"
      data-testid="drag-ghost"
      cx={dragX}
      cy={dragY}
      r={CHECKER_RADIUS}
      pointer-events="none"
    />
  {/if}
</svg>

{#if dragBearOffAvailable}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="bear-off-drop" data-testid="bear-off-drop" onpointerup={dropBearOff}>
    Брось сюда, чтобы сбросить →
  </div>
{/if}

{#if bearOffAvailable}
  <button type="button" class="bear-off" data-testid="bear-off" onclick={handleBearOff}>
    Сбросить шашку →
  </button>
{/if}

<style>
  .board {
    display: block;
    width: 100%;
    height: auto;
    max-width: 760px;
    background: #e7c79b;
    /* Тач: жесты на доске — это drag, а не скролл/зум/выделение текста.
       Без этого на телефоне touchmove перехватывается браузером (pointercancel)
       и шашки не перетаскиваются, а текст «выделяется». */
    touch-action: none;
    user-select: none;
    -webkit-user-select: none;
    -webkit-touch-callout: none;
    /* Деревянная рамка/окантовка под реальную доску (#4). */
    border: 14px solid #6b4423;
    border-radius: 10px;
    box-shadow:
      0 0 0 2px #3b2410,
      0 6px 18px rgba(0, 0, 0, 0.45);
  }
  .bar {
    fill: #5a3a1e;
  }
  .point.even {
    fill: #c19a6b;
  }
  .point.odd {
    fill: #8b6840;
  }
  .point.selected {
    fill: #ffd54f;
  }
  .point.legal-target {
    fill: #aed581;
    stroke: #2e7d32;
    stroke-width: 3;
  }
  .checker.white {
    fill: #f4ece1;
    stroke: #2a1e10;
    stroke-width: 2;
  }
  .checker.black {
    fill: #2a1e10;
    stroke: #f4ece1;
    stroke-width: 2;
  }
  .bear-off {
    display: block;
    margin: 0.5rem auto 0;
    background: #aed581;
    color: #1b5e20;
    border: 2px solid #2e7d32;
    border-radius: 6px;
    padding: 0.5rem 1rem;
    font-size: 15px;
    font-weight: 700;
    cursor: pointer;
  }
  .bear-off:hover {
    background: #9ccc65;
  }
  .drag-ghost {
    opacity: 0.75;
  }
  .bear-off-drop {
    display: block;
    margin: 0.5rem auto 0;
    max-width: 760px;
    background: #aed581;
    color: #1b5e20;
    border: 2px dashed #2e7d32;
    border-radius: 6px;
    padding: 0.75rem 1rem;
    font-size: 15px;
    font-weight: 700;
    text-align: center;
  }
</style>
