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
import { describe, expect, test, vi } from 'vitest'

import type { Move } from '../protocol/messages'

import Board from './Board.svelte'

function emptyBoard(): number[] {
  return Array(24).fill(0)
}

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
