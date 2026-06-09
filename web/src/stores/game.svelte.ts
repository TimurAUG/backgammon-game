// gameState — модульный $state-объект, отражающий последний
// серверный снапшот. applyServerMessage — единственная точка
// мутации; UI читает gameState через обычные field accessors,
// Svelte 5 реактивность сам триггерит перерендер.
//
// Все правила игры на сервере; стор только хранит то, что прислал.

import type { Color, Dice, GameStatus, Move, WinKind, ServerMessage } from '../protocol/messages'

export interface GameStoreState {
  board: number[]
  turn: Color
  status: GameStatus
  borneOff: { white: number; black: number }
  isFirstMove: { white: boolean; black: boolean }
  dice: Dice | null
  legalMoves: Move[]
  gameOver: { winner: Color; kind: WinKind } | null
}

function initialGameState(): GameStoreState {
  return {
    board: Array(24).fill(0),
    turn: 'white',
    status: 'waitingForRoll',
    borneOff: { white: 0, black: 0 },
    isFirstMove: { white: true, black: true },
    dice: null,
    legalMoves: [],
    gameOver: null,
  }
}

export const gameState = $state<GameStoreState>(initialGameState())

export function resetGameState(): void {
  Object.assign(gameState, initialGameState())
}

export function applyServerMessage(msg: ServerMessage): void {
  switch (msg.type) {
    case 'STATE':
      gameState.board = msg.board
      gameState.turn = msg.turn
      gameState.status = msg.status
      gameState.borneOff = msg.borneOff
      gameState.isFirstMove = msg.isFirstMove
      gameState.dice = msg.dice ?? null
      break
    case 'LEGAL_MOVES':
      gameState.legalMoves = msg.moves
      break
    case 'GAME_OVER':
      gameState.gameOver = { winner: msg.winner, kind: msg.kind }
      gameState.status = 'finished'
      break
    case 'ERROR':
      // По FRONTEND_SPEC #10 — не меняем gameState. Логирование
      // и нотификация UI — в connectionStore (отдельный слой).
      break
    case 'OPPONENT_JOINED':
    case 'OPPONENT_LEFT':
      // Соперник — connectionStore, не игровой стор.
      break
  }
}
