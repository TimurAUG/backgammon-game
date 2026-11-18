// FRONTEND_SPEC #24b — App.svelte: маршрутизация Connect ↔ Game и
// проводка WSClient (onMessage→gameStore, onAction→send).
// Card #19 — auth gate before seat auto-connect; GameOver seat exit (no create-and-enter).

import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { flushSync } from 'svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { createGame, logout, me, type MeResponse } from './lib/api'
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

function meOk(nick: string): MeResponse {
  return { nick, played: 0, wins: 0, losses: 0, win_percent: 0 }
}

/** Card #19: board only after me succeeds + saved/URL seat creds. */
function enterWithSavedCreds(gameId = 'g-1', token = 'tok', nick = 'Ann'): void {
  vi.mocked(me).mockResolvedValue(meOk(nick))
  saveCredentials({ gameId, token })
  render(App, { props: { createClient: testCreateClient } })
}

async function waitForBoard(): Promise<void> {
  await waitFor(() => expect(screen.getByTestId('point-1')).toBeInTheDocument())
}

describe('App routing (#24b / #19)', () => {
  test('App_noSavedCreds_showsConnectNotGame', async () => {
    render(App, { props: { createClient: testCreateClient } })

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(screen.queryByTestId('create-game')).toBeNull()
    expect(screen.queryByTestId('point-1')).toBeNull()
  })

  test('App_afterConnect_showsGameNotConnect', async () => {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()

    expect(screen.queryByTestId('create-game')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Log in' })).toBeNull()
  })

  // Card #23 AC5 — stats block lives only on Вошли, not on the board.
  test('App_boardShown_accountStatsAbsent', async () => {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()

    expect(screen.queryByTestId('account-stats')).toBeNull()
    expect(screen.queryByText('Статистика')).toBeNull()
  })
})

describe('App auth gate (#19)', () => {
  test('App_savedSeatWithoutMe_showsVhodNotBoard', async () => {
    saveCredentials({ gameId: 'g-saved', token: 'tok-saved' })
    render(App, { props: { createClient: testCreateClient } })

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(screen.queryByTestId('point-1')).toBeNull()
    expect(MockWebSocket.instances).toHaveLength(0)
  })

  test('App_urlTokenWithoutMe_showsVhodNotBoard', async () => {
    window.history.replaceState(null, '', '/?game=g-url&token=tok-url')
    render(App, { props: { createClient: testCreateClient } })

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(screen.queryByTestId('point-1')).toBeNull()
    expect(MockWebSocket.instances).toHaveLength(0)
  })

  test('App_mePending_doesNotStartSessionFromSaved', () => {
    let resolveMe!: (value: MeResponse) => void
    vi.mocked(me).mockReturnValue(
      new Promise<MeResponse>((resolve) => {
        resolveMe = resolve
      }),
    )
    saveCredentials({ gameId: 'g-saved', token: 'tok-saved' })
    render(App, { props: { createClient: testCreateClient } })

    expect(MockWebSocket.instances).toHaveLength(0)
    expect(screen.queryByTestId('point-1')).toBeNull()
    // Title-only Connect while sessionChecked is false.
    expect(screen.getByRole('heading', { name: 'Длинные нарды' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Log in' })).toBeNull()

    resolveMe(meOk('Ann'))
  })
})

describe('App WSClient wiring (#24b)', () => {
  test('App_afterConnect_opensSocket', async () => {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()

    expect(MockWebSocket.instances).toHaveLength(1)
  })

  test('App_incomingState_updatesBoard', async () => {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()

    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive(stateFixture())

    expect(await screen.findAllByTestId(/^checker-24-\d+$/)).toHaveLength(15)
  })

  test('App_gameAction_sentOverSocket', async () => {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()

    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    await fireEvent.click(await screen.findByTestId('action-resign'))

    expect(ws.sent).toContain(JSON.stringify({ type: 'RESIGN' }))
  })
})

describe('App game over exit (#19)', () => {
  async function reachGameOverWithQuery(gameId = 'g-old', token = 'tok-old'): Promise<MockWebSocket> {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    saveCredentials({ gameId, token })
    window.history.replaceState(null, '', `/?game=${gameId}`)
    render(App, { props: { createClient: testCreateClient } })
    await waitForBoard()
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture({ nicks: { white: 'Ann', black: 'Bob' } }))
    ws.receive({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })
    await screen.findByTestId('action-exit')
    return ws
  }

  test('App_gameOverExit_endsSeatReturnsVoshliSameNick_noCreateNoLogout', async () => {
    // After seat exit: Connect remounts; me still 200 → Вошли (create + account logout).
    // Keep me resolving to Ann for the remounted Connect.
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    await reachGameOverWithQuery('g-old', 'tok-old')

    await fireEvent.click(screen.getByTestId('action-exit'))

    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
    expect(screen.getByTestId('create-game')).toBeInTheDocument()
    expect(screen.getByTestId('logout')).toBeInTheDocument()
    expect(screen.queryByTestId('point-1')).toBeNull()
    expect(createGame).not.toHaveBeenCalled()
    expect(logout).not.toHaveBeenCalled()
    expect(loadCredentials()).toBeNull()
  })

  test('App_gameOverExit_clearsGameQuery', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    await reachGameOverWithQuery('g-old', 'tok-old')

    await fireEvent.click(screen.getByTestId('action-exit'))

    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
    expect(location.search).not.toContain('game=')
    expect(location.search).not.toContain('token')
  })
})

describe('App logout (card #13 / #19)', () => {
  test('App_connectLogoutSuccess_clearsAndShowsVhod', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    window.history.replaceState(null, '', '/')
    render(App, { props: { createClient: testCreateClient } })
    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())

    await fireEvent.click(screen.getByTestId('logout'))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(logout).toHaveBeenCalledOnce()
    expect(screen.queryByTestId('session-nick')).toBeNull()
    expect(loadCredentials()).toBeNull()
  })
})

