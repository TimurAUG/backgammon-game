// TS-зеркало internal/protocol/messages.go.
// Контракт сообщений — nardy-protocol.

export type ClientMessage =
  | { type: 'JOIN'; gameId: string; token: string }
  | { type: 'ROLL_FOR_FIRST' }
  | { type: 'ROLL' }
  | { type: 'MOVE'; from: number; to: number }
  | { type: 'END_TURN' }
  | { type: 'RESIGN' }

export function serializeClientMessage(msg: ClientMessage): string {
  return JSON.stringify(msg)
}
