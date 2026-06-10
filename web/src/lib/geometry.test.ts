// FRONTEND_SPEC #11 — pointAnchor: координаты якоря пункта в SVG-viewBox.
// FRONTEND_SPEC #12 — checkerAt: позиция j-й шашки в стопке на пункте.
//
// Геометрия — чистые функции, тесты идут напрямую без UI.
// Раскладка доски (стандарт):
//   1  — правый нижний угол (голова белых на 24 → правый ВЕРХНИЙ)
//   12 — левый нижний угол (рядом с 13 над ним)
//   13 — левый верхний угол (голова чёрных)
//   24 — правый верхний угол

import { describe, expect, test } from 'vitest'

import {
  BAR_WIDTH,
  CHECKER_DIAMETER,
  CHECKER_RADIUS,
  COLUMNS,
  COLUMN_WIDTH,
  VIEWBOX_HEIGHT,
  VIEWBOX_WIDTH,
  checkerAt,
  pointAnchor,
} from './geometry'

// x центра колонки col (0..11) с учётом бара после 6-й — зеркало geometry.
const colX = (col: number): number =>
  col * COLUMN_WIDTH + (col >= 6 ? BAR_WIDTH : 0) + COLUMN_WIDTH / 2

describe('pointAnchor (#11)', () => {
  test.each<[number, number, number, 'up' | 'down']>([
    // нижний ряд: 1 справа (col 11) → 12 слева (col 0), шашки растут вверх
    [1, colX(11), VIEWBOX_HEIGHT, 'up'],
    [6, colX(6), VIEWBOX_HEIGHT, 'up'],
    [7, colX(5), VIEWBOX_HEIGHT, 'up'],
    [12, colX(0), VIEWBOX_HEIGHT, 'up'],
    // верхний ряд: 13 слева (col 0) → 24 справа (col 11), шашки растут вниз
    [13, colX(0), 0, 'down'],
    [18, colX(5), 0, 'down'],
    [19, colX(6), 0, 'down'],
    [24, colX(11), 0, 'down'],
  ])('pointAnchor_point%i_atCorrectXYAndDirection', (point, x, y, dir) => {
    const a = pointAnchor(point)
    expect(a.x).toBeCloseTo(x, 5)
    expect(a.y).toBe(y)
    expect(a.direction).toBe(dir)
  })

  test('pointAnchor_outOfRange_throws', () => {
    expect(() => pointAnchor(0)).toThrow()
    expect(() => pointAnchor(25)).toThrow()
    expect(() => pointAnchor(-1)).toThrow()
  })

  test('pointAnchor_barGapWiderThanColumnStep (#3)', () => {
    // Центральный бар: разрыв через бар (6↔7 низ, 18↔19 верх) шире обычного
    // шага между соседними пунктами внутри половины.
    expect(pointAnchor(6).x - pointAnchor(7).x).toBeGreaterThan(
      pointAnchor(8).x - pointAnchor(9).x,
    )
    expect(pointAnchor(19).x - pointAnchor(18).x).toBeGreaterThan(
      pointAnchor(21).x - pointAnchor(20).x,
    )
  })
})

describe('checkerAt (#12)', () => {
  test('checkerAt_pointBottom_indexZero_offsetUpwardsFromBase', () => {
    const c = checkerAt(1, 0)
    const anchor = pointAnchor(1)
    expect(c.cx).toBeCloseTo(anchor.x, 5)
    // нижний ряд: первая шашка ВЫШЕ y=600
    expect(c.cy).toBeLessThan(VIEWBOX_HEIGHT)
    expect(c.r).toBe(CHECKER_RADIUS)
  })

  test('checkerAt_pointBottom_indexGrows_movesUpByDiameter', () => {
    const c0 = checkerAt(1, 0)
    const c1 = checkerAt(1, 1)
    const c2 = checkerAt(1, 2)
    // каждая следующая выше на CHECKER_DIAMETER
    expect(c0.cy - c1.cy).toBeCloseTo(CHECKER_DIAMETER, 5)
    expect(c1.cy - c2.cy).toBeCloseTo(CHECKER_DIAMETER, 5)
  })

  test('checkerAt_pointTop_indexGrows_movesDownByDiameter', () => {
    const c0 = checkerAt(13, 0)
    const c1 = checkerAt(13, 1)
    expect(c0.cy).toBeGreaterThan(0)
    expect(c1.cy - c0.cy).toBeCloseTo(CHECKER_DIAMETER, 5)
  })

  test('checkerAt_keepsXOfAnchor', () => {
    const anchor = pointAnchor(7)
    const c = checkerAt(7, 3)
    expect(c.cx).toBeCloseTo(anchor.x, 5)
  })
})

describe('checkerAt overlap when stack is tall (#5)', () => {
  test('checkerAt_fullStack15_topmostStaysWithinHalf', () => {
    // 15 шашек на нижнем пункте: верхняя (index 14) не должна вылезать за центр.
    const top = checkerAt(1, 14, 15)
    expect(top.cy).toBeGreaterThanOrEqual(VIEWBOX_HEIGHT / 2)
  })

  test('checkerAt_fullStack15_overlapsWithStepBelowDiameter', () => {
    const c0 = checkerAt(1, 0, 15)
    const c1 = checkerAt(1, 1, 15)
    expect(c0.cy - c1.cy).toBeLessThan(CHECKER_DIAMETER)
  })
})

describe('pointAnchor orientation flip (#1)', () => {
  test('pointAnchor_flipped_rotates180_point24goesToBottomLeft', () => {
    // Флип = поворот на 180°: пункт 24 (верх-право) уезжает туда же, где
    // пункт 12 без флипа (низ-лево) — «свой цвет слева».
    const flipped24 = pointAnchor(24, true)
    const normal12 = pointAnchor(12, false)
    expect(flipped24.x).toBeCloseTo(normal12.x, 5)
    expect(flipped24.y).toBe(normal12.y)
    expect(flipped24.direction).toBe(normal12.direction)
  })

  test('pointAnchor_defaultIsNotFlipped', () => {
    expect(pointAnchor(1, false)).toEqual(pointAnchor(1))
  })

  test('checkerAt_flipped_usesFlippedAnchor', () => {
    // Флипнутый нижний пункт 1 оказывается сверху (cy в верхней половине).
    const c = checkerAt(1, 0, 1, true)
    expect(c.cy).toBeLessThan(VIEWBOX_HEIGHT / 2)
  })
})

describe('viewBox constants', () => {
  test('viewBox_dimensions', () => {
    expect(VIEWBOX_WIDTH).toBe(760)
    expect(VIEWBOX_HEIGHT).toBe(760)
  })

  test('viewBoxWidth_isColumnsPlusBar', () => {
    expect(VIEWBOX_WIDTH).toBe(COLUMNS * COLUMN_WIDTH + BAR_WIDTH)
  })
})
