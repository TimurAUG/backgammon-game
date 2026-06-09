// Переиспользуемый мок WebSocket для тестов /web/src/transport/.
// Соответствует структурному типу WSConnection из transport/ws.ts.

export class MockWebSocket {
  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSING = 2
  static readonly CLOSED = 3

  static instances: MockWebSocket[] = []

  static reset(): void {
    MockWebSocket.instances = []
  }

  static last(): MockWebSocket {
    const ws = MockWebSocket.instances.at(-1)
    if (!ws) throw new Error('MockWebSocket: no instance created yet')
    return ws
  }

  readyState = MockWebSocket.CONNECTING
  sent: string[] = []

  onopen: ((ev: Event) => void) | null = null
  onclose: ((ev: CloseEvent) => void) | null = null
  onerror: ((ev: Event) => void) | null = null
  onmessage: ((ev: MessageEvent) => void) | null = null

  constructor(public url: string) {
    MockWebSocket.instances.push(this)
  }

  send(data: string): void {
    this.sent.push(data)
  }

  close(code = 1000, _reason?: string): void {
    if (this.readyState === MockWebSocket.CLOSED) return
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.(new CloseEvent('close', { code, wasClean: true }))
  }

  // Эмулируют действия сервера для теста.

  acceptOpen(): void {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.(new Event('open'))
  }

  receive(data: string | object): void {
    const payload = typeof data === 'string' ? data : JSON.stringify(data)
    this.onmessage?.(new MessageEvent('message', { data: payload }))
  }

  serverClose(code = 1006): void {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.(new CloseEvent('close', { code, wasClean: false }))
  }
}
