// REST invite-клиент. Сервер генерит gameId+token (crypto/rand) и отдаёт их
// в теле ответа — клиент больше не придумывает их вручную.
// См. internal/transport/rest/handler.go.

import type { Credentials } from './credentials'

async function postCredentials(url: string): Promise<Credentials> {
  const resp = await fetch(url, { method: 'POST' })
  if (!resp.ok) {
    throw new Error(`${url} → HTTP ${resp.status}`)
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

// createGame создаёт новую игру; возвращает креды создателя (слот White).
export function createGame(): Promise<Credentials> {
  return postCredentials('/api/games')
}

// joinGame входит в существующую игру по id; возвращает креды второго игрока.
export function joinGame(gameId: string): Promise<Credentials> {
  return postCredentials(`/api/games/${encodeURIComponent(gameId)}/join`)
}
