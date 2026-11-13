// FRONTEND_SPEC #23 / card #11 — Connect.svelte: create + invite only
// (manual gameId+token form removed). Card #4 — nick/password Log in /
// Register + me-on-mount. Card #19 — auth gate: create/join only when logged in;
// Вошли / Игра создана / invite layouts.

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

describe('Connect anonymous / auth gate (#19)', () => {
  test('Connect_anonymous_hidesCreateAndJoin', async () => {
    render(Connect, { props: { onConnect: vi.fn(), inviteGameId: 'g-inv' } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument()
    expect(screen.queryByTestId('create-game')).toBeNull()
    expect(screen.queryByTestId('join-invite')).toBeNull()
    expect(screen.queryByTestId('invite-link')).toBeNull()
    expect(screen.queryByText(/игра создана/i)).toBeNull()
  })

  test('Connect_default_loggedIn_showsCreateGameWithoutManualEntry', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByTestId('create-game')).toBeInTheDocument())

    expect(screen.queryByText('или войти вручную')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Подключиться' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Токен')).not.toBeInTheDocument()
  })
})

describe('Connect invite (#29 / #19)', () => {
  test('Connect_createGame_rendersInviteLinkWithGameId', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByTestId('create-game')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('create-game'))

    const link = (await screen.findByTestId('invite-link')) as HTMLInputElement
    expect(link.value).toContain('?game=g-new')
  })

  test('Connect_createGame_persistsCredentials', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByTestId('create-game')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('create-game'))
    await screen.findByTestId('invite-link')

    expect(loadCredentials()).toEqual({ gameId: 'g-new', token: 'tok' })
  })

  test('Connect_afterCreate_enterGame_callsOnConnect', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })
    await waitFor(() => expect(screen.getByTestId('create-game')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('create-game'))
    await fireEvent.click(await screen.findByTestId('enter-game'))

    expect(onConnect).toHaveBeenCalledWith({ gameId: 'g-new', token: 'tok' })
  })

  test('Connect_loggedInWithInvite_showsJoin', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn(), inviteGameId: 'g-inv' } })

    await waitFor(() => expect(screen.getByTestId('join-invite')).toBeInTheDocument())
    expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument()
    expect(screen.getByText('Тебя пригласили в игру.')).toBeInTheDocument()
    expect(screen.queryByTestId('create-game')).toBeNull()
    expect(screen.queryByTestId('logout')).toBeNull()
    expect(screen.queryByLabelText('Ник')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Log in' })).toBeNull()
  })

  test('Connect_inviteGameId_join_callsJoinGameAndOnConnect', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(joinGame).mockResolvedValue({ gameId: 'g-inv', token: 'tok2' })
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect, inviteGameId: 'g-inv' } })
    await waitFor(() => expect(screen.getByTestId('join-invite')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('join-invite'))

    expect(joinGame).toHaveBeenCalledWith('g-inv')
    await vi.waitFor(() =>
      expect(onConnect).toHaveBeenCalledWith({ gameId: 'g-inv', token: 'tok2' }),
    )
  })

  test('Connect_inviteGameId_doesNotShowManualEntry', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn(), inviteGameId: 'g-inv' } })
    await waitFor(() => expect(screen.getByTestId('join-invite')).toBeInTheDocument())

    expect(screen.queryByText('или войти вручную')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Подключиться' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Токен')).not.toBeInTheDocument()
  })

  test('Connect_loginWithInvite_showsJoinWithoutReload', async () => {
    vi.mocked(login).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn(), inviteGameId: 'g-inv' } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    expect(screen.queryByTestId('join-invite')).toBeNull()
    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Ann' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'password1' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    await waitFor(() => expect(screen.getByTestId('join-invite')).toBeInTheDocument())
    expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument()
    expect(screen.queryByTestId('create-game')).toBeNull()
    expect(screen.queryByTestId('logout')).toBeNull()
  })
})

describe('Connect auth (#4 / #19)', () => {
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

    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Log in' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Register' })).not.toBeInTheDocument()
  })

  test('Connect_loggedIn_showsVoshliCreateAndAccountLogout', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })

    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
    expect(screen.getByTestId('create-game')).toBeInTheDocument()
    expect(screen.getByTestId('logout')).toHaveTextContent('Выйти из аккаунта')
    // Logout is under create, not side-by-side nick-row.
    const create = screen.getByTestId('create-game')
    const logoutBtn = screen.getByTestId('logout')
    expect(
      create.compareDocumentPosition(logoutBtn) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  test('Connect_register_callsRegisterApi_setsSessionNick', async () => {
    vi.mocked(register).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fireEvent.input(screen.getByLabelText('Ник'), { target: { value: 'Ann' } })
    await fireEvent.input(screen.getByLabelText('Пароль'), { target: { value: 'password1' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Register' }))

    expect(register).toHaveBeenCalledWith('Ann', 'password1')
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
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
    await waitFor(() => expect(screen.getByText('Вы вошли как Bob')).toBeInTheDocument())
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
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())

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

describe('Connect logout (card #13 / #19)', () => {
  test('Connect_loggedIn_showsAccountLogoutUnderCreate', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    render(Connect, { props: { onConnect: vi.fn() } })

    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Выйти из аккаунта' })).toBeInTheDocument()
    expect(screen.getByTestId('logout')).toBeInTheDocument()
    expect(screen.getByTestId('create-game')).toBeInTheDocument()
  })

  test('Connect_afterCreate_hidesAccountLogout', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('create-game'))
    await screen.findByTestId('invite-link')

    expect(screen.getByText(/игра создана/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Копировать' })).toBeInTheDocument()
    expect(screen.getByTestId('enter-game')).toBeInTheDocument()
    // Игра создана: no session line (design 03-igra-sozdana), no account logout
    expect(screen.queryByTestId('session-nick')).toBeNull()
    expect(screen.queryByText('Вы вошли как Ann')).toBeNull()
    expect(screen.queryByTestId('logout')).toBeNull()
  })

  test('Connect_accountLogout_showsVhod', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(logout).mockResolvedValue(undefined)
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('logout'))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument()
    expect(screen.queryByTestId('session-nick')).toBeNull()
    expect(screen.queryByTestId('create-game')).toBeNull()
  })

  test('Connect_logoutSuccess_showsLoginFormAndClearsSeat', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok' })
    vi.mocked(logout).mockResolvedValue(undefined)
    const onLoggedOut = vi.fn()
    render(Connect, { props: { onConnect: vi.fn(), onLoggedOut } })
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
    // Logout only on Вошли (not after create); logout before create.
    expect(loadCredentials()).toBeNull()

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
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('logout'))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(screen.queryByTestId('session-nick')).toBeNull()
  })

  test('Connect_logoutFailure_keepsNickAndShowsAlert', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(logout).mockRejectedValue(new Error('network'))
    const onLoggedOut = vi.fn()
    render(Connect, { props: { onConnect: vi.fn(), onLoggedOut } })
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('logout'))

    expect(await screen.findByRole('alert')).toHaveTextContent(/не удалось/i)
    expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument()
    expect(screen.getByTestId('create-game')).toBeInTheDocument()
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
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())

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
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('logout'))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())

    expect(loadCredentials()).toBeNull()
    const all = JSON.stringify(localStorage)
    expect(all).not.toContain('Ann')
    expect(all).not.toContain('password')
  })
})
