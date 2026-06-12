// FRONTEND_SPEC #24b — App.svelte: маршрутизация Connect ↔ Game и
// проводка WSClient (onMessage→gameStore, onAction→send, onNewGame→сброс).
// createClient инжектируется: реальный WSClient поверх MockWebSocket.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { flushSync } from 'svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { loadCredentials, saveCredentials, type Credentials } from './lib/credentials'
import { resetConnectionState } from './stores/connection.svelte'
import { resetGameState } from './stores/game.svelte'
import { resetNotifications } from './stores/notifications.svelte'
import { chat, resetChat } from './stores/chat.svelte'
import { WSClient, type WSConnectionCtor } from './transport/ws'
import { MockWebSocket } from '../tests/mockWebSocket'
import { stateFixture } from '../tests/fixtures'

import App from './App.svelte'

beforeEach(() => {
  localStorage.clear()
  resetGameState()
  resetConnectionState()
  resetNotifications()
  resetChat()
  MockWebSocket.reset()
  window.history.replaceState(null, '', '/')
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

describe('App new game (#24b)', () => {
  test('App_newGame_clearsCredsAndReturnsToConnect', async () => {
    render(App, { props: { createClient: testCreateClient } })
    await connectVia('g-1', 'tok')

    const ws = MockWebSocket.last()
    ws.acceptOpen()
    ws.receive({ type: 'JOINED', color: 'white' })
    ws.receive(stateFixture())
    ws.receive({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })
    await fireEvent.click(await screen.findByTestId('action-new-game'))

    expect(screen.getByLabelText('ID игры')).toBeInTheDocument()
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
