// Геометрия SVG-доски длинных нард.
// Раскладка: 24 пункта в два ряда по 12, без bar (битья нет, см. SPEC.md § 1).
//
//   13 14 15 16 17 18 19 20 21 22 23 24   ← верхний ряд, шашки растут вниз
//   12 11 10  9  8  7  6  5  4  3  2  1   ← нижний ряд, шашки растут вверх

export const COLUMNS = 12
export const COLUMN_WIDTH = 60
// Центральный бар — визуальный разделитель двух половин (по 6 пунктов).
// Битья нет, но бар нужен как ориентир (#3).
export const BAR_WIDTH = 40
export const VIEWBOX_WIDTH = COLUMNS * COLUMN_WIDTH + BAR_WIDTH
export const VIEWBOX_HEIGHT = 760
// Левый край бара — сразу после 6 левых колонок.
export const BAR_X = (COLUMNS / 2) * COLUMN_WIDTH

export const CHECKER_RADIUS = 25
export const CHECKER_DIAMETER = CHECKER_RADIUS * 2
// Смещение центра первой шашки от основания (чтобы круг не вылезал за край).
const CHECKER_BASE_OFFSET = CHECKER_RADIUS + 5

export interface PointAnchor {
  /** x-координата центра треугольника пункта */
  x: number
  /** y-координата основания треугольника (откуда «растёт» стопка шашек) */
  y: number
  /** направление роста стопки: 'up' для нижнего ряда, 'down' для верхнего */
  direction: 'up' | 'down'
}

// columnX — x центра колонки col (0..11) с учётом зазора-бара после 6-й.
function columnX(col: number): number {
  const barGap = col >= COLUMNS / 2 ? BAR_WIDTH : 0
  return col * COLUMN_WIDTH + barGap + COLUMN_WIDTH / 2
}

// pointAnchor(point, flipped): якорь пункта. flipped=true поворачивает доску
// на 180° (перспектива второго игрока) — нужно, чтобы свой цвет был слева.
export function pointAnchor(point: number, flipped = false): PointAnchor {
  if (point < 1 || point > 24 || !Number.isInteger(point)) {
    throw new Error(`pointAnchor: point ${point} out of range [1..24]`)
  }
  const base: PointAnchor =
    point <= 12
      ? { x: columnX(COLUMNS - point), y: VIEWBOX_HEIGHT, direction: 'up' } // 1 справа, 12 слева
      : { x: columnX(point - 13), y: 0, direction: 'down' } // 13 слева, 24 справа
  if (!flipped) return base
  return {
    x: VIEWBOX_WIDTH - base.x,
    y: VIEWBOX_HEIGHT - base.y,
    direction: base.direction === 'up' ? 'down' : 'up',
  }
}

export interface CheckerPosition {
  cx: number
  cy: number
  r: number
}

// До STACK_NO_OVERLAP шашек в стопке кладём вплотную (шаг = диаметр); сверх —
// шаг сжимаем, чтобы вся стопка влезла в свою половину поля (15 шашек тоже).
const STACK_NO_OVERLAP = 5

function stackStep(count: number): number {
  if (count <= STACK_NO_OVERLAP) return CHECKER_DIAMETER
  // Центр верхней шашки не должен выходить за середину поля.
  const maxOffset = VIEWBOX_HEIGHT / 2 - CHECKER_RADIUS
  return Math.min(CHECKER_DIAMETER, (maxOffset - CHECKER_BASE_OFFSET) / (count - 1))
}

// checkerAt — позиция index-й шашки в стопке из count штук на пункте.
// count управляет наложением (0/малое → без наложения, легаси-вызовы),
// flipped — ориентацией доски (см. pointAnchor).
export function checkerAt(
  point: number,
  index: number,
  count = 0,
  flipped = false,
): CheckerPosition {
  const anchor = pointAnchor(point, flipped)
  const offset = CHECKER_BASE_OFFSET + index * stackStep(count)
  const cy = anchor.direction === 'up' ? anchor.y - offset : anchor.y + offset
  return { cx: anchor.x, cy, r: CHECKER_RADIUS }
}