describe('App invite link (#30 / #19)', () => {
  test('App_gameParamInUrl_loggedOut_showsVhodNotJoinInvite', async () => {
    window.history.replaceState(null, '', '/?game=g-xyz')

    render(App, { props: { createClient: testCreateClient } })

    await waitFor(() => expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument())
    expect(screen.queryByTestId('join-invite')).toBeNull()
    expect(screen.queryByTestId('create-game')).toBeNull()
  })

  test('App_gameParamInUrl_loggedIn_showsJoinInvite', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    window.history.replaceState(null, '', '/?game=g-xyz')

    render(App, { props: { createClient: testCreateClient } })

    await waitFor(() => expect(screen.getByTestId('join-invite')).toBeInTheDocument())
    expect(screen.queryByTestId('create-game')).toBeNull()
  })

  test('App_noGameParam_loggedIn_showsCreateInConnect', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    render(App, { props: { createClient: testCreateClient } })

    await waitFor(() => expect(screen.getByTestId('create-game')).toBeInTheDocument())
    expect(screen.queryByTestId('join-invite')).toBeNull()
  })
})

describe('App auto-connect (#24c / #19)', () => {
  test('App_savedCreds_autoConnectsToGameSkippingConnect', async () => {
    enterWithSavedCreds('g-saved', 'tok-saved')
    await waitForBoard()

    expect(screen.queryByTestId('create-game')).toBeNull()
    expect(MockWebSocket.instances).toHaveLength(1)
  })

  test('App_savedCreds_joinsWithSavedCreds', async () => {
    enterWithSavedCreds('g-saved', 'tok-saved')
    await waitForBoard()

    MockWebSocket.last().acceptOpen()

    expect(MockWebSocket.last().sent).toContain(
      JSON.stringify({ type: 'JOIN', gameId: 'g-saved', token: 'tok-saved' }),
    )
  })
})

describe('App reconnect link (#30 / #19)', () => {
  test('App_gameAndTokenInUrl_autoReconnects', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    window.history.replaceState(null, '', '/?game=g-url&token=tok-url')

    render(App, { props: { createClient: testCreateClient } })
    await waitForBoard()

    MockWebSocket.last().acceptOpen()
    expect(MockWebSocket.last().sent).toContain(
      JSON.stringify({ type: 'JOIN', gameId: 'g-url', token: 'tok-url' }),
    )
  })

  test('App_gameAndTokenInUrl_savesCredsAndStripsToken', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    window.history.replaceState(null, '', '/?game=g-url&token=tok-url')

    render(App, { props: { createClient: testCreateClient } })
    await waitForBoard()

    expect(loadCredentials()).toEqual({ gameId: 'g-url', token: 'tok-url' })
    expect(location.search).not.toContain('token')
  })

  test('App_urlToken_overridesSavedCreds', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    saveCredentials({ gameId: 'g-old', token: 'tok-old' })
    window.history.replaceState(null, '', '/?game=g-url&token=tok-url')

    render(App, { props: { createClient: testCreateClient } })
    await waitForBoard()

    MockWebSocket.last().acceptOpen()
    expect(MockWebSocket.last().sent).toContain(
      JSON.stringify({ type: 'JOIN', gameId: 'g-url', token: 'tok-url' }),
    )
  })
})

