<script lang="ts">
  import {
    COLUMN_WIDTH,
    VIEWBOX_HEIGHT,
    VIEWBOX_WIDTH,
    checkerAt,
    pointAnchor,
  } from '../lib/geometry'

  let { board }: { board: number[] } = $props()

  const POINTS: number[] = Array.from({ length: 24 }, (_, i) => i + 1)
  const TRIANGLE_HEIGHT = 250
  const TRIANGLE_HALF_BASE = COLUMN_WIDTH * 0.4

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
</script>

<svg
  class="board"
  viewBox={`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`}
  xmlns="http://www.w3.org/2000/svg"
>
  {#each POINTS as point (point)}
    <polygon
      class="point {point % 2 === 0 ? 'even' : 'odd'}"
      data-testid="point-{point}"
      points={trianglePoints(point)}
    />
  {/each}

  {#each POINTS as point (point)}
    {#each Array.from({ length: checkerCount(point) }, (_, j) => j) as j (j)}
      {@const pos = checkerAt(point, j)}
      <circle
        class="checker {checkerColor(point)}"
        data-testid="checker-{point}-{j}"
        cx={pos.cx}
        cy={pos.cy}
        r={pos.r}
      />
    {/each}
  {/each}
</svg>

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
</style>
