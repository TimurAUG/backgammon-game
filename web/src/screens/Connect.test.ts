// FRONTEND_SPEC #23 / card #11 — Connect.svelte: create + invite only
// (manual gameId+token form removed). Card #4 — nick/password Log in /
// Register + me-on-mount.

import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { createGame, joinGame, login, logout, me, register } from '../lib/api'
import { loadCredentials, saveCredentials } from '../lib/credentials'

import Connect from './Connect.svelte'

vi.mock('../lib/api', () => ({
  createGame: vi.fn(),
  joinGame: vi.fn(),
  register: vi.fn(),
  login: vi.fn(),
  me: vi.fn(),
  logout: vi.fn(),
}))

beforeEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
  vi.mocked(me).mockRejectedValue(Object.assign(new Error('unauthorized'), { status: 401 }))
})

describe('Connect default (#11 / #23)', () => {
  test('Connect_default_showsCreateGameWithoutManualEntry', async () => {
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    expect(screen.getByTestId('create-game')).toBeInTheDocument()
    expect(screen.queryByText('или войти вручную')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Подключиться' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Токен')).not.toBeInTheDocument()
  })
})

describe('Connect invite (#29)', () => {
  test('Connect_createGame_rendersInviteLinkWithGameId', async () => {
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.click(screen.getByTestId('create-game'))

    const link = (await screen.findByTestId('invite-link')) as HTMLInputElement
    expect(link.value).toContain('?game=g-new')
  })

  test('Connect_createGame_persistsCredentials', async () => {
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.click(screen.getByTestId('create-game'))
    await screen.findByTestId('invite-link')

    expect(loadCredentials()).toEqual({ gameId: 'g-new', token: 'tok' })
  })

  test('Connect_afterCreate_enterGame_callsOnConnect', async () => {
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.click(screen.getByTestId('create-game'))
    await fireEvent.click(await screen.findByTestId('enter-game'))

    expect(onConnect).toHaveBeenCalledWith({ gameId: 'g-new', token: 'tok' })
  })

  test('Connect_inviteGameId_join_callsJoinGameAndOnConnect', async () => {
    vi.mocked(joinGame).mockResolvedValue({ gameId: 'g-inv', token: 'tok2' })
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect, inviteGameId: 'g-inv' } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.click(screen.getByTestId('join-invite'))

    expect(joinGame).toHaveBeenCalledWith('g-inv')
    await vi.waitFor(() =>
      expect(onConnect).toHaveBeenCalledWith({ gameId: 'g-inv', token: 'tok2' }),
    )
  })

  test('Connect_inviteGameId_doesNotShowManualEntry', async () => {
    render(Connect, { props: { onConnect: vi.fn(), inviteGameId: 'g-inv' } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    expect(screen.getByTestId('join-invite')).toBeInTheDocument()
    expect(screen.queryByText('или войти вручную')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Подключиться' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Токен')).not.toBeInTheDocument()
  })
})

describe('Connect auth (#4)', () => {
  test('Connect_withoutSession_showsNickPasswordLoginRegister', async () => {
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    expect(screen.getByLabelText('Ник')).toBeInTheDocument()
    expect(screen.getByLabelText('Пароль')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Email')).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/имя/i)).not.toBeInTheDocument()
  })

  test('Connect_passwordField_isTypePassword', async () => {
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    expect(screen.getByLabelText('Пароль')).toHaveAttribute('type', 'password')
  })

  test('Connect_onMount_me200_showsNick', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })

    await waitFor(() => expect(screen.getByText('Ann')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Log in' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Register' })).not.toBeInTheDocument()
  })

  test('Connect_register_callsRegisterApi_setsSessionNick', async () => {
    vi.mocked(register).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Ann' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'password1' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    expect(register).toHaveBeenCalledWith('Ann', 'password1')
    await waitFor(() => expect(screen.getByText('Ann')).toBeInTheDocument())
    expect(screen.getByTestId('create-game')).toBeInTheDocument()
  })

  test('Connect_login_callsLoginApi_setsSessionNick', async () => {
    vi.mocked(login).mockResolvedValue({ nick: 'Bob' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Bob' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'password1' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    expect(login).toHaveBeenCalledWith('Bob', 'password1')
    await waitFor(() => expect(screen.getByText('Bob')).toBeInTheDocument())
  })

  test('Connect_register409_showsError', async () => {
    vi.mocked(register).mockRejectedValue(Object.assign(new Error('conflict'), { status: 409 }))
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Ann' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'password1' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/ник|занят|существует/i)
  })

  test('Connect_login401_showsError', async () => {
    vi.mocked(login).mockRejectedValue(Object.assign(new Error('unauthorized'), { status: 401 }))
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Ann' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'bad' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/неверн|парол|логин/i)
  })

  test('Connect_register400_showsError', async () => {
    vi.mocked(register).mockRejectedValue(Object.assign(new Error('bad request'), { status: 400 }))
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'A' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'short' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/некорр|коротк|проверь/i)
  })

  test('Connect_registerNetworkError_showsError', async () => {
    vi.mocked(register).mockRejectedValue(new Error('network'))
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Ann' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'password1' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/не удалось|сеть|ошибка/i)
  })

  test('Connect_register_doesNotWritePasswordToLocalStorage', async () => {
    vi.mocked(register).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Ann' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'password1' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Register' }))
    await waitFor(() => expect(screen.getByText('Ann')).toBeInTheDocument())

    expect(localStorage.getItem('nardy.credentials')).toBeNull()
    const all = JSON.stringify(localStorage)
    expect(all).not.toContain('password1')
  })

  test('Connect_registerSuccess_staysOnConnectWithCreateGame', async () => {
    vi.mocked(register).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Ann' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'password1' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    await waitFor(() => expect(screen.getByTestId('create-game')).toBeInTheDocument())
    expect(screen.getByRole('heading', { name: 'Длинные нарды' })).toBeInTheDocument()
  })
})

