// FRONTEND_SPEC #28 — lib/api.ts: REST invite-клиент. createGame() POST'ит
// /api/games, joinGame(id) POST'ит /api/games/{id}/join; оба возвращают
// Credentials из тела ответа. fetch замокан.

import { afterEach, describe, expect, test, vi } from 'vitest'

import { createGame, joinGame } from './api'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('api invite (#28)', () => {
  test('createGame_postsToApiGames_returnsCredentials', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ gameId: 'g-1', token: 'tok-1' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const creds = await createGame()

    expect(fetchMock).toHaveBeenCalledWith('/api/games', { method: 'POST' })
    expect(creds).toEqual({ gameId: 'g-1', token: 'tok-1' })
  })

  test('joinGame_postsToJoinEndpoint_returnsCredentials', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ gameId: 'g-1', token: 'tok-2' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const creds = await joinGame('g-1')

    expect(fetchMock).toHaveBeenCalledWith('/api/games/g-1/join', { method: 'POST' })
    expect(creds).toEqual({ gameId: 'g-1', token: 'tok-2' })
  })

  test('createGame_onHttpError_throws', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))

    await expect(createGame()).rejects.toThrow()
  })
})
