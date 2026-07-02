// gameState — модульный $state-объект, отражающий последний
// серверный снапшот. applyServerMessage — единственная точка
// мутации; UI читает gameState через обычные field accessors,
// Svelte 5 реактивность сам триггерит перерендер.
//
// Все правила игры на сервере; стор только хранит то, что прислал.

import type {
  Color,
  Dice,
  GameStatus,
  Move,
  ReachMove,
  WinKind,
  ServerMessage,
} from '../protocol/messages'

export interface GameStoreState {
  board: number[]
  turn: Color
  status: GameStatus
  borneOff: { white: number; black: number }
  isFirstMove: { white: boolean; black: boolean }
  // Все шашки цвета в доме (фаза сброса) — приходит из STATE (домен AllInHome).
  // Клиент показывает счётчик оставшихся к сбросу только когда allHome[c].
  allHome: { white: boolean; black: boolean }
  dice: Dice | null
  legalMoves: Move[]
  // Достижимые цели для подсветки прогресса хода (составные ходы одной шашкой).
  // Дополняет legalMoves; приходит в LEGAL_MOVES.reach. Пусто, если сервер не
  // прислал (старый сервер / нет ходов).
  reach: ReachMove[]
  gameOver: { winner: Color; kind: WinKind } | null
  myColor: Color | null
  firstRoll: { white: number; black: number } | null
  // true после первого STATE — отличает реальный снапшот от initial-заглушки
  // (нужно детектору «ожидается мой бросок», см. Game.svelte #34b).
  started: boolean
}

function initialGameState(): GameStoreState {
  return {
    board: Array(24).fill(0),
    turn: 'white',
    status: 'waitingForRoll',
    borneOff: { white: 0, black: 0 },
    isFirstMove: { white: true, black: true },
    allHome: { white: false, black: false },
    dice: null,
    legalMoves: [],
    reach: [],
    gameOver: null,
    myColor: null,
    firstRoll: null,
    started: false,
  }
}

export const gameState = $state<GameStoreState>(initialGameState())

export function resetGameState(): void {
  Object.assign(gameState, initialGameState())
}

export function applyServerMessage(msg: ServerMessage): void {
  switch (msg.type) {
    case 'JOINED':
      gameState.myColor = msg.color
      break
    case 'STATE':
      gameState.board = msg.board
      gameState.turn = msg.turn
      gameState.status = msg.status
      gameState.borneOff = msg.borneOff
      gameState.isFirstMove = msg.isFirstMove
      gameState.allHome = msg.allHome
      gameState.dice = msg.dice ?? null
      gameState.started = true
      break
    case 'LEGAL_MOVES':
      gameState.legalMoves = msg.moves
      gameState.reach = msg.reach ?? []
      break
    case 'FIRST_ROLL':
      gameState.firstRoll = msg.firstRoll
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
