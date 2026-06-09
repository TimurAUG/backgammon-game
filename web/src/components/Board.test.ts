// FRONTEND_SPEC #13 — Board.svelte рендерит 24 пункта из переданного board.
// FRONTEND_SPEC #14 — Шашки на пунктах: количество = |board[i]|,
//   цвет 'white' при board[i] > 0, 'black' при board[i] < 0.
//
// Board принимает board через $props (чистый компонент), а не читает
// напрямую из стора — это упрощает изоляцию тестов и переиспользование.

import { render, screen } from '@testing-library/svelte'
import { describe, expect, test } from 'vitest'

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
