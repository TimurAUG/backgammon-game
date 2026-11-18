// FRONTEND_SPEC #24a — Game.svelte: композиция Board + Dice + ActionBar +
// GameOver, всё из gameState; действия наружу через onAction/onExit.
// Card #19 — strip toolbar chrome; GameOver seat exit via onExit.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { flushSync } from 'svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { resetConnectionState, setConnectionState } from '../stores/connection.svelte'
import { applyServerMessage, resetGameState } from '../stores/game.svelte'
import { notifications, resetNotifications } from '../stores/notifications.svelte'
import { resetChat } from '../stores/chat.svelte'
import { playRollCue } from '../lib/sound'
import { stateFixture } from '../../tests/fixtures'

import Game from './Game.svelte'

vi.mock('../lib/sound', () => ({ playRollCue: vi.fn() }))

beforeEach(() => {
  resetGameState()
  resetConnectionState()
  resetNotifications()
  resetChat()
  vi.clearAllMocks()
})

const noop = {
  onAction: vi.fn(),
  onExit: vi.fn(),
}

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

describe('Game action wiring (#24a / #19)', () => {
  test('Game_rollForFirstClick_callsOnAction', async () => {
    const onAction = vi.fn()
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    render(Game, { props: { ...noop, onAction } })

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
    render(Game, { props: { ...noop, onAction } })

    await fireEvent.click(screen.getByTestId('point-24'))
    await fireEvent.click(screen.getByTestId('point-18'))

    expect(onAction).toHaveBeenCalledExactlyOnceWith({ type: 'MOVE', from: 24, to: 18 })
  })

  test('Game_gameOver_rendersOverlayAndForwardsOnExit', async () => {
    const onExit = vi.fn()
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    applyServerMessage({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })
    render(Game, { props: { ...noop, onExit } })

    await fireEvent.click(screen.getByTestId('action-exit'))

    expect(onExit).toHaveBeenCalledOnce()
  })
})

describe('Game toolbar (#19)', () => {
  test('Game_toolbar_hasNoReturnCopyLogoutSwitch', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: 'Bob' } }))
    render(Game, { props: noop })

    expect(screen.getByTestId('seat-nicks')).toBeInTheDocument()
    expect(screen.queryByTestId('reconnect-link')).toBeNull()
    expect(screen.queryByTestId('copy-reconnect')).toBeNull()
    expect(screen.queryByTestId('logout')).toBeNull()
    expect(screen.queryByTestId('switch-game')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Копировать' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Выйти' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Сменить игру' })).toBeNull()
  })
})

describe('Game first-roll banner (#2)', () => {
  test('Game_firstRoll_showsWhoRolledWhat', () => {
    applyServerMessage({ type: 'FIRST_ROLL', firstRoll: { white: 5, black: 3 } })
    applyServerMessage(stateFixture()) // waitingForRoll — окно до броска победителя
    render(Game, { props: noop })

    const banner = screen.getByTestId('first-roll-banner')
    expect(banner).toHaveTextContent('5')
    expect(banner).toHaveTextContent('3')
  })

  test('Game_afterWinnerRolls_hidesBanner', () => {
    applyServerMessage({ type: 'FIRST_ROLL', firstRoll: { white: 4, black: 6 } })
    applyServerMessage(
      stateFixture({
        status: 'waitingForMove',
        turn: 'black',
        dice: { a: 6, b: 4, isDouble: false, remaining: [6, 4] },
      }),
    )
    render(Game, { props: noop })

    expect(screen.queryByTestId('first-roll-banner')).toBeNull()
  })

  test('Game_afterFirstEndTurn_keepsBannerHidden', () => {
    applyServerMessage({ type: 'FIRST_ROLL', firstRoll: { white: 4, black: 6 } })
    applyServerMessage(
      stateFixture({
        status: 'waitingForRoll',
        turn: 'white',
        isFirstMove: { white: true, black: false },
      }),
    )
    render(Game, { props: noop })

    expect(screen.queryByTestId('first-roll-banner')).toBeNull()
  })

  test('Game_noFirstRoll_noBanner', () => {
    applyServerMessage(stateFixture())
    render(Game, { props: noop })

    expect(screen.queryByTestId('first-roll-banner')).toBeNull()
  })
})

