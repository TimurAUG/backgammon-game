// FRONTEND_SPEC #24b — App.svelte: маршрутизация Connect ↔ Game и
// проводка WSClient (onMessage→gameStore, onAction→send, onNewGame→сброс).
// createClient инжектируется: реальный WSClient поверх MockWebSocket.
// Card #6 — GameOver «Новая игра» create-and-enter via createGame().

import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { flushSync } from 'svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { createGame, logout, me } from './lib/api'
import { loadCredentials, saveCredentials, type Credentials } from './lib/credentials'
import { resetConnectionState } from './stores/connection.svelte'
import { resetGameState } from './stores/game.svelte'
import { resetNotifications } from './stores/notifications.svelte'
import { chat, resetChat } from './stores/chat.svelte'
import { WSClient, type WSConnectionCtor } from './transport/ws'
import { MockWebSocket } from '../tests/mockWebSocket'
import { stateFixture } from '../tests/fixtures'

import App from './App.svelte'

vi.mock('./lib/api', () => ({
  createGame: vi.fn(),
  joinGame: vi.fn(),
  register: vi.fn(),
  login: vi.fn(),
  me: vi.fn(),
  logout: vi.fn(),
}))

beforeEach(() => {
  localStorage.clear()
  resetGameState()
  resetConnectionState()
  resetNotifications()
  resetChat()
  MockWebSocket.reset()
  window.history.replaceState(null, '', '/')
  vi.mocked(createGame).mockReset()
  vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok-new' })
  vi.mocked(me).mockReset()
  vi.mocked(me).mockRejectedValue(Object.assign(new Error('unauthorized'), { status: 401 }))
  vi.mocked(logout).mockReset()
  vi.mocked(logout).mockResolvedValue(undefined)
})

function testCreateClient(creds: Credentials): WSClient {
  return new WSClient(
    { url: 'ws://test/ws', gameId: creds.gameId, token: creds.token },
    MockWebSocket as unknown as WSConnectionCtor,
  )
}

async function connectVia(gameId: string, token: string): Promise<void> {
  await fireEvent.input(screen.getByLabelText('ID игры'), { target: { value: gameId } })
  await fireEvent.input(screen.getByLabelText('Токен'), { target: { value: token } })
  await fireEvent.click(screen.getByRole('button', { name: 'Подключиться' }))
}

describe('App routing (#24b)', () => {
  test('App_noSavedCreds_showsConnectNotGame', () => {
    render(App, { props: { createClient: testCreateClient } })

    expect(screen.getByLabelText('ID игры')).toBeInTheDocument()
    expect(screen.queryByTestId('point-1')).toBeNull()
  })

  test('App_afterConnect_showsGameNotConnect', async () => {
    render(App, { props: { createClient: testCreateClient } })

    await connectVia('g-1', 'tok')

    expect(screen.getByTestId('point-1')).toBeInTheDocument()
    expect(screen.queryByLabelText('ID игры')).toBeNull()
  })
})

describe('App WSClient wiring (#24b)', () => {
  test('App_afterConnect_opensSocket', async () => {
    render(App, { props: { createClient: testCreateClient } })

    await connectVia('g-1', 'tok')

    expect(MockWebSocket.instances).toHaveLength(1)
  })

  test('App_incomingState_updatesBoard', async () => {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')

    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive(stateFixture())

    expect(await screen.findAllByTestId(/^checker-24-\d+$/)).toHaveLength(15)
  })

  test('App_gameAction_sentOverSocket', async () => {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')

    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    await fireEvent.click(await screen.findByTestId('action-resign'))

    expect(ws.sent).toContain(JSON.stringify({ type: 'RESIGN' }))
  })
})

