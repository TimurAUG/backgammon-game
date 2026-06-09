// FRONTEND_SPEC #24a — Game.svelte: композиция Board + Dice + ActionBar +
// GameOver, всё из gameState; действия наружу через onAction/onNewGame.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { resetConnectionState, setConnectionState } from '../stores/connection.svelte'
import { applyServerMessage, resetGameState } from '../stores/game.svelte'
import { stateFixture } from '../../tests/fixtures'

import Game from './Game.svelte'

beforeEach(() => {
  resetGameState()
  resetConnectionState()
})

const noop = { onAction: vi.fn(), onNewGame: vi.fn() }

describe('Game composition (#24a)', () => {
  test('Game_stateBoard_rendersCheckers', () => {
    applyServerMessage(stateFixture())
    render(Game, { props: noop })

    expect(screen.getAllByTestId(/^checker-24-\d+$/)).toHaveLength(15)
    expect(screen.getAllByTestId(/^checker-12-\d+$/)).toHaveLength(15)
  })

  test('Game_diceInState_rendersDice', () => {
    applyServerMessage(
      stateFixture({
        status: 'waitingForMove',
        dice: { a: 3, b: 5, isDouble: false, remaining: [3, 5] },
      }),
    )
    render(Game, { props: noop })

    expect(screen.getByTestId('die-a')).toHaveTextContent('3')
    expect(screen.getByTestId('die-b')).toHaveTextContent('5')
  })

  test('Game_beforeJoined_hidesActionBar', () => {
    applyServerMessage(stateFixture())
    render(Game, { props: noop })

    expect(screen.queryByTestId('action-resign')).toBeNull()
  })
})

describe('Game action wiring (#24a)', () => {
  test('Game_rollForFirstClick_callsOnAction', async () => {
    const onAction = vi.fn()
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    render(Game, { props: { onAction, onNewGame: vi.fn() } })

    await fireEvent.click(screen.getByTestId('action-roll-for-first'))

    expect(onAction).toHaveBeenCalledExactlyOnceWith({ type: 'ROLL_FOR_FIRST' })
  })

  test('Game_boardMoveClicks_callOnActionWithMOVE', async () => {
    const onAction = vi.fn()
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(
      stateFixture({
        status: 'waitingForMove',
        dice: { a: 6, b: 5, isDouble: false, remaining: [6, 5] },
      }),
    )
    applyServerMessage({ type: 'LEGAL_MOVES', moves: [{ from: 24, to: 18, pip: 6 }] })
    render(Game, { props: { onAction, onNewGame: vi.fn() } })

    await fireEvent.click(screen.getByTestId('point-24'))
    await fireEvent.click(screen.getByTestId('point-18'))

    expect(onAction).toHaveBeenCalledExactlyOnceWith({ type: 'MOVE', from: 24, to: 18 })
  })

  test('Game_gameOver_rendersOverlayAndForwardsNewGame', async () => {
    const onNewGame = vi.fn()
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    applyServerMessage({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })
    render(Game, { props: { onAction: vi.fn(), onNewGame } })

    await fireEvent.click(screen.getByTestId('action-new-game'))

    expect(onNewGame).toHaveBeenCalledOnce()
  })
})

describe('Game switch-game (#27)', () => {
  test('Game_switchGameButton_callsOnNewGame', async () => {
    const onNewGame = vi.fn()
    applyServerMessage(stateFixture())
    render(Game, { props: { onAction: vi.fn(), onNewGame } })

    await fireEvent.click(screen.getByTestId('switch-game'))

    expect(onNewGame).toHaveBeenCalledOnce()
  })
})

describe('Game reconnect blocking (#26d)', () => {
  test('Game_reconnecting_disablesActionBar', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    setConnectionState('reconnecting')
    render(Game, { props: noop })

    expect(screen.getByTestId('action-resign')).toBeDisabled()
  })

  test('Game_connected_keepsActionBarEnabled', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    setConnectionState('connected')
    render(Game, { props: noop })

    expect(screen.getByTestId('action-resign')).toBeEnabled()
  })
})
