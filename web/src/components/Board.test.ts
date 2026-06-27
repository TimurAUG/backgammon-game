// FRONTEND_SPEC #13 — Board.svelte рендерит 24 пункта из переданного board.
// FRONTEND_SPEC #14 — Шашки на пунктах: количество = |board[i]|,
//   цвет 'white' при board[i] > 0, 'black' при board[i] < 0.
// FRONTEND_SPEC #19 — Клик по своей шашке выделяет пункт, подсвечивает
//   targets из legalMoves для этого from.
// FRONTEND_SPEC #20 — Клик по подсвеченной точке вызывает onMove(from, to)
//   и сбрасывает выделение.
//
// Board принимает board через $props (чистый компонент), а не читает
// напрямую из стора — это упрощает изоляцию тестов и переиспользование.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { tick } from 'svelte'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { Move } from '../protocol/messages'

import Board from './Board.svelte'

function emptyBoard(): number[] {
  return Array(24).fill(0)
}

// Drag включается по таймеру удержания — нужны фейковые таймеры, чтобы его
// «домотать». Тап/клик таймер не запускает (или снимает на pointerup).
beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('Board pieces rendering (#13)', () => {
  test('Board_renders24Points', () => {
    render(Board, { props: { board: emptyBoard() } })
    expect(screen.getAllByTestId(/^point-\d+$/)).toHaveLength(24)
  })

  test('Board_emptyBoard_noCheckersRendered', () => {
    render(Board, { props: { board: emptyBoard() } })
    expect(screen.queryAllByTestId(/^checker-/)).toHaveLength(0)
  })
})

describe('Board checker count and color (#14)', () => {
  test('Board_initial24_15WhiteCheckers', () => {
    const board = emptyBoard()
    board[23] = 15 // голова белых
    render(Board, { props: { board } })

    const checkers = screen.getAllByTestId(/^checker-24-\d+$/)
    expect(checkers).toHaveLength(15)
    for (const c of checkers) {
      expect(c).toHaveClass('white')
    }
  })

  test('Board_initial12_15BlackCheckers', () => {
    const board = emptyBoard()
    board[11] = -15 // голова чёрных
    render(Board, { props: { board } })

    const checkers = screen.getAllByTestId(/^checker-12-\d+$/)
    expect(checkers).toHaveLength(15)
    for (const c of checkers) {
      expect(c).toHaveClass('black')
    }
  })

  test('Board_modulusDeterminesCount_signDeterminesColor', () => {
    const board = emptyBoard()
    board[0] = 3 // пункт 1: 3 белых
    board[5] = -2 // пункт 6: 2 чёрных
    render(Board, { props: { board } })

    const whites = screen.getAllByTestId(/^checker-1-\d+$/)
    expect(whites).toHaveLength(3)
    expect(whites[0]).toHaveClass('white')

    const blacks = screen.getAllByTestId(/^checker-6-\d+$/)
    expect(blacks).toHaveLength(2)
    expect(blacks[0]).toHaveClass('black')
  })

  test('Board_unaffectedPoints_noCheckersRendered', () => {
    const board = emptyBoard()
    board[0] = 3
    render(Board, { props: { board } })
    // только point 1 имеет шашки; на остальных пунктах — нет
    expect(screen.queryAllByTestId(/^checker-7-/)).toHaveLength(0)
    expect(screen.queryAllByTestId(/^checker-24-/)).toHaveLength(0)
  })
})