describe('Connect logout (card #13)', () => {
  test('Connect_loggedIn_showsLogoutNextToNick', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })

    await waitFor(() => expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann'))
    expect(screen.getByRole('button', { name: 'Выйти' })).toBeInTheDocument()
    expect(screen.getByTestId('logout')).toBeInTheDocument()
    expect(screen.getByTestId('create-game')).toBeInTheDocument()
  })

  test('Connect_afterCreate_logoutStillNextToNick', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann'))

    await fireEvent.click(screen.getByTestId('create-game'))
    await screen.findByTestId('invite-link')

    expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann')
    expect(screen.getByTestId('logout')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Выйти' })).toBeInTheDocument()
  })

  test('Connect_logoutSuccess_showsLoginFormAndClearsSeat', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    vi.mocked(logout).mockResolvedValue(undefined)
    const onLoggedOut = vi.fn()
    render(Connect, { props: { onConnect: vi.fn(), onLoggedOut } })
    await waitFor(() => expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann'))
    await fireEvent.click(screen.getByTestId('create-game'))
    await screen.findByTestId('invite-link')
    expect(loadCredentials()).toEqual({ gameId: 'g-new', token: 'tok' })

    await fireEvent.click(screen.getByTestId('logout'))

    expect(logout).toHaveBeenCalledOnce()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument()
    expect(screen.queryByTestId('session-nick')).toBeNull()
    expect(screen.queryByTestId('invite-link')).toBeNull()
    expect(loadCredentials()).toBeNull()
    expect(onLoggedOut).toHaveBeenCalledOnce()
  })

  test('Connect_logoutIdempotent_missingCookie_stillShowsLoginForm', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(logout).mockResolvedValue(undefined)
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann'))

    await fireEvent.click(screen.getByTestId('logout'))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(screen.queryByTestId('session-nick')).toBeNull()
  })

  test('Connect_logoutFailure_keepsNickAndSeatShowsAlert', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    vi.mocked(logout).mockRejectedValue(new Error('network'))
    const onLoggedOut = vi.fn()
    render(Connect, { props: { onConnect: vi.fn(), onLoggedOut } })
    await waitFor(() => expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann'))
    await fireEvent.click(screen.getByTestId('create-game'))
    await screen.findByTestId('invite-link')

    await fireEvent.click(screen.getByTestId('logout'))

    expect(await screen.findByRole('alert')).toHaveTextContent(/не удалось/i)
    expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann')
    expect(screen.getByTestId('invite-link')).toBeInTheDocument()
    expect(loadCredentials()).toEqual({ gameId: 'g-new', token: 'tok' })
    expect(onLoggedOut).not.toHaveBeenCalled()
  })

  test('Connect_logoutInFlight_ignoresRepeatClicks', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    let resolveLogout!: () => void
    const pending = new Promise<void>((resolve) => {
      resolveLogout = resolve
    })
    vi.mocked(logout).mockReturnValue(pending)
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann'))

    await fireEvent.click(screen.getByTestId('logout'))
    await fireEvent.click(screen.getByTestId('logout'))
    expect(logout).toHaveBeenCalledTimes(1)

    resolveLogout()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
  })

  test('Connect_logout_doesNotWritePasswordOrNickToStorage', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(logout).mockResolvedValue(undefined)
    saveCredentials({ gameId: 'g-1', token: 'tok-1' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann'))

    await fireEvent.click(screen.getByTestId('logout'))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())

    expect(loadCredentials()).toBeNull()
    const all = JSON.stringify(localStorage)
    expect(all).not.toContain('Ann')
    expect(all).not.toContain('password')
  })
})
