// FRONTEND_SPEC #23 — Connect.svelte: форма gameId+token, сохранение
// в localStorage, колбек onConnect с введёнными кредами.
// Card #4 — nick/password Log in / Register + me-on-mount.

import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { createGame, joinGame, login, me, register } from '../lib/api'
import { loadCredentials } from '../lib/credentials'

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

async function fillAndSubmit(gameId: string, token: string): Promise<void> {
  await fireEvent.input(screen.getByLabelText('ID игры'), { target: { value: gameId } })
  await fireEvent.input(screen.getByLabelText('Токен'), { target: { value: token } })
  await fireEvent.click(screen.getByRole('button', { name: 'Подключиться' }))
}

describe('Connect form (#23)', () => {
  test('Connect_renders_formFields', async () => {
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())
    expect(screen.getByLabelText('ID игры')).toBeInTheDocument()
    expect(screen.getByLabelText('Токен')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Подключиться' })).toBeInTheDocument()
  })

  test('Connect_submitWithFilledFields_callsOnConnect', async () => {
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fillAndSubmit('g-42', 'tok-abc')

    expect(onConnect).toHaveBeenCalledExactlyOnceWith({ gameId: 'g-42', token: 'tok-abc' })
  })

  test('Connect_submitWithFilledFields_persistsToLocalStorage', async () => {
    render(Connect, { props: { onConnect: vi.fn() } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fillAndSubmit('g-42', 'tok-abc')

    expect(loadCredentials()).toEqual({ gameId: 'g-42', token: 'tok-abc' })
  })

  test('Connect_submitWithEmptyGameId_doesNotCallOnConnect', async () => {
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })
    await waitFor(() => expect(me).toHaveBeenCalled())

    await fillAndSubmit('   ', 'tok-abc')

    expect(onConnect).not.toHaveBeenCalled()
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