describe('Board click selection (#19)', () => {
  test('Board_clickOwnChecker_marksPointSelected', async () => {
    const board = emptyBoard()
    board[23] = 15
    render(Board, {
      props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() },
    })

    await fireEvent.click(screen.getByTestId('point-24'))
    expect(screen.getByTestId('point-24')).toHaveClass('selected')
  })

  test('Board_clickOpponentChecker_doesNotSelect', async () => {
    const board = emptyBoard()
    board[11] = -15
    render(Board, {
      props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() },
    })

    await fireEvent.click(screen.getByTestId('point-12'))
    expect(screen.getByTestId('point-12')).not.toHaveClass('selected')
  })

  test('Board_clickEmptyPoint_noSelection', async () => {
    const board = emptyBoard()
    render(Board, {
      props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() },
    })

    await fireEvent.click(screen.getByTestId('point-5'))
    expect(screen.getByTestId('point-5')).not.toHaveClass('selected')
  })

  test('Board_afterSelect_legalTargetsHighlighted', async () => {
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [
      { from: 24, to: 18, pip: 6 },
      { from: 24, to: 19, pip: 5 },
    ]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await fireEvent.click(screen.getByTestId('point-24'))

    expect(screen.getByTestId('point-18')).toHaveClass('legal-target')
    expect(screen.getByTestId('point-19')).toHaveClass('legal-target')
    expect(screen.getByTestId('point-23')).not.toHaveClass('legal-target')
  })

  test('Board_noSelection_noTargetsHighlighted', () => {
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    expect(screen.getByTestId('point-18')).not.toHaveClass('legal-target')
  })

  test('Board_withoutMyColor_clicksAreNoOp', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, legalMoves, onMove } })

    await fireEvent.click(screen.getByTestId('point-24'))
    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
    expect(screen.getByTestId('point-18')).not.toHaveClass('legal-target')
  })
})

describe('Board click target → onMove (#20)', () => {
  test('Board_clickLegalTarget_callsOnMoveWithFromTo', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove } })

    await fireEvent.click(screen.getByTestId('point-24'))
    await fireEvent.click(screen.getByTestId('point-18'))

    expect(onMove).toHaveBeenCalledWith(24, 18)
  })

  test('Board_clickIllegalTarget_doesNotCallOnMove', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove } })

    await fireEvent.click(screen.getByTestId('point-24'))
    await fireEvent.click(screen.getByTestId('point-1')) // не в legalMoves

    expect(onMove).not.toHaveBeenCalled()
  })

  test('Board_afterMoveSent_selectionCleared', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove } })

    await fireEvent.click(screen.getByTestId('point-24'))
    await fireEvent.click(screen.getByTestId('point-18'))

    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
    expect(screen.getByTestId('point-18')).not.toHaveClass('legal-target')
  })

  test('Board_clickOtherOwnChecker_switchesSelection', async () => {
    const board = emptyBoard()
    board[23] = 5 // на 24 пять белых
    board[0] = 3 // на 1 три белых
    const legalMoves: Move[] = [
      { from: 24, to: 18, pip: 6 },
      { from: 1, to: 0, pip: 1 },
    ]
    render(Board, {
      props: { board, myColor: 'white', legalMoves, onMove: vi.fn() },
    })

    await fireEvent.click(screen.getByTestId('point-24'))
    expect(screen.getByTestId('point-24')).toHaveClass('selected')

    await fireEvent.click(screen.getByTestId('point-1'))
    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
    expect(screen.getByTestId('point-1')).toHaveClass('selected')
  })
})

describe('Board bear-off UI (#20b)', () => {
  test('Board_selectCheckerWithBearOff_showsBearOffControl', async () => {
    const board = emptyBoard()
    board[5] = 3 // 3 белых на пункте 6
    const legalMoves: Move[] = [{ from: 6, to: 0, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    expect(screen.queryByTestId('bear-off')).toBeNull()

    await fireEvent.click(screen.getByTestId('point-6'))

    expect(screen.getByTestId('bear-off')).toBeInTheDocument()
  })

  test('Board_clickBearOff_callsOnMoveWithToZero', async () => {
    const board = emptyBoard()
    board[5] = 3
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 6, to: 0, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove } })

    await fireEvent.click(screen.getByTestId('point-6'))
    await fireEvent.click(screen.getByTestId('bear-off'))

    expect(onMove).toHaveBeenCalledWith(6, 0)
  })

  test('Board_selectCheckerWithoutBearOff_hidesBearOffControl', async () => {
    const board = emptyBoard()
    board[23] = 5 // белые на 24, только обычный ход
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await fireEvent.click(screen.getByTestId('point-24'))

    expect(screen.queryByTestId('bear-off')).toBeNull()
  })
})

