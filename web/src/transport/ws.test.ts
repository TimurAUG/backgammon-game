// FRONTEND_SPEC #3 — connect / send (мок WebSocket).
// FRONTEND_SPEC #4 — onMessage парсит через ServerMessage.
// FRONTEND_SPEC #5 — реконнект с экспоненциальным backoff.
// FRONTEND_SPEC #6 — auto-JOIN после open (включая реконнект).
//
// Тесты используют общий MockWebSocket. WSClient получает
// конструктор через DI — в тестах это MockWebSocket, в проде —
// глобальный WebSocket (по умолчанию).

import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

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

describe('WSClient auto-JOIN (#6)', () => {
  test('WSClient_open_autoSendsJoinWithConfiguredCredentials', () => {
    const client = mkClient()
    client.connect()
    MockWebSocket.last().acceptOpen()
    expect(MockWebSocket.last().sent).toEqual([
      '{"type":"JOIN","gameId":"g1","token":"tok"}',
    ])
  })
})

describe('WSClient reconnect (#5)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  test('WSClient_abnormalClose_reconnectsAfter1s', () => {
    const client = mkClient()
    client.connect()
    MockWebSocket.last().acceptOpen()
    MockWebSocket.last().serverClose(1006)

    vi.advanceTimersByTime(999)
    expect(MockWebSocket.instances).toHaveLength(1)

    vi.advanceTimersByTime(1)
    expect(MockWebSocket.instances).toHaveLength(2)
    expect(MockWebSocket.last().url).toBe('ws://localhost:8080/ws')
  })

  test('WSClient_reconnect_backoffDoublesOnFailures', () => {
    const client = mkClient()
    client.connect()

    // первый сокет — open → close → реконнект через 1с
    MockWebSocket.last().acceptOpen()
    MockWebSocket.last().serverClose(1006)
    vi.advanceTimersByTime(1000)
    expect(MockWebSocket.instances).toHaveLength(2)

    // второй — close БЕЗ open → реконнект через 2с
    MockWebSocket.last().serverClose(1006)
    vi.advanceTimersByTime(1999)
    expect(MockWebSocket.instances).toHaveLength(2)
    vi.advanceTimersByTime(1)
    expect(MockWebSocket.instances).toHaveLength(3)

    // третий — снова close без open → 4с
    MockWebSocket.last().serverClose(1006)
    vi.advanceTimersByTime(3999)
    expect(MockWebSocket.instances).toHaveLength(3)
    vi.advanceTimersByTime(1)
    expect(MockWebSocket.instances).toHaveLength(4)
  })

  test('WSClient_reconnect_backoffCappedAt30s', () => {
    const client = mkClient()
    client.connect()

    // Последовательность задержек: 1, 2, 4, 8, 16, 30 (потолок).
    const delays = [1000, 2000, 4000, 8000, 16000, 30000]
    for (let i = 0; i < delays.length; i++) {
      MockWebSocket.last().serverClose(1006)
      vi.advanceTimersByTime(delays[i] as number)
      expect(MockWebSocket.instances).toHaveLength(i + 2)
    }

    // Седьмая попытка — снова 30с (потолок держится).
    MockWebSocket.last().serverClose(1006)
    vi.advanceTimersByTime(29999)
    expect(MockWebSocket.instances).toHaveLength(delays.length + 1)
    vi.advanceTimersByTime(1)
    expect(MockWebSocket.instances).toHaveLength(delays.length + 2)
  })

  test('WSClient_reconnect_resetsBackoffAfterSuccessfulOpen', () => {
    const client = mkClient()
    client.connect()

    // 1й сокет: open → close → реконнект через 1с
    MockWebSocket.last().acceptOpen()
    MockWebSocket.last().serverClose(1006)
    vi.advanceTimersByTime(1000)
    expect(MockWebSocket.instances).toHaveLength(2)

    // 2й сокет: open (успех сбрасывает счётчик) → close → снова 1с, не 2с
    MockWebSocket.last().acceptOpen()
    MockWebSocket.last().serverClose(1006)
    vi.advanceTimersByTime(999)
    expect(MockWebSocket.instances).toHaveLength(2)
    vi.advanceTimersByTime(1)
    expect(MockWebSocket.instances).toHaveLength(3)
  })

  test('WSClient_reconnect_alsoAutoSendsJoinAfterReopen', () => {
    const client = mkClient()
    client.connect()
    MockWebSocket.last().acceptOpen()
    MockWebSocket.last().serverClose(1006)

    vi.advanceTimersByTime(1000)
    const second = MockWebSocket.last()
    second.acceptOpen()

    expect(second.sent).toEqual(['{"type":"JOIN","gameId":"g1","token":"tok"}'])
  })

  test('WSClient_close_doesNotReconnect', () => {
    const client = mkClient()
    client.connect()
    MockWebSocket.last().acceptOpen()
    client.close()

    vi.advanceTimersByTime(60000)
    expect(MockWebSocket.instances).toHaveLength(1)
  })
})

describe('WSClient state changes (#26b)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  test('WSClient_connect_emitsConnecting', () => {
    const client = mkClient()
    const phases: string[] = []
    client.onStateChange((p) => phases.push(p))

    client.connect()

    expect(phases).toEqual(['connecting'])
  })

  test('WSClient_open_emitsConnected', () => {
    const client = mkClient()
    const phases: string[] = []
    client.onStateChange((p) => phases.push(p))

    client.connect()
    MockWebSocket.last().acceptOpen()

    expect(phases).toEqual(['connecting', 'connected'])
  })

  test('WSClient_abnormalClose_emitsReconnecting', () => {
    const client = mkClient()
    const phases: string[] = []
    client.onStateChange((p) => phases.push(p))

    client.connect()
    MockWebSocket.last().acceptOpen()
    MockWebSocket.last().serverClose(1006)

    expect(phases).toEqual(['connecting', 'connected', 'reconnecting'])
  })

  test('WSClient_deliberateClose_doesNotEmitReconnecting', () => {
    const client = mkClient()
    const phases: string[] = []
    client.onStateChange((p) => phases.push(p))

    client.connect()
    MockWebSocket.last().acceptOpen()
    client.close()

    expect(phases).toEqual(['connecting', 'connected'])
  })
})
