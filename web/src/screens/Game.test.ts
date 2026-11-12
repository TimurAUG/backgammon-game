// FRONTEND_SPEC #24a — Game.svelte: композиция Board + Dice + ActionBar +
// GameOver, всё из gameState; действия наружу через onAction/onNewGame.

import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
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
  onNewGame: vi.fn(),
  onLeave: vi.fn(),
  onLogout: vi.fn().mockResolvedValue(undefined),
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

describe('Game action wiring (#24a)', () => {
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

  test('Game_gameOver_rendersOverlayAndForwardsNewGame', async () => {
    const onNewGame = vi.fn()
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    applyServerMessage({ type: 'GAME_OVER', winner: 'white', kind: 'mars' })
    render(Game, { props: { ...noop, onNewGame } })

    await fireEvent.click(screen.getByTestId('action-new-game'))

    expect(onNewGame).toHaveBeenCalledOnce()
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
    // Победитель бросил кубики на ход (waitingForMove, кубики на столе) —
    // плашка первого броска больше не нужна.
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
    // Первый игрок (black) завершил ход → очередь white бросать
    // (waitingForRoll), isFirstMove[black]=false. Плашка не возвращается.
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
  // После розыгрыша победитель бросает кубики заново: бэк шлёт STATE с
  // status=waitingForRoll и dice=null (значения розыгрыша не переносятся).
  // Признак «розыгрыш состоялся» — firstRoll, а не наличие кубиков, поэтому
  // победитель видит «Бросить кубики» (ROLL), а не снова «Бросить за первый ход».
  test('Game_afterRollForFirst_winnerSeesRollDice', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage({ type: 'FIRST_ROLL', firstRoll: { white: 5, black: 3 } })
    applyServerMessage(stateFixture()) // waitingForRoll, turn=white, dice=null

    render(Game, { props: noop })

    expect(screen.queryByTestId('action-roll')).not.toBeNull()
    expect(screen.queryByTestId('action-roll-for-first')).toBeNull()
  })
})

describe('Game waiting for opponent first roll (#51)', () => {
  // Пользовательский баг: нажал «Бросить за первый ход» — и ничего не видно,
  // пока соперник не бросит. Теперь кнопка сменяется индикатором ожидания.
  test('Game_afterRollForFirstClick_showsWaitingIndicator', async () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture())
    render(Game, { props: noop })

    await fireEvent.click(screen.getByTestId('action-roll-for-first'))

    expect(screen.queryByTestId('action-roll-for-first')).toBeNull()
    expect(screen.getByTestId('waiting-first-roll')).toBeInTheDocument()
  })
})

describe('Game switch-game (#27)', () => {
  test('Game_switchGame_callsOnLeaveNotOnNewGame', async () => {
    const onNewGame = vi.fn()
    const onLeave = vi.fn()
    applyServerMessage(stateFixture())
    render(Game, { props: { ...noop, onNewGame, onLeave } })

    await fireEvent.click(screen.getByTestId('switch-game'))

    expect(onLeave).toHaveBeenCalledOnce()
    expect(onNewGame).not.toHaveBeenCalled()
  })
})