// Жест перетаскивания: нажать на источник и УДЕРЖАТЬ (домотать таймер). Чистый
// pointerDown без удержания — это тап (клик-режим), не drag.
async function dragGesture(source: HTMLElement): Promise<void> {
  await fireEvent.pointerDown(source)
  vi.advanceTimersByTime(200) // удержание дольше HOLD_MS → берём шашку
  await tick()
}

describe('Board drag start (#41)', () => {
  test('Board_dragOwnChecker_marksSourceSelected', async () => {
    const board = emptyBoard()
    board[23] = 15
    render(Board, { props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() } })

    await dragGesture(screen.getByTestId('checker-24-0'))

    expect(screen.getByTestId('point-24')).toHaveClass('selected')
  })

  test('Board_dragOwnChecker_highlightsLegalTargets', async () => {
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [
      { from: 24, to: 18, pip: 6 },
      { from: 24, to: 19, pip: 5 },
    ]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await dragGesture(screen.getByTestId('checker-24-0'))

    expect(screen.getByTestId('point-18')).toHaveClass('legal-target')
    expect(screen.getByTestId('point-19')).toHaveClass('legal-target')
  })

  test('Board_dragOpponentChecker_doesNotStartDrag', async () => {
    const board = emptyBoard()
    board[11] = -15
    render(Board, { props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() } })

    await dragGesture(screen.getByTestId('checker-12-0'))

    expect(screen.getByTestId('point-12')).not.toHaveClass('selected')
  })

  test('Board_dragWithoutMyColor_doesNotStartDrag', async () => {
    const board = emptyBoard()
    board[23] = 15
    render(Board, { props: { board, legalMoves: [], onMove: vi.fn() } })

    await dragGesture(screen.getByTestId('checker-24-0'))

    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
  })

  test('Board_pointerDownWithoutMove_doesNotStartDrag', async () => {
    // чистый тап (нажатие без движения) — это клик-режим, не drag:
    // ни призрака, ни подсветки источника до движения
    const board = emptyBoard()
    board[23] = 15
    render(Board, { props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() } })

    await fireEvent.pointerDown(screen.getByTestId('checker-24-0'))

    expect(screen.queryByTestId('drag-ghost')).toBeNull()
    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
  })
})

describe('Board hold-to-drag (#41)', () => {
  test('Board_hold_picksUpAndShowsGhost', async () => {
    const board = emptyBoard()
    board[23] = 15
    render(Board, { props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() } })

    expect(screen.queryByTestId('drag-ghost')).toBeNull()

    await fireEvent.pointerDown(screen.getByTestId('checker-24-0'))
    vi.advanceTimersByTime(200) // удержали дольше HOLD_MS
    await tick()

    expect(screen.getByTestId('drag-ghost')).toBeInTheDocument()
  })

  test('Board_releaseBeforeHold_doesNotPickUp', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    render(Board, { props: { board, myColor: 'white', legalMoves: [], onMove } })
    const checker = screen.getByTestId('checker-24-0')

    await fireEvent.pointerDown(checker)
    vi.advanceTimersByTime(100) // меньше HOLD_MS
    await fireEvent.pointerUp(checker)
    await tick()

    expect(screen.queryByTestId('drag-ghost')).toBeNull()
    expect(onMove).not.toHaveBeenCalled()
  })

  test('Board_shortPress_clearedTimer_doesNotPickUpLater', async () => {
    // регрессия «прилипает к курсору»: отпустили до удержания — таймер снят,
    // позже шашка не «подхватывается» сама
    const board = emptyBoard()
    board[23] = 15
    render(Board, { props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() } })
    const checker = screen.getByTestId('checker-24-0')

    await fireEvent.pointerDown(checker)
    await fireEvent.pointerUp(checker)
    vi.advanceTimersByTime(500) // даже если подождать — таймер уже снят
    await tick()

    expect(screen.queryByTestId('drag-ghost')).toBeNull()
  })
})

