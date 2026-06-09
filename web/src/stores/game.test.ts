// FRONTEND_SPEC #7 — STATE обновляет board/turn/dice/borneOff/status/isFirstMove.
// FRONTEND_SPEC #8 — LEGAL_MOVES обновляет legalMoves (пустой = ход пропускается).
//
// gameState — модульный runes-объект ($state в game.svelte.ts).
// applyServerMessage — reducer-style action.

import { beforeEach, describe, expect, test } from 'vitest'

import { applyServerMessage, gameState, resetGameState } from './game.svelte'

beforeEach(() => {
  resetGameState()
})

describe('applyServerMessage STATE (#7)', () => {
  test('gameStore_onSTATE_updatesAllFields', () => {
    const board = Array(24).fill(0)
    board[23] = 15 // 15 белых на голове 24
    board[11] = -15 // 15 чёрных на голове 12

    applyServerMessage({
      type: 'STATE',
      board,
      turn: 'black',
      status: 'waitingForMove',
      borneOff: { white: 0, black: 0 },
      isFirstMove: { white: false, black: true },
      dice: { a: 6, b: 6, isDouble: true, remaining: [6, 6, 6, 6] },
    })

    expect(gameState.board).toEqual(board)
    expect(gameState.turn).toBe('black')
    expect(gameState.status).toBe('waitingForMove')
    expect(gameState.borneOff).toEqual({ white: 0, black: 0 })
    expect(gameState.isFirstMove).toEqual({ white: false, black: true })
    expect(gameState.dice).toEqual({ a: 6, b: 6, isDouble: true, remaining: [6, 6, 6, 6] })
  })

  test('gameStore_onSTATE_withoutDice_setsDiceToNull', () => {
    applyServerMessage({
      type: 'STATE',
      board: Array(24).fill(0),
      turn: 'white',
      status: 'waitingForRoll',
      borneOff: { white: 0, black: 0 },
      isFirstMove: { white: true, black: true },
    })

    expect(gameState.dice).toBeNull()
  })
})

describe('applyServerMessage LEGAL_MOVES (#8)', () => {
  test('gameStore_onLEGAL_MOVES_replacesMoveList', () => {
    applyServerMessage({
      type: 'LEGAL_MOVES',
      moves: [
        { from: 24, to: 18, pip: 6 },
        { from: 24, to: 19, pip: 5 },
      ],
    })

    expect(gameState.legalMoves).toHaveLength(2)
    expect(gameState.legalMoves[0]).toEqual({ from: 24, to: 18, pip: 6 })
  })

  test('gameStore_onLEGAL_MOVES_emptyArray_clearsLegalMoves', () => {
    // Сначала кладём в стор какие-то ходы.
    applyServerMessage({
      type: 'LEGAL_MOVES',
      moves: [{ from: 24, to: 18, pip: 6 }],
    })
    expect(gameState.legalMoves).toHaveLength(1)

    // Пустой LEGAL_MOVES сбрасывает (= ход пропускается на сервере).
    applyServerMessage({ type: 'LEGAL_MOVES', moves: [] })
    expect(gameState.legalMoves).toEqual([])
  })
})
