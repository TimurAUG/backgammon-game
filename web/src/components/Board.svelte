<script lang="ts">
  import type { Color, Move } from '../protocol/messages'

  import {
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

  function trianglePoints(point: number): string {
    const a = pointAnchor(point)
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
    if (selectedFrom === null) return false
    return legalMoves.some((m) => m.from === selectedFrom && m.to === point)
  }

  function handlePointClick(point: number): void {
    if (myColor === null) return
    if (selectedFrom !== null && isLegalTarget(point)) {
      const from = selectedFrom
      selectedFrom = null
      onMove(from, point)
      return
    }
    if (isMyChecker(point)) {
      selectedFrom = selectedFrom === point ? null : point
      return
    }
    // клик по чужой/пустой при невыделенной → ничего
  }

  // Выкид: у пункта 0 на доске нет, поэтому отдельный контрол. Появляется,
  // когда у выделенной шашки есть легальный выкид (to === 0).
  const bearOffAvailable = $derived(
    selectedFrom !== null && legalMoves.some((m) => m.from === selectedFrom && m.to === 0),
  )

  function handleBearOff(): void {
    if (selectedFrom === null) return
    const from = selectedFrom
    selectedFrom = null
    onMove(from, 0)
  }
</script>

<svg
  class="board"
  viewBox={`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`}
  xmlns="http://www.w3.org/2000/svg"
>
  {#each POINTS as point (point)}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <polygon
      class="point {point % 2 === 0 ? 'even' : 'odd'}"
      class:selected={selectedFrom === point}
      class:legal-target={isLegalTarget(point)}
      data-testid="point-{point}"
      points={trianglePoints(point)}
      onclick={() => handlePointClick(point)}
    />
  {/each}

  {#each POINTS as point (point)}
    {#each Array.from({ length: checkerCount(point) }, (_, j) => j) as j (j)}
      {@const pos = checkerAt(point, j, checkerCount(point))}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <circle
        class="checker {checkerColor(point)}"
        data-testid="checker-{point}-{j}"
        cx={pos.cx}
        cy={pos.cy}
        r={pos.r}
        onclick={() => handlePointClick(point)}
      />
    {/each}
  {/each}
</svg>

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
    max-width: 800px;
    border: 2px solid #5a3a1e;
    background: #e7c79b;
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
</style>
