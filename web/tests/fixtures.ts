// Фабрики серверных сообщений для тестов. Дефолт — начальная позиция
// длинных нард: 15 белых на пункте 24, 15 чёрных на пункте 12.

import type { ServerMessage } from '../src/protocol/messages'

export type StateMessage = Extract<ServerMessage, { type: 'STATE' }>

export function stateFixture(overrides: Partial<Omit<StateMessage, 'type'>> = {}): StateMessage {
  const board = Array(24).fill(0) as number[]
  board[23] = 15
  board[11] = -15
  return {
    type: 'STATE',
    board,
    turn: 'white',
    status: 'waitingForRoll',
    borneOff: { white: 0, black: 0 },
    isFirstMove: { white: true, black: true },
    allHome: { white: false, black: false },
    ...overrides,
  }
}
