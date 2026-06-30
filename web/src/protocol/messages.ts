// TS-зеркало internal/protocol/messages.go.
// Контракт сообщений — nardy-protocol.

export type ClientMessage =
  | { type: 'JOIN'; gameId: string; token: string }
  | { type: 'ROLL_FOR_FIRST' }
  | { type: 'ROLL' }
  | { type: 'MOVE'; from: number; to: number }
  | { type: 'END_TURN' }
  | { type: 'RESIGN' }
  | { type: 'CHAT'; text: string }

export function serializeClientMessage(msg: ClientMessage): string {
  return JSON.stringify(msg)
}

export type Color = 'white' | 'black'
export type GameStatus = 'waitingForRoll' | 'waitingForMove' | 'finished'
export type WinKind = 'oin' | 'mars' | 'koks'

export type ErrorCode =
  | 'INVALID_MOVE'
  | 'HEAD_RULE'
  | 'RULE_OF_SIX'
  | 'MUST_USE_LARGER'
  | 'MUST_USE_PIP'
  | 'NOT_YOUR_TURN'
  | 'INVALID_STATE'
  | 'GAME_NOT_FOUND'
  | 'UNAUTHORIZED'
  | 'ROOM_FULL'

export interface Dice {
  a: number
  b: number
  isDouble: boolean
  remaining: number[]
}

export interface Move {
  from: number
  to: number
  pip: number
}

// ReachMove — достижимая цель ОДНОЙ шашки, в том числе составным ходом
// несколькими кубиками. Дополняет Move (одиночные шаги), не заменяет.
// path — пункты-остановки по порядку (каждый — цель отдельного MOVE),
// последний = финальная цель; pips — кубик на каждый шаг (len == path.length).
// Дистанция = сумма pips, число кубиков = path.length. Выкид сюда не входит.
// Зеркало protocol.ReachPayload на бэке.
export interface ReachMove {
  from: number
  path: number[]
  pips: number[]
}

// ChatMessage — одно сообщение чата партии. sender — цвет автора (для CHAT и
// элементов CHAT_HISTORY). Зеркало protocol.ChatPayload на бэке.
export interface ChatMessage {
  sender: Color
  text: string
}

export type ServerMessage =
  | { type: 'JOINED'; color: Color }
  | {
      type: 'STATE'
      board: number[]
      turn: Color
      status: GameStatus
      borneOff: { white: number; black: number }
      isFirstMove: { white: boolean; black: boolean }
      dice?: Dice
    }
  | { type: 'LEGAL_MOVES'; moves: Move[]; reach?: ReachMove[] }
  | { type: 'TURN_SKIPPED'; color: Color; dice: Dice }
  | { type: 'FIRST_ROLL'; firstRoll: { white: number; black: number } }
  | { type: 'OPPONENT_JOINED'; name?: string }
  | { type: 'OPPONENT_LEFT' }
  | { type: 'GAME_OVER'; winner: Color; kind: WinKind }
  | { type: 'CHAT'; sender: Color; text: string }
  | { type: 'CHAT_HISTORY'; chat: ChatMessage[] }
  | { type: 'ERROR'; code: ErrorCode; message: string }

export function parseServerMessage(raw: string): ServerMessage {
  const obj: unknown = JSON.parse(raw)
  if (
    typeof obj !== 'object' ||
    obj === null ||
    !('type' in obj) ||
    typeof (obj as { type: unknown }).type !== 'string'
  ) {
    throw new Error('ServerMessage: expected object with string `type` field')
  }
  return obj as ServerMessage
}