describe('App public invite vs saved creds (fix / #19)', () => {
  test('App_publicInvite_staleCredsForOtherGame_showsJoinInviteAndDoesNotConnect', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    saveCredentials({ gameId: 'g-other', token: 'tok-other' })
    window.history.replaceState(null, '', '/?game=g-new')

    render(App, { props: { createClient: testCreateClient } })

    await waitFor(() => expect(screen.getByTestId('join-invite')).toBeInTheDocument())
    expect(MockWebSocket.instances).toHaveLength(0)
  })

  test('App_publicInvite_ownSavedGame_autoReconnects', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    saveCredentials({ gameId: 'g-1', token: 'tok-1' })
    window.history.replaceState(null, '', '/?game=g-1')

    render(App, { props: { createClient: testCreateClient } })
    await waitForBoard()

    expect(MockWebSocket.instances).toHaveLength(1)
  })
})

describe('App UNAUTHORIZED (#25)', () => {
  async function connectAndOpen(): Promise<MockWebSocket> {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()
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
    // Seat token invalid — endSession; account cookie still valid → Вошли.
    const ws = await connectAndOpen()

    ws.receive({ type: 'ERROR', code: 'UNAUTHORIZED', message: 'нет доступа' })

    expect(await screen.findByText('Вы вошли как Ann')).toBeInTheDocument()
    expect(screen.getByTestId('create-game')).toBeInTheDocument()
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
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()
    const ws = MockWebSocket.last()
    ws.acceptOpen()

    ws.receive({ type: 'OPPONENT_JOINED' })

    expect(await screen.findByText('Соперник присоединился')).toBeInTheDocument()
  })

  test('App_gameOverExit_clearsNotifications', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    ws.receive({ type: 'OPPONENT_JOINED' })
    await screen.findByText('Соперник присоединился')

    ws.receive({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })
    await fireEvent.click(await screen.findByTestId('action-exit'))

    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
    expect(screen.queryByText('Соперник присоединился')).toBeNull()
  })
})

describe('App turn-skipped notification', () => {
  test('App_myTurnSkipped_showsDiceAndPassNotice', async () => {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())

    ws.receive({
      type: 'TURN_SKIPPED',
      color: 'white',
      dice: { a: 1, b: 2, isDouble: false, remaining: [1, 2] },
    })

    expect(
      await screen.findByText((t) => t.includes('1') && t.includes('2') && t.includes('нечем')),
    ).toBeInTheDocument()
  })

  test('App_opponentTurnSkipped_showsItIsMyTurn', async () => {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()
    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())

    ws.receive({
      type: 'TURN_SKIPPED',
      color: 'black',
      dice: { a: 3, b: 5, isDouble: false, remaining: [3, 5] },
    })

    expect(await screen.findByText((t) => t.includes('ваш ход'))).toBeInTheDocument()
  })
})

describe('App chat (#40)', () => {
  async function connectedSocket(myColor: 'white' | 'black' = 'white'): Promise<MockWebSocket> {
    enterWithSavedCreds('g-1', 'tok')
    await waitForBoard()
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

  test('App_gameOverExit_clearsChat', async () => {
    vi.mocked(me).mockResolvedValue(meOk('Ann'))
    const ws = await connectedSocket('white')
    ws.receive({ type: 'CHAT', sender: 'black', text: 'до встречи' })
    ws.receive({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })

    await fireEvent.click(await screen.findByTestId('action-exit'))

    await waitFor(() => expect(screen.getByText('Вы вошли как Ann')).toBeInTheDocument())
    expect(chat.messages).toEqual([])
  })
})

describe('App connection state (#26e)', () => {
  test('App_socketReconnecting_disablesActionBar', async () => {
    vi.useFakeTimers()
    try {
      enterWithSavedCreds('g-1', 'tok')
      // Fake timers: waitForBoard via microtasks — flush me promise.
      await vi.waitFor(() => expect(screen.getByTestId('point-1')).toBeInTheDocument())
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