describe('Game first-roll winner re-roll (#2)', () => {
  test('Game_afterRollForFirst_winnerSeesRollDice', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage({ type: 'FIRST_ROLL', firstRoll: { white: 5, black: 3 } })
    applyServerMessage(stateFixture())

    render(Game, { props: noop })

    expect(screen.queryByTestId('action-roll')).not.toBeNull()
    expect(screen.queryByTestId('action-roll-for-first')).toBeNull()
  })
})

describe('Game waiting for opponent first roll (#51)', () => {
  test('Game_afterRollForFirstClick_showsWaitingIndicator', async () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    render(Game, { props: noop })

    await fireEvent.click(screen.getByTestId('action-roll-for-first'))

    expect(screen.queryByTestId('action-roll-for-first')).toBeNull()
    expect(screen.getByTestId('waiting-first-roll')).toBeInTheDocument()
  })
})

describe('Game seat nicks (card #16 / #19)', () => {
  test('Game_seatNicks_showsOwnNickColorAndOpponent', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: 'Bob' } }))
    render(Game, { props: noop })

    const block = screen.getByTestId('seat-nicks')
    expect(screen.getByTestId('seat-nicks-own')).toHaveTextContent(/^Alice · Белые$/)
    expect(screen.getByTestId('seat-nicks-opponent')).toHaveTextContent(/^Bob$/)
    expect(block).toBeInTheDocument()
  })

  test('Game_seatNicks_isFirstInHeader', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: 'Bob' } }))
    render(Game, { props: noop })

    const header = screen.getByTestId('seat-nicks').closest('header')
    expect(header).not.toBeNull()
    const kids = Array.from(header!.children)
    expect(kids[0]).toBe(screen.getByTestId('seat-nicks'))
  })

  test('Game_seatNicks_emptyOpponent_showsWaitingForPlayer', () => {
    // Card #25 AC2 — empty opponent nick → italic «Ожидание игрока...», not «Гость».
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: '' } }))
    render(Game, { props: noop })

    const opp = screen.getByTestId('seat-nicks-opponent')
    expect(opp).toHaveTextContent(/^Ожидание игрока\.\.\.$/)
    expect(opp).toHaveClass('seat-nicks-waiting')
    expect(opp).not.toHaveTextContent('Гость')
    expect(opp).not.toHaveTextContent('Чёрные')
  })

  test('Game_seatNicks_stateUpdatesOpponentNickWithoutReload', () => {
    // Card #25 AC1 — nick STATE after join updates opponent line in place.
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: '' } }))
    render(Game, { props: noop })
    expect(screen.getByTestId('seat-nicks-opponent')).toHaveTextContent(/^Ожидание игрока\.\.\.$/)

    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: 'Bob' } }))
    flushSync()
    expect(screen.getByTestId('seat-nicks-opponent')).toHaveTextContent(/^Bob$/)
  })

  test('Game_seatNicks_blackPerspective_swapsOwnAndOpponent', () => {
    applyServerMessage({ type: 'JOINED', color: 'black' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: 'Bob' } }))
    render(Game, { props: noop })

    expect(screen.getByTestId('seat-nicks-own')).toHaveTextContent(/^Bob · Чёрные$/)
    expect(screen.getByTestId('seat-nicks-opponent')).toHaveTextContent(/^Alice$/)
  })

  test('Game_firstRollBanner_stillUsesColorWords', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: 'Bob' } }))
    applyServerMessage({ type: 'FIRST_ROLL', firstRoll: { white: 5, black: 3 } })
    render(Game, { props: noop })

    const banner = screen.getByTestId('first-roll-banner')
    expect(banner).toHaveTextContent('Белые')
    expect(banner).toHaveTextContent('Чёрные')
    expect(banner).not.toHaveTextContent('Alice')
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

