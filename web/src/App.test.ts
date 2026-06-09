// FRONTEND_SPEC #24b — App.svelte: маршрутизация Connect ↔ Game и
// проводка WSClient (onMessage→gameStore, onAction→send, onNewGame→сброс).
// createClient инжектируется: реальный WSClient поверх MockWebSocket.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, test } from 'vitest'

import { loadCredentials, saveCredentials, type Credentials } from './lib/credentials'
import { resetGameState } from './stores/game.svelte'
import { WSClient, type WSConnectionCtor } from './transport/ws'
import { MockWebSocket } from '../tests/mockWebSocket'
import { stateFixture } from '../tests/fixtures'

import App from './App.svelte'

beforeEach(() => {
  localStorage.clear()
  resetGameState()
  MockWebSocket.reset()
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
