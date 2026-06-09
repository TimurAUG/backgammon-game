// Геометрия SVG-доски длинных нард.
// Раскладка: 24 пункта в два ряда по 12, без bar (битья нет, см. SPEC.md § 1).
//
//   13 14 15 16 17 18 19 20 21 22 23 24   ← верхний ряд, шашки растут вниз
//   12 11 10  9  8  7  6  5  4  3  2  1   ← нижний ряд, шашки растут вверх

export const VIEWBOX_WIDTH = 800
export const VIEWBOX_HEIGHT = 600
export const COLUMNS = 12
export const COLUMN_WIDTH = VIEWBOX_WIDTH / COLUMNS

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

export function pointAnchor(point: number): PointAnchor {
  if (point < 1 || point > 24 || !Number.isInteger(point)) {
    throw new Error(`pointAnchor: point ${point} out of range [1..24]`)
  }
  if (point <= 12) {
    // нижний ряд: 1 справа, 12 слева
    const col = COLUMNS - point
    return {
      x: (col + 0.5) * COLUMN_WIDTH,
      y: VIEWBOX_HEIGHT,
      direction: 'up',
    }
  }
  // верхний ряд: 13 слева, 24 справа
  const col = point - 13
  return {
    x: (col + 0.5) * COLUMN_WIDTH,
    y: 0,
    direction: 'down',
  }
}

export interface CheckerPosition {
  cx: number
  cy: number
  r: number
}

export function checkerAt(point: number, index: number): CheckerPosition {
  const anchor = pointAnchor(point)
  const offset = CHECKER_BASE_OFFSET + index * CHECKER_DIAMETER
  const cy = anchor.direction === 'up' ? anchor.y - offset : anchor.y + offset
  return { cx: anchor.x, cy, r: CHECKER_RADIUS }
}
