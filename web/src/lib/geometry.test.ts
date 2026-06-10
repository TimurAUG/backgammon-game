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
  CHECKER_DIAMETER,
  CHECKER_RADIUS,
  COLUMN_WIDTH,
  VIEWBOX_HEIGHT,
  VIEWBOX_WIDTH,
  checkerAt,
  pointAnchor,
} from './geometry'

describe('pointAnchor (#11)', () => {
  test.each<[number, number, number, 'up' | 'down']>([
    // нижний ряд: 1 справа → 12 слева, шашки растут вверх
    [1, 11.5 * COLUMN_WIDTH, VIEWBOX_HEIGHT, 'up'],
    [6, 6.5 * COLUMN_WIDTH, VIEWBOX_HEIGHT, 'up'],
    [7, 5.5 * COLUMN_WIDTH, VIEWBOX_HEIGHT, 'up'],
    [12, 0.5 * COLUMN_WIDTH, VIEWBOX_HEIGHT, 'up'],
    // верхний ряд: 13 слева → 24 справа, шашки растут вниз
    [13, 0.5 * COLUMN_WIDTH, 0, 'down'],
    [18, 5.5 * COLUMN_WIDTH, 0, 'down'],
    [19, 6.5 * COLUMN_WIDTH, 0, 'down'],
    [24, 11.5 * COLUMN_WIDTH, 0, 'down'],
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

describe('viewBox constants', () => {
  test('viewBox_dimensionsAre800x600', () => {
    expect(VIEWBOX_WIDTH).toBe(800)
    expect(VIEWBOX_HEIGHT).toBe(600)
  })

  test('columnWidth_isViewBoxWidthDivided12', () => {
    expect(COLUMN_WIDTH).toBeCloseTo(VIEWBOX_WIDTH / 12, 5)
  })
})