describe('App new game after game over (#22 / card #6)', () => {
  // With ?game= in the URL Connect hides the manual form (join-invite).
  // Seed matching saved creds so App auto-connects into the finished session.
  async function reachGameOverWithQuery(gameId = 'g-old', token = 'tok-old'): Promise<MockWebSocket> {
    saveCredentials({ gameId, token })
    window.history.replaceState(null, '', `/?game=${gameId}`)
    render(App, { props: { createClient: testCreateClient } })
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    ws.receive({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })
    await screen.findByTestId('action-new-game')
    return ws
  }

  test('App_newGame_afterGameOver_createsAndEntersFreshGame', async () => {
    await reachGameOverWithQuery('g-old', 'tok-old')

    await fireEvent.click(screen.getByTestId('action-new-game'))

    await waitFor(() => {
      expect(screen.queryByTestId('game-over')).toBeNull()
      expect(screen.getByTestId('point-1')).toBeInTheDocument()
    })
    expect(createGame).toHaveBeenCalledOnce()
    expect(loadCredentials()).toEqual({ gameId: 'g-new', token: 'tok-new' })
    expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(2)
    const latest = MockWebSocket.last()
    latest.acceptOpen()
    expect(latest.sent).toContain(
      JSON.stringify({ type: 'JOIN', gameId: 'g-new', token: 'tok-new' }),
    )
    expect(location.search).toContain('game=g-new')
    expect(location.search).not.toContain('g-old')
    expect(location.search).not.toContain('token')
  })

  test('App_newGame_createFails_keepsGameOverAndShowsError', async () => {
    vi.mocked(createGame).mockRejectedValue(Object.assign(new Error('fail'), { status: 500 }))
    await reachGameOverWithQuery()

    await fireEvent.click(screen.getByTestId('action-new-game'))

    expect(await screen.findByRole('alert')).toHaveTextContent('Не удалось создать игру')
    expect(screen.getByTestId('game-over')).toBeInTheDocument()
    expect(screen.queryByLabelText('ID игры')).toBeNull()
    expect(loadCredentials()).toEqual({ gameId: 'g-old', token: 'tok-old' })
    expect(location.search).toContain('game=g-old')
  })

  test('App_newGame_createUnauthorized_returnsToConnectAndClearsGameQuery', async () => {
    vi.mocked(createGame).mockRejectedValue(Object.assign(new Error('unauthorized'), { status: 401 }))
    await reachGameOverWithQuery()

    await fireEvent.click(screen.getByTestId('action-new-game'))

    expect(await screen.findByLabelText('ID игры')).toBeInTheDocument()
    expect(screen.queryByTestId('point-1')).toBeNull()
    expect(loadCredentials()).toBeNull()
    expect(location.search).not.toContain('game=')
    expect(screen.queryByTestId('join-invite')).toBeNull()
  })

  // Edge: double-press while create is in flight — App must set creatingNewGame
  // so the button disables and a second click cannot call createGame again.
  test('App_newGame_doublePressWhileInFlight_callsCreateOnceAndDisablesButton', async () => {
    let resolveCreate!: (value: Credentials) => void
    const pending = new Promise<Credentials>((resolve) => {
      resolveCreate = resolve
    })
    vi.mocked(createGame).mockReturnValue(pending)

    await reachGameOverWithQuery()

    await fireEvent.click(screen.getByTestId('action-new-game'))

    await waitFor(() => {
      expect(screen.getByTestId('action-new-game')).toBeDisabled()
    })
    expect(createGame).toHaveBeenCalledTimes(1)

    await fireEvent.click(screen.getByTestId('action-new-game'))
    expect(createGame).toHaveBeenCalledTimes(1)

    resolveCreate({ gameId: 'g-new', token: 'tok-new' })

    await waitFor(() => {
      expect(screen.queryByTestId('game-over')).toBeNull()
      expect(screen.getByTestId('point-1')).toBeInTheDocument()
    })
    expect(createGame).toHaveBeenCalledTimes(1)
  })

  // Edge: mid-game «Сменить игру» must endSession → Connect, not create-and-enter.
  test('App_switchGame_midGame_returnsToConnectWithoutCreate', async () => {
    saveCredentials({ gameId: 'g-1', token: 'tok-1' })
    render(App, { props: { createClient: testCreateClient } })

    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    await screen.findByTestId('switch-game')

    await fireEvent.click(screen.getByTestId('switch-game'))

    expect(await screen.findByTestId('create-game')).toBeInTheDocument()
    expect(screen.queryByTestId('point-1')).toBeNull()
    expect(createGame).not.toHaveBeenCalled()
    expect(logout).not.toHaveBeenCalled()
    expect(loadCredentials()).toBeNull()
  })
})

