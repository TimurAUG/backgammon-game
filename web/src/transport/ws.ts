// WSClient — клиент WebSocket-канала. Сериализует ClientMessage,
// парсит ServerMessage, нотифицирует подписчиков.
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

const OPEN = 1

export class WSClient {
  private socket: WSConnection | null = null
  private listeners: Array<(msg: ServerMessage) => void> = []

  constructor(
    private readonly config: WSClientConfig,
    private readonly wsCtor: WSConnectionCtor = WebSocket as unknown as WSConnectionCtor,
  ) {}

  connect(): void {
    const socket = new this.wsCtor(this.config.url)
    socket.onmessage = (ev) => this.handleMessage(ev.data)
    this.socket = socket
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
