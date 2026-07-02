// FRONTEND_SPEC #7 — STATE обновляет board/turn/dice/borneOff/status/isFirstMove.
// FRONTEND_SPEC #8 — LEGAL_MOVES обновляет legalMoves (пустой = ход пропускается).
// FRONTEND_SPEC #9 — GAME_OVER ставит gameOver и status='finished'.
// FRONTEND_SPEC #10 — ERROR не меняет state (логирование на стороне connectionStore).
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
      allHome: { white: false, black: false },
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
      allHome: { white: false, black: false },
    })

    expect(gameState.dice).toBeNull()
  })

  test('gameStore_initial_allHomeIsFalse', () => {
    // До первого STATE никто не в фазе сброса — плашка счётчика скрыта.
    expect(gameState.allHome).toEqual({ white: false, black: false })
  })

  test('gameStore_onSTATE_updatesAllHome', () => {
    applyServerMessage({
      type: 'STATE',
      board: Array(24).fill(0),
      turn: 'white',
      status: 'waitingForMove',
      borneOff: { white: 4, black: 0 },
      isFirstMove: { white: false, black: false },
      allHome: { white: true, black: false },
    })

    expect(gameState.allHome).toEqual({ white: true, black: false })
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

  // FRONTEND_SPEC #48 — LEGAL_MOVES несёт reach (составные цели одной шашки).
  test('gameStore_onLEGAL_MOVES_storesReachChains', () => {
    applyServerMessage({
      type: 'LEGAL_MOVES',
      moves: [{ from: 13, to: 11, pip: 2 }],
      reach: [
        { from: 13, path: [11], pips: [2] },
        { from: 13, path: [11, 7], pips: [2, 4] },
      ],
    })

    expect(gameState.reach).toHaveLength(2)
    expect(gameState.reach[1]).toEqual({ from: 13, path: [11, 7], pips: [2, 4] })
  })

  test('gameStore_onLEGAL_MOVES_withoutReach_defaultsToEmpty', () => {
    applyServerMessage({
      type: 'LEGAL_MOVES',
      moves: [{ from: 13, to: 11, pip: 2 }],
      reach: [{ from: 13, path: [11, 7], pips: [2, 4] }],
    })
    expect(gameState.reach).toHaveLength(1)

    // LEGAL_MOVES без reach → стор обнуляет reach, не тащит прошлый набор.
    applyServerMessage({ type: 'LEGAL_MOVES', moves: [{ from: 24, to: 18, pip: 6 }] })
    expect(gameState.reach).toEqual([])
  })
})

describe('applyServerMessage GAME_OVER (#9)', () => {
  test('gameStore_onGAME_OVER_setsWinnerKindAndFinishedStatus', () => {
    applyServerMessage({
      type: 'GAME_OVER',
      winner: 'black',
      kind: 'koks',
    })

    expect(gameState.gameOver).toEqual({ winner: 'black', kind: 'koks' })
    expect(gameState.status).toBe('finished')
  })

  test('gameStore_onGAME_OVER_doesNotTouchBoardOrLegalMoves', () => {
    const board = Array(24).fill(0)
    board[0] = 3
    applyServerMessage({
      type: 'STATE',
      board,
      turn: 'white',
      status: 'waitingForMove',
      borneOff: { white: 12, black: 0 },
      isFirstMove: { white: false, black: false },
      allHome: { white: true, black: false },
    })
    applyServerMessage({
      type: 'LEGAL_MOVES',
      moves: [{ from: 1, to: 0, pip: 1 }],
    })

    applyServerMessage({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })

    expect(gameState.board).toEqual(board)
    expect(gameState.borneOff).toEqual({ white: 12, black: 0 })
    expect(gameState.legalMoves).toEqual([{ from: 1, to: 0, pip: 1 }])
    expect(gameState.gameOver).toEqual({ winner: 'white', kind: 'mars' })
    expect(gameState.status).toBe('finished')
  })
})

describe('applyServerMessage JOINED (web#24-prep)', () => {
  test('gameStore_initial_myColorIsNull', () => {
    expect(gameState.myColor).toBeNull()
  })

  test('gameStore_onJOINED_setsMyColor', () => {
    applyServerMessage({ type: 'JOINED', color: 'black' })
    expect(gameState.myColor).toBe('black')
  })

  test('gameStore_reset_clearsMyColor', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    resetGameState()
    expect(gameState.myColor).toBeNull()
  })
})

describe('applyServerMessage ERROR (#10)', () => {
  test('gameStore_onERROR_doesNotMutateAnyField', () => {
    const board = Array(24).fill(0)
    board[23] = 15
    applyServerMessage({
      type: 'STATE',
      board,
      turn: 'white',
      status: 'waitingForMove',
      borneOff: { white: 1, black: 2 },
      isFirstMove: { white: false, black: true },
      allHome: { white: false, black: false },
      dice: { a: 3, b: 5, isDouble: false, remaining: [3, 5] },
    })
    applyServerMessage({
      type: 'LEGAL_MOVES',
      moves: [{ from: 24, to: 21, pip: 3 }],
    })
    const snapshot = JSON.parse(JSON.stringify(gameState))

    applyServerMessage({
      type: 'ERROR',
      code: 'RULE_OF_SIX',
      message: 'финальный блок 6+ при пустом доме соперника',
    })

    expect(JSON.parse(JSON.stringify(gameState))).toEqual(snapshot)
  })
})

describe('applyServerMessage started flag (#34b)', () => {
  // started=true только после первого STATE: до него (между JOINED и STATE)
  // стор хранит initial-снапшот, по которому нельзя судить «ожидается мой
  // бросок» — иначе ложный звон при возврате в игру в чужой ход.
  function anyState() {
    return {
      type: 'STATE' as const,
      board: Array(24).fill(0),
      turn: 'white' as const,
      status: 'waitingForRoll' as const,
      borneOff: { white: 0, black: 0 },
      isFirstMove: { white: true, black: true },
      allHome: { white: false, black: false } as const,
    }
  }

  test('gameStore_initial_startedIsFalse', () => {
    expect(gameState.started).toBe(false)
  })

  test('gameStore_onSTATE_setsStartedTrue', () => {
    applyServerMessage(anyState())

    expect(gameState.started).toBe(true)
  })

  test('gameStore_reset_clearsStarted', () => {
    applyServerMessage(anyState())

    resetGameState()

    expect(gameState.started).toBe(false)
  })
})

describe('applyServerMessage FIRST_ROLL (#2)', () => {
  test('gameStore_onFIRST_ROLL_storesValues', () => {
    applyServerMessage({ type: 'FIRST_ROLL', firstRoll: { white: 5, black: 3 } })

    expect(gameState.firstRoll).toEqual({ white: 5, black: 3 })
  })

  test('gameStore_reset_clearsFirstRoll', () => {
    applyServerMessage({ type: 'FIRST_ROLL', firstRoll: { white: 5, black: 3 } })

    resetGameState()

    expect(gameState.firstRoll).toBeNull()
  })
})
