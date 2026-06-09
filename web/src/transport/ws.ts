// WSClient — клиент WebSocket-канала. Сериализует ClientMessage,
// парсит ServerMessage, нотифицирует подписчиков, переподключается
// с экспоненциальным backoff, после каждого open автоматически
// шлёт JOIN с конфиг-кредами.
//
// WebSocket-конструктор передаётся через DI (опционально); в проде
// используется глобальный WebSocket, в тестах — MockWebSocket.

import {
  parseServerMessage,
  serializeClientMessage,
  type ClientMessage,
  type ServerMessage,
} from '../protocol/messages'

export interface WSConnection {
  readyState: number
  onopen: ((ev: Event) => void) | null
  onclose: ((ev: CloseEvent) => void) | null
  onmessage: ((ev: MessageEvent) => void) | null
  onerror: ((ev: Event) => void) | null
  send(data: string): void
  close(code?: number, reason?: string): void
}

export type WSConnectionCtor = new (url: string) => WSConnection

export interface WSClientConfig {
  url: string
  gameId: string
  token: string
}

// Транспортная фаза сокета. App мапит её в стор connection (#26b/#26c);
// WSClient про стор не знает — отсюда собственный тип, а не ConnectionState.
export type WSPhase = 'connecting' | 'connected' | 'reconnecting'

const OPEN = 1
const BACKOFF_BASE_MS = 1000
const BACKOFF_CAP_MS = 30000

export class WSClient {
  private socket: WSConnection | null = null
  private listeners: Array<(msg: ServerMessage) => void> = []
  private stateListeners: Array<(phase: WSPhase) => void> = []
  private stopped = false
  private attempts = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null

  constructor(
    private readonly config: WSClientConfig,
    private readonly wsCtor: WSConnectionCtor = WebSocket as unknown as WSConnectionCtor,
  ) {}

  connect(): void {
    this.stopped = false
    this.attempts = 0
    this.emitState('connecting')
    this.openSocket()
  }

  send(msg: ClientMessage): void {
    if (!this.socket || this.socket.readyState !== OPEN) {
      throw new Error('WSClient.send: socket is not open')
    }
    this.socket.send(serializeClientMessage(msg))
  }

  onMessage(cb: (msg: ServerMessage) => void): () => void {
    this.listeners.push(cb)
    return () => {
      this.listeners = this.listeners.filter((l) => l !== cb)
    }
  }

  onStateChange(cb: (phase: WSPhase) => void): () => void {
    this.stateListeners.push(cb)
    return () => {
      this.stateListeners = this.stateListeners.filter((l) => l !== cb)
    }
  }

  close(): void {
    this.stopped = true
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.socket?.close()
  }

  private openSocket(): void {
    const socket = new this.wsCtor(this.config.url)
    socket.onopen = () => {
      this.attempts = 0
      this.emitState('connected')
      socket.send(
        serializeClientMessage({
          type: 'JOIN',
          gameId: this.config.gameId,
          token: this.config.token,
        }),
      )
    }
    socket.onclose = () => {
      if (this.stopped) return
      this.emitState('reconnecting')
      this.scheduleReconnect()
    }
    socket.onmessage = (ev) => this.handleMessage(ev.data)
    this.socket = socket
  }

  private scheduleReconnect(): void {
    const delay = Math.min(BACKOFF_BASE_MS * Math.pow(2, this.attempts), BACKOFF_CAP_MS)
    this.attempts++
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.openSocket()
    }, delay)
  }

  private emitState(phase: WSPhase): void {
    for (const cb of this.stateListeners) cb(phase)
  }

  private handleMessage(data: unknown): void {
    if (typeof data !== 'string') return
    let msg: ServerMessage
    try {
      msg = parseServerMessage(data)
    } catch {
      return
    }
    for (const cb of this.listeners) cb(msg)
  }
}