describe('Game reconnecting banner (persistence)', () => {
  test('Game_reconnecting_showsBanner', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    setConnectionState('reconnecting')
    render(Game, { props: noop })

    expect(screen.getByTestId('reconnecting-banner')).toHaveTextContent('Переподключение…')
  })

  test('Game_connected_noBanner', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    setConnectionState('connected')
    render(Game, { props: noop })

    expect(screen.queryByTestId('reconnecting-banner')).toBeNull()
  })
})

describe('Game your-roll cue (#34b)', () => {
  const moveState = (turn: 'white' | 'black') =>
    stateFixture({
      status: 'waitingForMove',
      turn,
      isFirstMove: { white: false, black: false },
      dice: { a: 3, b: 5, isDouble: false, remaining: [3, 5] },
    })

  const rollState = (turn: 'white' | 'black') =>
    stateFixture({
      status: 'waitingForRoll',
      turn,
      isFirstMove: { white: false, black: false },
    })

  test('Game_becomesMyRoll_pushesYourRollToastAndPlaysSound', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(moveState('black'))
    render(Game, { props: noop })
    expect(playRollCue).not.toHaveBeenCalled()

    applyServerMessage(rollState('white'))
    flushSync()

    expect(notifications.items.some((n) => n.text === 'Твой бросок')).toBe(true)
    expect(playRollCue).toHaveBeenCalledOnce()
  })

  test('Game_opponentRoll_doesNotNotify', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(moveState('white'))
    render(Game, { props: noop })

    applyServerMessage(rollState('black'))
    flushSync()

    expect(playRollCue).not.toHaveBeenCalled()
    expect(notifications.items).toHaveLength(0)
  })

  test('Game_firstRollStage_notifiesEvenIfNotMyTurn', () => {
    render(Game, { props: noop })
    applyServerMessage({ type: 'JOINED', color: 'black' })
    applyServerMessage(stateFixture({ status: 'waitingForRoll', turn: 'white' }))
    flushSync()

    expect(notifications.items.some((n) => n.text === 'Твой бросок')).toBe(true)
    expect(playRollCue).toHaveBeenCalledOnce()
  })

  test('Game_sameMyRollStateTwice_notifiesOnce', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(moveState('black'))
    render(Game, { props: noop })

    applyServerMessage(rollState('white'))
    flushSync()
    applyServerMessage(rollState('white'))
    flushSync()

    expect(playRollCue).toHaveBeenCalledOnce()
  })
})

describe('Game chat panel (#40)', () => {
  test('Game_rendersChatToggle', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    render(Game, { props: noop })

    expect(screen.getByTestId('chat-toggle')).toBeInTheDocument()
  })

  test('Game_chatSend_forwardsChatActionThroughOnAction', async () => {
    const onAction = vi.fn()
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    render(Game, { props: { ...noop, onAction } })

    await fireEvent.click(screen.getByTestId('chat-toggle'))
    await fireEvent.input(screen.getByTestId('chat-input'), { target: { value: 'привет' } })
    await fireEvent.click(screen.getByTestId('chat-send'))

    expect(onAction).toHaveBeenCalledWith({ type: 'CHAT', text: 'привет' })
  })
})

describe('Game bear-off counter (#52)', () => {
  test('Game_colorAllHome_showsRemainingCounter', () => {
    applyServerMessage(
      stateFixture({ allHome: { white: true, black: false }, borneOff: { white: 4, black: 0 } }),
    )
    render(Game, { props: noop })

    expect(screen.getByTestId('bear-off-remaining-white')).toHaveTextContent('11/15')
  })

  test('Game_noneAllHome_noBearOffCounter', () => {
    applyServerMessage(stateFixture())
    render(Game, { props: noop })

    expect(screen.queryByTestId('bear-off-counter')).toBeNull()
  })
})