describe('App logout (card #13)', () => {
  test('App_gameLogoutSuccess_returnsToLoginFormClosesSocket', async () => {
    // beforeEach: me → 401. Auto-connect via seat creds; after logout Connect
    // remounts and me stays 401 → nick/password Log in / Register.
    saveCredentials({ gameId: 'g-1', token: 'tok-1' })
    window.history.replaceState(null, '', '/?game=g-1')
    render(App, { props: { createClient: testCreateClient } })

    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    await screen.findByTestId('logout')

    await fireEvent.click(screen.getByTestId('logout'))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument()
    })
    expect(logout).toHaveBeenCalledOnce()
    expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument()
    expect(screen.queryByTestId('point-1')).toBeNull()
    expect(screen.queryByTestId('invite-link')).toBeNull()
    expect(screen.queryByTestId('join-invite')).toBeNull()
    expect(screen.queryByTestId('session-nick')).toBeNull()
    expect(loadCredentials()).toBeNull()
    expect(ws.readyState).toBe(MockWebSocket.CLOSED)
    expect(location.search).not.toContain('game=')
  })

  test('App_gameLogoutFailure_staysOnBoardKeepsSeat', async () => {
    vi.mocked(logout).mockRejectedValue(new Error('network'))
    saveCredentials({ gameId: 'g-1', token: 'tok-1' })
    render(App, { props: { createClient: testCreateClient } })

    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    await screen.findByTestId('logout')

    await fireEvent.click(screen.getByTestId('logout'))

    expect(await screen.findByRole('alert')).toHaveTextContent(/не удалось/i)
    expect(screen.getByTestId('point-1')).toBeInTheDocument()
    expect(loadCredentials()).toEqual({ gameId: 'g-1', token: 'tok-1' })
    expect(ws.readyState).not.toBe(MockWebSocket.CLOSED)
  })

  test('App_connectLogoutSuccess_clearsInviteQueryShowsLogin', async () => {
    vi.mocked(me).mockResolvedValue({ nick: 'Ann' })
    vi.mocked(createGame).mockResolvedValue({ gameId: 'g-new', token: 'tok-new' })
    window.history.replaceState(null, '', '/')
    render(App, { props: { createClient: testCreateClient } })
    await waitFor(() => expect(screen.getByTestId('session-nick')).toHaveTextContent('Ann'))
    await fireEvent.click(screen.getByTestId('create-game'))
    await screen.findByTestId('invite-link')

    await fireEvent.click(screen.getByTestId('logout'))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(logout).toHaveBeenCalledOnce()
    expect(screen.queryByTestId('invite-link')).toBeNull()
    expect(loadCredentials()).toBeNull()
  })
})

describe('App invite link (#30)', () => {
  test('App_gameParamInUrl_showsJoinInviteInConnect', () => {
    window.history.replaceState(null, '', '/?game=g-xyz')

    render(App, { props: { createClient: testCreateClient } })

    expect(screen.getByTestId('join-invite')).toBeInTheDocument()
    expect(screen.queryByTestId('create-game')).toBeNull()
  })

  test('App_noGameParam_showsCreateInConnect', () => {
    render(App, { props: { createClient: testCreateClient } })

    expect(screen.getByTestId('create-game')).toBeInTheDocument()
    expect(screen.queryByTestId('join-invite')).toBeNull()
  })
})

describe('App auto-connect (#24c)', () => {
  test('App_savedCreds_autoConnectsToGameSkippingConnect', () => {
    saveCredentials({ gameId: 'g-saved', token: 'tok-saved' })

    render(App, { props: { createClient: testCreateClient } })

    expect(screen.getByTestId('point-1')).toBeInTheDocument()
    expect(screen.queryByLabelText('ID игры')).toBeNull()
    expect(MockWebSocket.instances).toHaveLength(1)
  })

  test('App_savedCreds_joinsWithSavedCreds', () => {
    saveCredentials({ gameId: 'g-saved', token: 'tok-saved' })
    render(App, { props: { createClient: testCreateClient } })

    MockWebSocket.last().acceptOpen()

    expect(MockWebSocket.last().sent).toContain(
      JSON.stringify({ type: 'JOIN', gameId: 'g-saved', token: 'tok-saved' }),
    )
  })
})