describe('Game logout (card #13)', () => {
  test('Game_toolbar_showsLogoutBesideSwitchGame', () => {
    applyServerMessage(stateFixture())
    render(Game, { props: noop })

    expect(screen.getByTestId('logout')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Выйти' })).toBeInTheDocument()
    expect(screen.getByTestId('switch-game')).toBeInTheDocument()
  })

  test('Game_logoutClick_callsOnLogoutNotOnLeave', async () => {
    const onLeave = vi.fn()
    const onLogout = vi.fn().mockResolvedValue(undefined)
    applyServerMessage(stateFixture())
    render(Game, { props: { ...noop, onLeave, onLogout } })

    await fireEvent.click(screen.getByTestId('logout'))

    expect(onLogout).toHaveBeenCalledOnce()
    expect(onLeave).not.toHaveBeenCalled()
  })

  test('Game_switchGame_doesNotCallOnLogout', async () => {
    const onLeave = vi.fn()
    const onLogout = vi.fn().mockResolvedValue(undefined)
    applyServerMessage(stateFixture())
    render(Game, { props: { ...noop, onLeave, onLogout } })

    await fireEvent.click(screen.getByTestId('switch-game'))

    expect(onLeave).toHaveBeenCalledOnce()
    expect(onLogout).not.toHaveBeenCalled()
  })

  test('Game_logoutFailure_staysOnBoardShowsAlert', async () => {
    const onLogout = vi.fn().mockRejectedValue(new Error('network'))
    const onLeave = vi.fn()
    applyServerMessage(stateFixture())
    render(Game, { props: { ...noop, onLeave, onLogout } })

    await fireEvent.click(screen.getByTestId('logout'))

    expect(await screen.findByRole('alert')).toHaveTextContent(/не удалось/i)
    expect(screen.getByTestId('point-1')).toBeInTheDocument()
    expect(onLeave).not.toHaveBeenCalled()
  })

  test('Game_logoutInFlight_ignoresRepeatClicks', async () => {
    let resolveLogout!: () => void
    const pending = new Promise<void>((resolve) => {
      resolveLogout = resolve
    })
    const onLogout = vi.fn().mockReturnValue(pending)
    applyServerMessage(stateFixture())
    render(Game, { props: { ...noop, onLogout } })

    await fireEvent.click(screen.getByTestId('logout'))
    await fireEvent.click(screen.getByTestId('logout'))
    expect(onLogout).toHaveBeenCalledTimes(1)

    resolveLogout()
    await waitFor(() => expect(onLogout).toHaveBeenCalledTimes(1))
  })
})

describe('Game seat nicks (card #16)', () => {
  test('Game_seatNicks_showsOwnNickColorAndOpponent', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: 'Bob' } }))
    render(Game, { props: noop })

    const block = screen.getByTestId('seat-nicks')
    expect(screen.getByTestId('seat-nicks-own')).toHaveTextContent('Alice · Белые')
    expect(screen.getByTestId('seat-nicks-opponent')).toHaveTextContent('Bob')
    expect(block).toBeInTheDocument()
    expect(screen.getByTestId('switch-game')).toBeInTheDocument()
  })

  test('Game_seatNicks_emptyOpponent_showsGuest', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: '' } }))
    render(Game, { props: noop })

    expect(screen.getByTestId('seat-nicks-opponent')).toHaveTextContent('Гость')
    expect(screen.getByTestId('seat-nicks-opponent')).not.toHaveTextContent('Чёрные')
  })

  test('Game_seatNicks_blackPerspective_swapsOwnAndOpponent', () => {
    applyServerMessage({ type: 'JOINED', color: 'black' })
    applyServerMessage(stateFixture({ nicks: { white: 'Alice', black: 'Bob' } }))
    render(Game, { props: noop })

    expect(screen.getByTestId('seat-nicks-own')).toHaveTextContent('Bob · Чёрные')
    expect(screen.getByTestId('seat-nicks-opponent')).toHaveTextContent('Alice')
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

describe('Game reconnect link (#30)', () => {
  // Личная ссылка для возврата в игру (в т.ч. с другого устройства):
  // содержит и gameId, и token.
  test('Game_rendersReconnectLinkWithGameIdAndToken', () => {
    applyServerMessage(stateFixture())
    render(Game, {
      props: {
        ...noop,
        gameId: 'g-42',
        token: 'tok-42',
      },
    })

    const link = screen.getByTestId('reconnect-link') as HTMLInputElement
    expect(link.value).toContain('?game=g-42')
    expect(link.value).toContain('token=tok-42')
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
    applyServerMessage(moveState('black')) // ход соперника — не мой бросок
    render(Game, { props: noop })
    expect(playRollCue).not.toHaveBeenCalled()

    applyServerMessage(rollState('white')) // соперник передал ход → мой бросок
    flushSync()

    expect(notifications.items.some((n) => n.text === 'Твой бросок')).toBe(true)
    expect(playRollCue).toHaveBeenCalledOnce()
  })

  test('Game_opponentRoll_doesNotNotify', () => {
    applyServerMessage({ type: 'JOINED', color: 'white' })
    applyServerMessage(moveState('white')) // мой ход двигать
    render(Game, { props: noop })

    applyServerMessage(rollState('black')) // я завершил → бросает соперник
    flushSync()

    expect(playRollCue).not.toHaveBeenCalled()
    expect(notifications.items).toHaveLength(0)
  })

  test('Game_firstRollStage_notifiesEvenIfNotMyTurn', () => {
    render(Game, { props: noop }) // монтируем до данных: started=false
    applyServerMessage({ type: 'JOINED', color: 'black' })
    // стадия розыгрыша: оба ещё не ходили, firstRoll нет, turn=white (не мой) —
    // бросают оба, поэтому «Твой бросок» уместен и чёрным тоже.
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
    applyServerMessage(rollState('white')) // повтор — нового перехода нет
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
    applyServerMessage(stateFixture()) // начальная позиция — ни у кого не все дома
    render(Game, { props: noop })

    expect(screen.queryByTestId('bear-off-counter')).toBeNull()
  })
})
