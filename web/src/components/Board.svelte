<script lang="ts">
  import type { Color, Move, ReachMove } from '../protocol/messages'

  import {
    BAR_WIDTH,
    BAR_X,
    CHECKER_DIAMETER,
    COLUMN_WIDTH,
    VIEWBOX_HEIGHT,
    VIEWBOX_WIDTH,
    checkerAt,
    pointAnchor,
  } from '../lib/geometry'

  interface Props {
    board: number[]
    legalMoves?: Move[]
    reach?: ReachMove[]
    myColor?: Color | null
    onMove?: (from: number, to: number) => void
  }

  let {
    board,
    legalMoves = [],
    reach = [],
    myColor = null,
    onMove = () => {},
  }: Props = $props()

  const POINTS: number[] = Array.from({ length: 24 }, (_, i) => i + 1)
  const TRIANGLE_HEIGHT = 320
  const TRIANGLE_HALF_BASE = COLUMN_WIDTH * 0.4
  // Бейдж-подсказка (#47): кружок с цифрой pip у острия треугольника-цели.
  // Радиус чуть меньше шашки; смещение от основания — почти к острию, чтобы
  // метка читалась как «конец стрелки», на которую указывает треугольник.
  const HINT_RADIUS = 22
  const HINT_OFFSET = TRIANGLE_HEIGHT - 40

  let selectedFrom = $state<number | null>(null)
  // Источник перетаскивания (#41). Отдельно от selectedFrom: drag и клик-режим
  // не мешают друг другу. Перетаскивание включается по УДЕРЖАНИЮ: короткий
  // тап/клик не «подхватывает» шашку. pointerdown ставит кандидата и заводит
  // таймер; держишь дольше HOLD_MS — берём шашку, отпустил раньше — это клик.
  let dragFrom = $state<number | null>(null)
  let pendingDrag: { point: number } | null = null
  let holdTimer: ReturnType<typeof setTimeout> | null = null
  const HOLD_MS = 180
  // Позиция и размер призрака в ЭКРАННЫХ координатах: рисуем HTML-оверлеем
  // (position:fixed) поверх страницы, а не SVG-кружком внутри доски — иначе он
  // обрезается границей доски и его не дотащить до зоны выкида под доской.
  let ghostX = $state(0)
  let ghostY = $state(0)
  let ghostSize = $state(0)
  // Ссылка на сам <svg>. Через event.currentTarget нельзя: Svelte 5 делегирует
  // pointermove, и там currentTarget — не этот узел (баг найден вживую).
  let boardEl: SVGSVGElement | undefined
  // id активного указателя — для захвата на доску (touch/боттом-шит).
  let activePointerId: number | null = null
  // Клик, который браузер шлёт после drag, не должен менять выделение.
  let suppressClick = false
  const activeFrom = $derived(dragFrom ?? selectedFrom)

  // Подсказки прогресса хода (#49): для выбранной/захваченной шашки — ВСЕ
  // достижимые цели (reach с сервера), включая составные ходы несколькими
  // кубиками одной шашкой. dice = сколько кубиков тратится (определяет цвет),
  // dist = суммарная дистанция (цифра на бейдже), path = пункты-остановки для
  // проигрывания цепочки MOVE (#50). Если сервер не прислал reach (старый
  // сервер или тест без reach) — фолбэк на одиночные шаги из legalMoves
  // (выкид to === 0 исключаем: это отдельная зона, не треугольник).
  type Hint = { to: number; dice: number; dist: number; path: number[] }
  const moveHints = $derived.by((): Hint[] => {
    if (activeFrom === null) return []
    if (reach.length > 0) {
      return reach.flatMap((r) => {
        if (r.from !== activeFrom) return []
        const to = r.path[r.path.length - 1]
        if (to === undefined) return []
        return [{ to, dice: r.path.length, dist: r.pips.reduce((sum, p) => sum + p, 0), path: r.path }]
      })
    }
    return legalMoves
      .filter((m) => m.from === activeFrom && m.to !== 0)
      .map((m) => ({ to: m.to, dice: 1, dist: m.pip, path: [m.to] }))
  })

  function hintFor(point: number): Hint | undefined {
    return moveHints.find((h) => h.to === point)
  }

  // Легенда цветов (#49): показываем только когда на экране есть составная цель
  // (есть что объяснять), перечисляем лишь присутствующие тиры.
  const showDiceLegend = $derived(moveHints.some((h) => h.dice > 1))
  const legendTiers = $derived([...new Set(moveHints.map((h) => h.dice))].sort((a, b) => a - b))

  function diceLabel(dice: number): string {
    return dice === 1 ? '1 кубик' : `${dice} кубика`
  }

  // Перспектива: белый видит доску повёрнутой на 180°, чтобы его шашки были
  // слева (у чёрного/наблюдателя — как есть). #1.
  const flipped = $derived(myColor === 'white')

  function trianglePoints(point: number, flip: boolean): string {
    const a = pointAnchor(point, flip)
    const tipY = a.direction === 'up' ? a.y - TRIANGLE_HEIGHT : a.y + TRIANGLE_HEIGHT
    return `${a.x - TRIANGLE_HALF_BASE},${a.y} ${a.x + TRIANGLE_HALF_BASE},${a.y} ${a.x},${tipY}`
  }

  // Центр бейджа-подсказки (#47): по оси колонки, у острия треугольника-цели.
  function hintPos(point: number, flip: boolean): { x: number; y: number } {
    const a = pointAnchor(point, flip)
    const y = a.direction === 'up' ? a.y - HINT_OFFSET : a.y + HINT_OFFSET
    return { x: a.x, y }
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

  function handlePointClick(point: number): void {
    // клик после drag браузер шлёт сам — он не должен трогать выделение
    if (suppressClick) {
      suppressClick = false
      return
    }
    if (myColor === null) return
    if (selectedFrom !== null) {
      const hint = hintFor(point)
      if (hint !== undefined) {
        commitMoveChain(selectedFrom, hint.path)
        return
      }
    }
    if (isMyChecker(point)) {
      selectedFrom = selectedFrom === point ? null : point
      return
    }
    // клик по пустому/чужому пункту → снять выделение (отмена выбора)
    selectedFrom = null
  }

  // Drag&drop (#41–#42). pointerdown ставит кандидата и заводит таймер
  // удержания; шашку «берём» только когда таймер сработал. Короткий тап →
  // таймер снят на pointerup → обычный клик.
  function startDrag(point: number, event: PointerEvent): void {
    suppressClick = false
    clearHoldTimer()
    if (myColor === null || !isMyChecker(point)) {
      pendingDrag = null
      return
    }
    pendingDrag = { point }
    activePointerId = event.pointerId
    ghostX = event.clientX
    ghostY = event.clientY
    holdTimer = setTimeout(activateDrag, HOLD_MS)
  }

  // Срабатывает по удержанию: «берём» шашку. Захватываем указатель на доску —
  // события идут к нам (не в скролл/боттом-шит), на touch снимается неявный
  // захват на исходную шашку (drop-цель берём по координатам отпускания).
  function activateDrag(): void {
    holdTimer = null
    if (pendingDrag === null) return
    dragFrom = pendingDrag.point
    const rect = boardEl?.getBoundingClientRect()
    ghostSize =
      rect && rect.width > 0 ? CHECKER_DIAMETER * (rect.width / VIEWBOX_WIDTH) : CHECKER_DIAMETER
    if (activePointerId !== null) boardEl?.setPointerCapture?.(activePointerId)
  }

  function clearHoldTimer(): void {
    if (holdTimer !== null) {
      clearTimeout(holdTimer)
      holdTimer = null
    }
  }

  // Призрак следует за указателем (экранные координаты — для HTML-оверлея).
  function moveDrag(event: PointerEvent): void {
    ghostX = event.clientX
    ghostY = event.clientY
  }

  // Завершение жеста (pointerup/pointercancel). Не дотащили до удержания → клик.
  // Иначе цель определяем по координатам отпускания (elementFromPoint): на touch
  // pointerup приходит на исходную шашку. Нет легальной цели → вернуть на место.
  function endDrag(event: PointerEvent): void {
    clearHoldTimer()
    pendingDrag = null
    if (dragFrom === null) {
      activePointerId = null
      return
    }
    const from = dragFrom
    suppressClick = true
    const point = resolveDropPoint(event)
    if (point === 0) {
      // Выкид: его нет в reach (отдельная зона) — решаем по legalMoves.
      if (legalMoves.some((m) => m.from === from && m.to === 0)) commitMove(from, 0)
      else cancelDrag()
      return
    }
    const hint = point === null ? undefined : hintFor(point)
    if (hint !== undefined) commitMoveChain(from, hint.path)
    else cancelDrag()
  }

  function resolveDropPoint(event: PointerEvent): number | null {
    let el: Element | null = null
    if (typeof event.clientX === 'number' && typeof document.elementFromPoint === 'function') {
      el = document.elementFromPoint(event.clientX, event.clientY)
    }
    el = el ?? (event.target as Element | null)
    if (el === null) return null
    if (el.closest('[data-testid="bear-off-drop"]') !== null) return 0 // выкид
    const node = el.closest('[data-testid]')
    const id = node?.getAttribute('data-testid') ?? ''
    const pointMatch = /^point-(\d+)$/.exec(id)
    if (pointMatch) return Number(pointMatch[1])
    const checkerMatch = /^checker-(\d+)-/.exec(id)
    if (checkerMatch) return Number(checkerMatch[1])
    return null
  }

  function releaseCapture(): void {
    if (activePointerId !== null && boardEl?.hasPointerCapture?.(activePointerId)) {
      boardEl.releasePointerCapture(activePointerId)
    }
    activePointerId = null
  }

  // Отмена перетаскивания (отпустили вне цели / pointercancel) — шашка остаётся
  // на месте. selectedFrom НЕ трогаем (клик-режим отдельно).
  function cancelDrag(): void {
    clearHoldTimer()
    pendingDrag = null
    dragFrom = null
    releaseCapture()
  }

  // Сброс клик- и drag-выделения после хода. Источник правды — сервер, ждём STATE.
  function resetSelection(): void {
    clearHoldTimer()
    pendingDrag = null
    dragFrom = null
    selectedFrom = null
    releaseCapture()
  }

  // Одиночный ход из любого режима (клик или drag).
  function commitMove(from: number, to: number): void {
    resetSelection()
    onMove(from, to)
  }

  // Составной ход (#50): проигрываем цепочку остановок одной шашки как
  // последовательность одиночных MOVE — протокол атомарный (один кубик за MOVE),
  // сервер применяет их по порядку и шлёт STATE после каждого. Ждём STATE, без
  // оптимистичного применения.
  function commitMoveChain(from: number, path: number[]): void {
    resetSelection()
    let cur = from
    for (const next of path) {
      onMove(cur, next)
      cur = next
    }
  }

  // Завершение жеста слушаем на window: на touch с захватом pointerup приходит
  // не на цель, а зона выкида вообще вне <svg> — глобальный слушатель ловит
  // отпускание где угодно. Если отпустили ДО удержания — снимаем таймер (иначе
  // шашка «подхватилась» бы уже после отпускания). Таймер чистим и на unmount.
  $effect(() => {
    const onUp = (e: PointerEvent) => {
      if (dragFrom !== null) {
        endDrag(e)
      } else {
        clearHoldTimer()
        pendingDrag = null
        activePointerId = null
      }
    }
    const onCancel = () => cancelDrag()
    window.addEventListener('pointerup', onUp)
    window.addEventListener('pointercancel', onCancel)
    return () => {
      clearHoldTimer()
      window.removeEventListener('pointerup', onUp)
      window.removeEventListener('pointercancel', onCancel)
    }
  })

  // Выкид: у пункта 0 на доске нет, поэтому отдельный контрол. Появляется,
  // когда у выделенной шашки есть легальный выкид (to === 0).
  const bearOffAvailable = $derived(
    selectedFrom !== null && legalMoves.some((m) => m.from === selectedFrom && m.to === 0),
  )

  function handleBearOff(): void {
    if (selectedFrom !== null) commitMove(selectedFrom, 0)
  }

  // Выкид перетаскиванием (#43): drop-зона видна, пока тащим шашку с легальным
  // выкидом. Отпускание над ней резолвится в endDrag по координатам → to=0.
  const dragBearOffAvailable = $derived(
    dragFrom !== null && legalMoves.some((m) => m.from === dragFrom && m.to === 0),
  )
</script>

<div class="board-area">
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <svg
    bind:this={boardEl}
    class="board"
    viewBox={`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`}
    xmlns="http://www.w3.org/2000/svg"
    onpointermove={moveDrag}
  >
    <rect class="bar" x={BAR_X} y="0" width={BAR_WIDTH} height={VIEWBOX_HEIGHT} />
    {#each POINTS as point (point)}
      {@const hint = hintFor(point)}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <polygon
        class="point {point % 2 === 0 ? 'even' : 'odd'}{hint ? ` tier-${hint.dice}` : ''}"
        class:selected={selectedFrom === point || dragFrom === point}
        class:legal-target={hint !== undefined}
        data-testid="point-{point}"
        points={trianglePoints(point, flipped)}
        onclick={() => handlePointClick(point)}
        onpointerdown={(e) => startDrag(point, e)}
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
        />
      {/each}
    {/each}

    <!-- Метки-подсказки (#49): бейдж над каждой достижимой целью. Цифра =
         суммарная дистанция, цвет (через tier-N) = число потраченных кубиков.
         Рисуются после шашек (поверх). pointer-events:none — не перехватывают
         клик/drop, адресованный треугольнику-цели под ними. -->
    {#each moveHints as hint (hint.to)}
      {@const p = hintPos(hint.to, flipped)}
      <g class="move-hint tier-{hint.dice}" data-testid="move-hint-{hint.to}">
        <circle class="move-hint-bg" cx={p.x} cy={p.y} r={HINT_RADIUS} />
        <text
          class="move-hint-pip"
          x={p.x}
          y={p.y}
          text-anchor="middle"
          dominant-baseline="central">{hint.dist}</text>
      </g>
    {/each}
  </svg>

  <!-- Зона выкида. Позиция — через CSS: десктоп (≥1024px) слева от доски,
       мобила — оверлеем на доске по центру. Drag-режим показывает drop-цель
       (резолвится по координатам отпускания в endDrag, не обработчиком),
       клик-режим — кнопку. И то и другое — в одном месте по запросу. -->
  {#if dragBearOffAvailable || bearOffAvailable}
    <div class="bear-off-zone">
      {#if dragBearOffAvailable}
        <div class="bear-off-drop" data-testid="bear-off-drop">Сбросить сюда</div>
      {/if}
      {#if bearOffAvailable}
        <button type="button" class="bear-off" data-testid="bear-off" onclick={handleBearOff}>
          Сбросить шашку
        </button>
      {/if}
    </div>
  {/if}

  <!-- Легенда (#49): какой цвет = сколько кубиков. Видна только когда есть
       составная цель (одиночные ходы и так зелёные, объяснять нечего). -->
  {#if showDiceLegend}
    <div class="dice-legend" data-testid="dice-legend">
      {#each legendTiers as dice (dice)}
        <span class="legend-item tier-{dice}">
          <span class="legend-swatch"></span>{diceLabel(dice)}
        </span>
      {/each}
    </div>
  {/if}
</div>

{#if dragFrom !== null}
  <!-- Призрак-шашка следует за указателем. HTML-оверлей (position:fixed) поверх
       всей страницы — виден и за пределами доски (до зоны выкида под ней).
       pointer-events:none — не перехватывает отпускание у цели под пальцем. -->
  <div
    class="drag-ghost {myColor === 'white' ? 'white' : 'black'}"
    data-testid="drag-ghost"
    style="left: {ghostX}px; top: {ghostY}px; width: {ghostSize}px; height: {ghostSize}px;"
  ></div>
{/if}

<style>
  .board-area {
    /* Контекст позиционирования для зоны выкида (абсолютный оверлей). */
    position: relative;
  }
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
  /* Тач: явно и на интерактивных элементах — чтобы жест перетаскивания не
     уходил в скролл страницы/боттом-шита. */
  .point,
  .checker {
    touch-action: none;
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
  /* Метка-подсказка (#47): не перехватывает указатель — клик/drop идёт сквозь
     неё к треугольнику-цели под ней. */
  .move-hint {
    pointer-events: none;
  }
  .move-hint-bg {
    fill: #2e7d32;
    stroke: #f1f8e9;
    stroke-width: 3;
  }
  .move-hint-pip {
    fill: #ffffff;
    font-size: 30px;
    font-weight: 700;
  }
  /* Тиры по числу кубиков (#49): цель и бейдж окрашиваются по тому, сколько
     кубиков тратится на ход до неё. tier-1 (1 кубик) — зелёный (база выше). */
  .point.legal-target.tier-2 {
    fill: #90caf9;
    stroke: #1565c0;
  }
  .point.legal-target.tier-3 {
    fill: #ffcc80;
    stroke: #e65100;
  }
  .point.legal-target.tier-4 {
    fill: #ce93d8;
    stroke: #6a1b9a;
  }
  .move-hint.tier-2 .move-hint-bg {
    fill: #1565c0;
  }
  .move-hint.tier-3 .move-hint-bg {
    fill: #e65100;
  }
  .move-hint.tier-4 .move-hint-bg {
    fill: #6a1b9a;
  }
  .dice-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem 1rem;
    margin-top: 0.5rem;
    font-size: 13px;
    font-weight: 700;
    color: #3b2410;
  }
  .legend-item {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .legend-swatch {
    width: 14px;
    height: 14px;
    border-radius: 3px;
    border: 2px solid #2e7d32;
    background: #aed581;
    box-sizing: border-box;
  }
  .legend-item.tier-2 .legend-swatch {
    background: #90caf9;
    border-color: #1565c0;
  }
  .legend-item.tier-3 .legend-swatch {
    background: #ffcc80;
    border-color: #e65100;
  }
  .legend-item.tier-4 .legend-swatch {
    background: #ce93d8;
    border-color: #6a1b9a;
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
  .bear-off-zone {
    position: absolute;
    z-index: 5;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    align-items: stretch;
    border-radius: 6px;
    box-shadow: 0 2px 12px rgba(42, 30, 16, 0.4);
    /* Мобила (по умолчанию): на доске, примерно в середине. */
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: max-content;
    max-width: 70%;
  }
  @media (min-width: 1024px) {
    /* Десктоп: слева от доски, у верхнего края. */
    .bear-off-zone {
      top: 0.5rem;
      right: calc(100% + 1rem);
      left: auto;
      bottom: auto;
      transform: none;
      width: 150px;
      max-width: none;
    }
  }
  .bear-off {
    background: #aed581;
    color: #1b5e20;
    border: 2px solid #2e7d32;
    border-radius: 6px;
    padding: 0.6rem 1rem;
    font-size: 15px;
    font-weight: 700;
    cursor: pointer;
  }
  .bear-off:hover {
    background: #9ccc65;
  }
  /* Призрак — HTML-оверлей поверх всей страницы (виден за пределами доски). */
  .drag-ghost {
    position: fixed;
    z-index: 1000;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    pointer-events: none;
    opacity: 0.85;
    box-sizing: border-box;
  }
  .drag-ghost.white {
    background: #f4ece1;
    border: 2px solid #2a1e10;
  }
  .drag-ghost.black {
    background: #2a1e10;
    border: 2px solid #f4ece1;
  }
  .bear-off-drop {
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