describe('App reconnect link (#30)', () => {
  test('App_gameAndTokenInUrl_autoReconnects', () => {
    window.history.replaceState(null, '', '/?game=g-url&token=tok-url')

    render(App, { props: { createClient: testCreateClient } })

    expect(screen.getByTestId('point-1')).toBeInTheDocument()
    MockWebSocket.last().acceptOpen()
    expect(MockWebSocket.last().sent).toContain(
      JSON.stringify({ type: 'JOIN', gameId: 'g-url', token: 'tok-url' }),
    )
  })

  test('App_gameAndTokenInUrl_savesCredsAndStripsToken', () => {
    window.history.replaceState(null, '', '/?game=g-url&token=tok-url')

    render(App, { props: { createClient: testCreateClient } })

    expect(loadCredentials()).toEqual({ gameId: 'g-url', token: 'tok-url' })
    expect(location.search).not.toContain('token')
  })

  test('App_urlToken_overridesSavedCreds', () => {
    saveCredentials({ gameId: 'g-old', token: 'tok-old' })
    window.history.replaceState(null, '', '/?game=g-url&token=tok-url')

    render(App, { props: { createClient: testCreateClient } })

    MockWebSocket.last().acceptOpen()
    expect(MockWebSocket.last().sent).toContain(
      JSON.stringify({ type: 'JOIN', gameId: 'g-url', token: 'tok-url' }),
    )
  })
})

describe('App public invite vs saved creds (fix)', () => {
  // Публичное приглашение ?game=<id> (без token): нельзя авто-подключаться по
  // сохранённым кредам ДРУГОЙ игры — иначе игрок уходит не в ту игру (или в
  // одном браузере занимает чужой слот, став дублем создателя). Должны показать
  // join-invite, чтобы получить собственный токен.
  test('App_publicInvite_staleCredsForOtherGame_showsJoinInviteAndDoesNotConnect', () => {
    saveCredentials({ gameId: 'g-other', token: 'tok-other' })
    window.history.replaceState(null, '', '/?game=g-new')

    render(App, { props: { createClient: testCreateClient } })

    expect(screen.getByTestId('join-invite')).toBeInTheDocument()
    expect(MockWebSocket.instances).toHaveLength(0)
  })

  // Приглашение в СВОЮ сохранённую игру (saved.gameId == invite) — это возврат
  // (F5/повторный заход по ссылке): авто-реконнект по saved сохраняется.
  test('App_publicInvite_ownSavedGame_autoReconnects', () => {
    saveCredentials({ gameId: 'g-1', token: 'tok-1' })
    window.history.replaceState(null, '', '/?game=g-1')

    render(App, { props: { createClient: testCreateClient } })

    expect(screen.getByTestId('point-1')).toBeInTheDocument()
    expect(MockWebSocket.instances).toHaveLength(1)
  })
})

describe('App UNAUTHORIZED (#25)', () => {
  async function connectAndOpen(): Promise<MockWebSocket> {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    return ws
  }

  test('App_unauthorized_clearsCredentials', async () => {
    const ws = await connectAndOpen()

    ws.receive({ type: 'ERROR', code: 'UNAUTHORIZED', message: 'нет доступа' })

    expect(loadCredentials()).toBeNull()
  })

  test('App_unauthorized_returnsToConnect', async () => {
    const ws = await connectAndOpen()

    ws.receive({ type: 'ERROR', code: 'UNAUTHORIZED', message: 'нет доступа' })

    expect(await screen.findByLabelText('ID игры')).toBeInTheDocument()
    expect(screen.queryByTestId('point-1')).toBeNull()
  })

  test('App_unauthorized_closesSocketToStopReconnect', async () => {
    const ws = await connectAndOpen()

    ws.receive({ type: 'ERROR', code: 'UNAUTHORIZED', message: 'нет доступа' })

    expect(ws.readyState).toBe(MockWebSocket.CLOSED)
  })
})

