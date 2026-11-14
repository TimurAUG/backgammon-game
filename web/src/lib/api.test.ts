// FRONTEND_SPEC #28 — lib/api.ts: REST invite-клиент. createGame() POST'ит
// /api/games, joinGame(id) POST'ит /api/games/{id}/join; оба возвращают
// Credentials из тела ответа. fetch замокан.
// Card #4 — register/login/me/logout + credentials: 'include' на create/join.

import { afterEach, describe, expect, test, vi } from 'vitest'

import { createGame, joinGame, login, logout, me, register } from './api'

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

    expect(fetchMock).toHaveBeenCalledWith('/api/games', {
      method: 'POST',
      credentials: 'include',
    })
    expect(creds).toEqual({ gameId: 'g-1', token: 'tok-1' })
  })

  test('joinGame_postsToJoinEndpoint_returnsCredentials', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ gameId: 'g-1', token: 'tok-2' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const creds = await joinGame('g-1')

    expect(fetchMock).toHaveBeenCalledWith('/api/games/g-1/join', {
      method: 'POST',
      credentials: 'include',
    })
    expect(creds).toEqual({ gameId: 'g-1', token: 'tok-2' })
  })

  test('createGame_includesCredentialsInclude', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ gameId: 'g-1', token: 'tok-1' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    await createGame()

    const createInit = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(createInit).toMatchObject({ credentials: 'include' })
  })

  test('joinGame_includesCredentialsInclude', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ gameId: 'g-1', token: 'tok-2' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    await joinGame('g-x')

    const joinInit = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(joinInit).toMatchObject({ credentials: 'include' })
  })

  test('createGame_onHttpError_throws', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))

    await expect(createGame()).rejects.toThrow()
  })
})

describe('api auth (#4)', () => {
  test('register_postsJson_withCredentialsInclude', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ nick: 'Ann' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const result = await register('Ann', 'password1')

    expect(fetchMock).toHaveBeenCalledWith('/api/register', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nick: 'Ann', password: 'password1' }),
    })
    expect(result).toEqual({ nick: 'Ann' })
  })

  test('login_postsJson_withCredentialsInclude', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ nick: 'Bob' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const result = await login('Bob', 'password1')

    expect(fetchMock).toHaveBeenCalledWith('/api/login', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nick: 'Bob', password: 'password1' }),
    })
    expect(result).toEqual({ nick: 'Bob' })
  })

  test('me_getWithCredentialsInclude', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        nick: 'Carla',
        played: 0,
        wins: 0,
        losses: 0,
        win_percent: 0,
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const result = await me()

    expect(fetchMock).toHaveBeenCalledWith('/api/me', {
      method: 'GET',
      credentials: 'include',
    })
    expect(result).toEqual({
      nick: 'Carla',
      played: 0,
      wins: 0,
      losses: 0,
      win_percent: 0,
    })
  })

  // Card #23 — GET /api/me returns nick + four stats ints (SPEC #24).
  test('me_parsesPlayedWinsLossesWinPercent', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        nick: 'Carla',
        played: 42,
        wins: 27,
        losses: 15,
        win_percent: 64,
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const result = await me()

    expect(result).toEqual({
      nick: 'Carla',
      played: 42,
      wins: 27,
      losses: 15,
      win_percent: 64,
    })
  })

  test('me_missingWinPercent_throws', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ nick: 'Carla', played: 1, wins: 1, losses: 0 }),
      }),
    )

    await expect(me()).rejects.toThrow(/unexpected response shape/)
  })

  test('me_nonIntegerPlayed_throws', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          nick: 'Carla',
          played: 1.5,
          wins: 1,
          losses: 0,
          win_percent: 100,
        }),
      }),
    )

    await expect(me()).rejects.toThrow(/unexpected response shape/)
  })

  test('logout_postsWithCredentialsInclude', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', fetchMock)

    await logout()

    expect(fetchMock).toHaveBeenCalledWith('/api/logout', {
      method: 'POST',
      credentials: 'include',
    })
  })

  test('register_on409_throwsWithStatus', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 409 }))

    await expect(register('Ann', 'password1')).rejects.toMatchObject({ status: 409 })
  })

  test('login_on401_throwsWithStatus', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 401 }))

    await expect(login('Ann', 'bad')).rejects.toMatchObject({ status: 401 })
  })

  test('me_on401_throwsWithStatus', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 401 }))

    await expect(me()).rejects.toMatchObject({ status: 401 })
  })
})