describe('Board drag drop → onMove (#42)', () => {
  test('Board_dragToLegalTarget_callsOnMoveWithFromTo', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove } })

    await dragGesture(screen.getByTestId('checker-24-0'))
    await fireEvent.pointerUp(screen.getByTestId('point-18'))

    expect(onMove).toHaveBeenCalledWith(24, 18)
  })

  test('Board_dragToIllegalTarget_doesNotCallOnMove', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove } })

    await dragGesture(screen.getByTestId('checker-24-0'))
    await fireEvent.pointerUp(screen.getByTestId('point-1')) // не в legalMoves

    expect(onMove).not.toHaveBeenCalled()
  })

  test('Board_tapWithoutDrag_doesNotCallOnMove', async () => {
    // нажать и отпустить на легальной цели БЕЗ движения — drag не активен,
    // ход не уходит (это работа клик-режима, не drag)
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove } })

    await fireEvent.pointerDown(screen.getByTestId('checker-24-0'))
    await fireEvent.pointerUp(screen.getByTestId('point-18'))

    expect(onMove).not.toHaveBeenCalled()
  })

  test('Board_afterDrop_sourceAndTargetsCleared', async () => {
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await dragGesture(screen.getByTestId('checker-24-0'))
    await fireEvent.pointerUp(screen.getByTestId('point-18'))

    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
    expect(screen.getByTestId('point-18')).not.toHaveClass('legal-target')
  })

  test('Board_releaseOutsideTarget_cancelsWithoutMove', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    const { container } = render(Board, {
      props: { board, myColor: 'white', legalMoves, onMove },
    })

    await dragGesture(screen.getByTestId('checker-24-0'))
    const svg = container.querySelector('svg.board') as SVGSVGElement
    await fireEvent.pointerUp(svg)

    expect(onMove).not.toHaveBeenCalled()
    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
  })
})

describe('Board drag bear-off (#43)', () => {
  test('Board_dragCheckerWithBearOff_showsDropZone', async () => {
    const board = emptyBoard()
    board[5] = 3 // 3 белых на пункте 6
    const legalMoves: Move[] = [{ from: 6, to: 0, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    expect(screen.queryByTestId('bear-off-drop')).toBeNull()

    await dragGesture(screen.getByTestId('checker-6-0'))

    expect(screen.getByTestId('bear-off-drop')).toBeInTheDocument()
  })

  test('Board_dragToBearOffZone_callsOnMoveToZero', async () => {
    const board = emptyBoard()
    board[5] = 3
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 6, to: 0, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove } })

    await dragGesture(screen.getByTestId('checker-6-0'))
    await fireEvent.pointerUp(screen.getByTestId('bear-off-drop'))

    expect(onMove).toHaveBeenCalledWith(6, 0)
  })

  test('Board_dragCheckerWithoutBearOff_noDropZone', async () => {
    const board = emptyBoard()
    board[23] = 5 // белые на 24, только обычный ход — выкида нет
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await dragGesture(screen.getByTestId('checker-24-0'))

    expect(screen.queryByTestId('bear-off-drop')).toBeNull()
  })
})

describe('Board drag ghost and cancel (#44)', () => {
  test('Board_duringDrag_showsGhost', async () => {
    const board = emptyBoard()
    board[23] = 15
    render(Board, { props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() } })

    expect(screen.queryByTestId('drag-ghost')).toBeNull()

    await dragGesture(screen.getByTestId('checker-24-0'))

    expect(screen.getByTestId('drag-ghost')).toBeInTheDocument()
  })

  test('Board_afterDrop_ghostRemoved', async () => {
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await dragGesture(screen.getByTestId('checker-24-0'))
    await fireEvent.pointerUp(screen.getByTestId('point-18'))

    expect(screen.queryByTestId('drag-ghost')).toBeNull()
  })

  test('Board_pointerCancel_cancelsDragWithoutMove', async () => {
    const board = emptyBoard()
    board[23] = 15
    const onMove = vi.fn()
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    const { container } = render(Board, {
      props: { board, myColor: 'white', legalMoves, onMove },
    })

    await dragGesture(screen.getByTestId('checker-24-0'))
    const svg = container.querySelector('svg.board') as SVGSVGElement
    await fireEvent.pointerCancel(svg)

    expect(onMove).not.toHaveBeenCalled()
    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
    expect(screen.queryByTestId('drag-ghost')).toBeNull()
  })
})