describe('App opponent-joined notification (#34a)', () => {
  test('App_opponentJoined_showsToast', async () => {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')
    const ws = MockWebSocket.last()
    ws.acceptOpen()

    ws.receive({ type: 'OPPONENT_JOINED' })

    expect(await screen.findByText('Соперник присоединился')).toBeInTheDocument()
  })

  test('App_newGame_clearsNotifications', async () => {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    ws.receive({ type: 'OPPONENT_JOINED' })
    await screen.findByText('Соперник присоединился')

    ws.receive({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })
    await fireEvent.click(await screen.findByTestId('action-new-game'))

    expect(screen.queryByText('Соперник присоединился')).toBeNull()
  })
})

describe('App turn-skipped notification', () => {
  test('App_myTurnSkipped_showsDiceAndPassNotice', async () => {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())

    ws.receive({
      type: 'TURN_SKIPPED',
      color: 'white',
      dice: { a: 1, b: 2, isDouble: false, remaining: [1, 2] },
    })

    // мой ход пропущен: показаны кубики 1 и 2 + что ходить нечем
    expect(
      await screen.findByText((t) => t.includes('1') && t.includes('2') && t.includes('нечем')),
    ).toBeInTheDocument()
  })

  test('App_opponentTurnSkipped_showsItIsMyTurn', async () => {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())

    ws.receive({
      type: 'TURN_SKIPPED',
      color: 'black',
      dice: { a: 3, b: 5, isDouble: false, remaining: [3, 5] },
    })

    // ход соперника пропущен → сообщаем, что теперь ваш ход
    expect(await screen.findByText((t) => t.includes('ваш ход'))).toBeInTheDocument()
  })
})

describe('App chat (#40)', () => {
  async function connectedSocket(myColor: 'white' | 'black' = 'white'): Promise<MockWebSocket> {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: myColor })
    ws.receive(stateFixture())
    return ws
  }

  test('App_incomingChatFromOpponent_appendsAndToasts', async () => {
    const ws = await connectedSocket('white')

    ws.receive({ type: 'CHAT', sender: 'black', text: 'привет соперник' })

    expect(chat.messages).toEqual([{ sender: 'black', text: 'привет соперник' }])
    expect(await screen.findByText('привет соперник')).toBeInTheDocument()
  })

  test('App_ownChatEcho_appendsButNoToast', async () => {
    const ws = await connectedSocket('white')

    ws.receive({ type: 'CHAT', sender: 'white', text: 'моё эхо' })

    expect(chat.messages).toEqual([{ sender: 'white', text: 'моё эхо' }])
    // Своё сообщение (эхо) не тостим; панель свёрнута → текста в DOM нет.
    expect(screen.queryByText('моё эхо')).toBeNull()
  })

  test('App_incomingChatHistory_populatesStore', async () => {
    const ws = await connectedSocket('white')

    ws.receive({
      type: 'CHAT_HISTORY',
      chat: [
        { sender: 'white', text: 'раз' },
        { sender: 'black', text: 'два' },
      ],
    })

    expect(chat.messages).toEqual([
      { sender: 'white', text: 'раз' },
      { sender: 'black', text: 'два' },
    ])
  })

  test('App_newGame_clearsChat', async () => {
    const ws = await connectedSocket('white')
    ws.receive({ type: 'CHAT', sender: 'black', text: 'до встречи' })
    ws.receive({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })

    await fireEvent.click(await screen.findByTestId('action-new-game'))

    expect(chat.messages).toEqual([])
  })
})

describe('App connection state (#26e)', () => {
  test('App_socketReconnecting_disablesActionBar', async () => {
    vi.useFakeTimers()
    try {
      render(App, { props: { createClient: testCreateClient } })
      await connectVia('g-1', 'tok')
      const ws = MockWebSocket.last()
      ws.acceptOpen()
      ws.receive({ type: 'JOINED', color: 'white' })
      ws.receive(stateFixture())

      ws.serverClose(1006)
      flushSync()

      expect(screen.getByTestId('action-resign')).toBeDisabled()
    } finally {
      vi.useRealTimers()
    }
  })
})
