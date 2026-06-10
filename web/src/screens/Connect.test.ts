// FRONTEND_SPEC #23 — Connect.svelte: форма gameId+token, сохранение
// в localStorage, колбек onConnect с введёнными кредами.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { createGame, joinGame } from '../lib/api'
import { loadCredentials } from '../lib/credentials'

import Connect from './Connect.svelte'

vi.mock('../lib/api', () => ({
  createGame: vi.fn(),
  joinGame: vi.fn(),
}))

beforeEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
})

async function fillAndSubmit(gameId: string, token: string): Promise<void> {
  await fireEvent.input(screen.getByLabelText('ID игры'), { target: { value: gameId } })
  await fireEvent.input(screen.getByLabelText('Токен'), { target: { value: token } })
  await fireEvent.click(screen.getByRole('button', { name: 'Подключиться' }))
}

describe('Connect form (#23)', () => {
  test('Connect_renders_formFields', () => {
    render(Connect, { props: { onConnect: vi.fn() } })
    expect(screen.getByLabelText('ID игры')).toBeInTheDocument()
    expect(screen.getByLabelText('Токен')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Подключиться' })).toBeInTheDocument()
  })

  test('Connect_submitWithFilledFields_callsOnConnect', async () => {
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })

    await fillAndSubmit('g-42', 'tok-abc')

    expect(onConnect).toHaveBeenCalledExactlyOnceWith({ gameId: 'g-42', token: 'tok-abc' })
  })

  test('Connect_submitWithFilledFields_persistsToLocalStorage', async () => {
    render(Connect, { props: { onConnect: vi.fn() } })

    await fillAndSubmit('g-42', 'tok-abc')

    expect(loadCredentials()).toEqual({ gameId: 'g-42', token: 'tok-abc' })
  })

  test('Connect_submitWithEmptyGameId_doesNotCallOnConnect', async () => {
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })

    await fillAndSubmit('   ', 'tok-abc')

    expect(onConnect).not.toHaveBeenCalled()
  })
})

describe('Connect invite (#29)', () => {
  test('Connect_createGame_rendersInviteLinkWithGameId', async () => {
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    render(Connect, { props: { onConnect: vi.fn() } })

    await fireEvent.click(screen.getByTestId('create-game'))

    const link = (await screen.findByTestId('invite-link')) as HTMLInputElement
    expect(link.value).toContain('?game=g-new')
  })

  test('Connect_createGame_persistsCredentials', async () => {
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    render(Connect, { props: { onConnect: vi.fn() } })

    await fireEvent.click(screen.getByTestId('create-game'))
    await screen.findByTestId('invite-link')

    expect(loadCredentials()).toEqual({ gameId: 'g-new', token: 'tok' })
  })

  test('Connect_afterCreate_enterGame_callsOnConnect', async () => {
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })

    await fireEvent.click(screen.getByTestId('create-game'))
    await fireEvent.click(await screen.findByTestId('enter-game'))

    expect(onConnect).toHaveBeenCalledWith({ gameId: 'g-new', token: 'tok' })
  })

  test('Connect_inviteGameId_join_callsJoinGameAndOnConnect', async () => {
    vi.mocked(joinGame).mockResolvedValue({ gameId: 'g-inv', token: 'tok2' })
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect, inviteGameId: 'g-inv' } })

    await fireEvent.click(screen.getByTestId('join-invite'))

    expect(joinGame).toHaveBeenCalledWith('g-inv')
    await vi.waitFor(() =>
      expect(onConnect).toHaveBeenCalledWith({ gameId: 'g-inv', token: 'tok2' }),
    )
  })
})
