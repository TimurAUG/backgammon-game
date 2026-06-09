// FRONTEND_SPEC #3 — connect / send (мок WebSocket).
// FRONTEND_SPEC #4 — onMessage парсит через ServerMessage.
//
// Тесты используют общий MockWebSocket. WSClient получает
// конструктор через DI — в тестах это MockWebSocket, в проде —
// глобальный WebSocket (по умолчанию).

import { beforeEach, describe, expect, test } from 'vitest'

import type { ServerMessage } from '../protocol/messages'

import { MockWebSocket } from '../../tests/mockWebSocket'
import { WSClient } from './ws'

beforeEach(() => {
  MockWebSocket.reset()
})

function mkClient(): WSClient {
  return new WSClient(
    { url: 'ws://localhost:8080/ws', gameId: 'g1', token: 'tok' },
    MockWebSocket,
  )
}

describe('WSClient connect/send (#3)', () => {
  test('WSClient_connect_opensSocketAtConfiguredUrl', () => {
    const client = mkClient()
    client.connect()
    expect(MockWebSocket.instances).toHaveLength(1)
    expect(MockWebSocket.last().url).toBe('ws://localhost:8080/ws')
  })

  test('WSClient_send_afterOpen_serializesClientMessage', () => {
    const client = mkClient()
    client.connect()
    MockWebSocket.last().acceptOpen()
    // отбрасываем всё, что WSClient мог отправить автоматически
    // (этой парой #3+#4 auto-JOIN ещё не реализован, но тесты
    // готовы пережить его появление в #6).
    MockWebSocket.last().sent.length = 0

    client.send({ type: 'MOVE', from: 24, to: 18 })

    expect(MockWebSocket.last().sent).toEqual(['{"type":"MOVE","from":24,"to":18}'])
  })

  test('WSClient_send_beforeOpen_throws', () => {
    const client = mkClient()
    client.connect()
    expect(() => client.send({ type: 'ROLL' })).toThrow(/not open/i)
  })
})

describe('WSClient onMessage (#4)', () => {
  test('WSClient_onMessage_callbackReceivesParsedServerMessage', () => {
    const client = mkClient()
    const received: ServerMessage[] = []
    client.onMessage((msg) => received.push(msg))

    client.connect()
    MockWebSocket.last().acceptOpen()
    MockWebSocket.last().receive({
      type: 'STATE',
      board: Array(24).fill(0),
      turn: 'white',
      status: 'waitingForRoll',
      borneOff: { white: 0, black: 0 },
      isFirstMove: { white: true, black: true },
    })

    expect(received).toHaveLength(1)
    const first = received[0]
    expect(first?.type).toBe('STATE')
    if (first?.type === 'STATE') {
      expect(first.turn).toBe('white')
      expect(first.borneOff).toEqual({ white: 0, black: 0 })
    }
  })

  test('WSClient_onMessage_unsubscribe_stopsDelivery', () => {
    const client = mkClient()
    const received: ServerMessage[] = []
    const off = client.onMessage((msg) => received.push(msg))

    client.connect()
    MockWebSocket.last().acceptOpen()
    off()
    MockWebSocket.last().receive('{"type":"OPPONENT_LEFT"}')

    expect(received).toHaveLength(0)
  })

  test('WSClient_onMessage_invalidPayload_doesNotInvokeCallback', () => {
    const client = mkClient()
    const received: ServerMessage[] = []
    client.onMessage((msg) => received.push(msg))

    client.connect()
    MockWebSocket.last().acceptOpen()
    MockWebSocket.last().receive('not-json-at-all')
    MockWebSocket.last().receive('{"board":[]}') // нет поля type

    expect(received).toHaveLength(0)
  })
})