describe('Board click deselect (regression)', () => {
  test('Board_tapSelectedChecker_deselects', async () => {
    // повторный тап по выделенной шашке снимает выделение. Полная
    // последовательность тапа: pointerdown + pointerup (без движения) + click.
    const board = emptyBoard()
    board[23] = 15
    render(Board, { props: { board, myColor: 'white', legalMoves: [], onMove: vi.fn() } })
    const checker = screen.getByTestId('checker-24-0')

    await fireEvent.pointerDown(checker)
    await fireEvent.pointerUp(checker)
    await fireEvent.click(checker)
    expect(screen.getByTestId('point-24')).toHaveClass('selected')

    await fireEvent.pointerDown(checker)
    await fireEvent.pointerUp(checker)
    await fireEvent.click(checker)
    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
  })

  test('Board_clickEmptyPoint_deselects', async () => {
    // клик по пустому пункту при выделении снимает выделение
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await fireEvent.click(screen.getByTestId('checker-24-0'))
    expect(screen.getByTestId('point-24')).toHaveClass('selected')

    await fireEvent.click(screen.getByTestId('point-1')) // пустой пункт
    expect(screen.getByTestId('point-24')).not.toHaveClass('selected')
  })
})

// FRONTEND_SPEC #47 — метка-подсказка с цифрой pip (значение кубика = дальность
//   хода) над каждой подсвеченной целью выбранной/захваченной шашки.
describe('Board move-hint pip labels (#47)', () => {
  test('Board_afterSelect_showsPipLabelOnEachLegalTarget', async () => {
    // выбрали шашку → над каждой целью её pip (каким кубиком туда идём)
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [
      { from: 24, to: 18, pip: 6 },
      { from: 24, to: 19, pip: 5 },
    ]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await fireEvent.click(screen.getByTestId('point-24'))

    expect(screen.getByTestId('move-hint-18')).toHaveTextContent('6')
    expect(screen.getByTestId('move-hint-19')).toHaveTextContent('5')
  })

  test('Board_withoutSelection_noPipLabels', async () => {
    // legalMoves есть, но шашка не выбрана → меток нет
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    expect(screen.queryAllByTestId(/^move-hint-/)).toHaveLength(0)
  })

  test('Board_bearOffTarget_noPipLabelOnBoard', async () => {
    // выкид (to == 0) — это зона/кнопка, а не треугольник: метки на доске нет
    const board = emptyBoard()
    board[5] = 3
    const legalMoves: Move[] = [{ from: 6, to: 0, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await fireEvent.click(screen.getByTestId('point-6'))

    expect(screen.queryByTestId('move-hint-0')).toBeNull()
  })

  test('Board_dragOwnChecker_showsPipLabels', async () => {
    // тот же activeFrom: метки появляются и при захвате (drag), не только клике
    const board = emptyBoard()
    board[23] = 15
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await dragGesture(screen.getByTestId('checker-24-0'))

    expect(screen.getByTestId('move-hint-18')).toHaveTextContent('6')
  })

  test('Board_targetWithOwnCheckers_hintRenderedAbovePieces', async () => {
    // на цели уже стоят свои шашки: бейдж должен идти ПОСЛЕ них в DOM →
    // рисуется поверх, цифра pip не перекрывается шашками (много фишек на цели)
    const board = emptyBoard()
    board[23] = 12 // голова белых
    board[17] = 3 // 3 свои белые уже на пункте 18 (целевой)
    const legalMoves: Move[] = [{ from: 24, to: 18, pip: 6 }]
    render(Board, { props: { board, myColor: 'white', legalMoves, onMove: vi.fn() } })

    await fireEvent.click(screen.getByTestId('point-24'))

    const hint = screen.getByTestId('move-hint-18')
    const topChecker = screen.getByTestId('checker-18-2')
    expect(topChecker.compareDocumentPosition(hint) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })
})
