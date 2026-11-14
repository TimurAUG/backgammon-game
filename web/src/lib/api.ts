// REST invite-клиент. Сервер генерит gameId+token (crypto/rand) и отдаёт их
// в теле ответа — клиент больше не придумывает их вручную.
// Auth (card #4): register/login/me/logout + credentials: 'include'.
// См. internal/transport/rest/handler.go.

import type { Credentials } from './credentials'

export interface NickResponse {
  nick: string
}

/** Successful GET /api/me — nick plus account game stats (card #23 / SPEC #24). */
export interface MeResponse {
  nick: string
  played: number
  wins: number
  losses: number
  win_percent: number
}

export class ApiError extends Error {
  readonly status: number

  constructor(url: string, status: number) {
    super(`${url} → HTTP ${status}`)
    this.name = 'ApiError'
    this.status = status
  }
}

async function apiFetch(url: string, init: RequestInit = {}): Promise<Response> {
  return fetch(url, { ...init, credentials: 'include' })
}

async function postCredentials(url: string): Promise<Credentials> {
  const resp = await apiFetch(url, { method: 'POST' })
  if (!resp.ok) {
    throw new ApiError(url, resp.status)
  }
  const data: unknown = await resp.json()
  if (
    typeof data !== 'object' ||
    data === null ||
    typeof (data as Credentials).gameId !== 'string' ||
    typeof (data as Credentials).token !== 'string'
  ) {
    throw new Error(`${url}: unexpected response shape`)
  }
  const { gameId, token } = data as Credentials
  return { gameId, token }
}

function parseNick(url: string, data: unknown): NickResponse {
  if (
    typeof data !== 'object' ||
    data === null ||
    typeof (data as NickResponse).nick !== 'string'
  ) {
    throw new Error(`${url}: unexpected response shape`)
  }
  return { nick: (data as NickResponse).nick }
}

function isNonNegInt(value: unknown): value is number {
  return (
    typeof value === 'number' &&
    Number.isInteger(value) &&
    Number.isFinite(value) &&
    value >= 0
  )
}

function parseMe(url: string, data: unknown): MeResponse {
  if (
    typeof data !== 'object' ||
    data === null ||
    typeof (data as MeResponse).nick !== 'string' ||
    !isNonNegInt((data as MeResponse).played) ||
    !isNonNegInt((data as MeResponse).wins) ||
    !isNonNegInt((data as MeResponse).losses) ||
    !isNonNegInt((data as MeResponse).win_percent)
  ) {
    throw new Error(`${url}: unexpected response shape`)
  }
  const body = data as MeResponse
  return {
    nick: body.nick,
    played: body.played,
    wins: body.wins,
    losses: body.losses,
    win_percent: body.win_percent,
  }
}

async function postAuth(url: string, nick: string, password: string): Promise<NickResponse> {
  const resp = await apiFetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ nick, password }),
  })
  if (!resp.ok) {
    throw new ApiError(url, resp.status)
  }
  return parseNick(url, await resp.json())
}

// createGame создаёт новую игру; возвращает креды создателя (слот White).
export function createGame(): Promise<Credentials> {
  return postCredentials('/api/games')
}

// joinGame входит в существующую игру по id; возвращает креды второго игрока.
export function joinGame(gameId: string): Promise<Credentials> {
  return postCredentials(`/api/games/${encodeURIComponent(gameId)}/join`)
}

export function register(nick: string, password: string): Promise<NickResponse> {
  return postAuth('/api/register', nick, password)
}

export function login(nick: string, password: string): Promise<NickResponse> {
  return postAuth('/api/login', nick, password)
}

export async function me(): Promise<MeResponse> {
  const url = '/api/me'
  const resp = await apiFetch(url, { method: 'GET' })
  if (!resp.ok) {
    throw new ApiError(url, resp.status)
  }
  return parseMe(url, await resp.json())
}

export async function logout(): Promise<void> {
  const url = '/api/logout'
  const resp = await apiFetch(url, { method: 'POST' })
  if (!resp.ok) {
    throw new ApiError(url, resp.status)
  }
}
