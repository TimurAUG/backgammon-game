// FRONTEND_SPEC #17 — ActionBar показывает ROLL_FOR_FIRST по условиям и шлёт сообщение.
// FRONTEND_SPEC #18 — ROLL/END_TURN/RESIGN по своим условиям + шлют сообщения.
//
// ActionBar — чистый UI без знания о WSClient. Кнопки рендерятся
// по props (status/turn/myColor/rolledForFirst). Клики → onAction(msg).

import { fireEvent, render, screen } from '@testing-library/svelte'
import { describe, expect, test, vi } from 'vitest'

import type { ClientMessage } from '../protocol/messages'

import ActionBar from './ActionBar.svelte'

interface BaseProps {
  status: 'waitingForRoll' | 'waitingForMove' | 'finished'
  turn: 'white' | 'black'
  myColor: 'white' | 'black'
  rolledForFirst: boolean
  disabled: boolean
  onAction: (msg: ClientMessage) => void
}

function defaultProps(overrides: Partial<BaseProps> = {}): BaseProps {
  return {
    status: 'waitingForRoll',
    turn: 'white',
    myColor: 'white',
    rolledForFirst: false,
    disabled: false,
    onAction: vi.fn(),
    ...overrides,
  }
}

describe('ActionBar visibility (#17/#18 show conditions)', () => {
  test('ActionBar_firstRollPhase_showsRollForFirst', () => {
    render(ActionBar, { props: defaultProps({ rolledForFirst: false }) })

    expect(screen.queryByTestId('action-roll-for-first')).not.toBeNull()
    expect(screen.queryByTestId('action-roll')).toBeNull()
    expect(screen.queryByTestId('action-end-turn')).toBeNull()
    expect(screen.queryByTestId('action-resign')).not.toBeNull()
  })

  test('ActionBar_afterRolledForFirst_myTurn_showsRoll', () => {
    render(ActionBar, {
      props: defaultProps({
        rolledForFirst: true,
        status: 'waitingForRoll',
        turn: 'white',
        myColor: 'white',
      }),
    })

    expect(screen.queryByTestId('action-roll-for-first')).toBeNull()
    expect(screen.queryByTestId('action-roll')).not.toBeNull()
    expect(screen.queryByTestId('action-end-turn')).toBeNull()
  })

  test('ActionBar_afterRolledForFirst_opponentTurn_hidesRoll', () => {
    render(ActionBar, {
      props: defaultProps({
        rolledForFirst: true,
        status: 'waitingForRoll',
        turn: 'black',
        myColor: 'white',
      }),
    })

    expect(screen.queryByTestId('action-roll')).toBeNull()
    expect(screen.queryByTestId('action-end-turn')).toBeNull()
    expect(screen.queryByTestId('action-resign')).not.toBeNull()
  })

  test('ActionBar_waitingForMove_myTurn_showsEndTurn', () => {
    render(ActionBar, {
      props: defaultProps({
        rolledForFirst: true,
        status: 'waitingForMove',
        turn: 'white',
        myColor: 'white',
      }),
    })

    expect(screen.queryByTestId('action-roll')).toBeNull()
    expect(screen.queryByTestId('action-end-turn')).not.toBeNull()
  })

  test('ActionBar_finished_hidesAllActiveButtons', () => {
    render(ActionBar, {
      props: defaultProps({
        rolledForFirst: true,
        status: 'finished',
        turn: 'white',
        myColor: 'white',
      }),
    })

    expect(screen.queryByTestId('action-roll-for-first')).toBeNull()
    expect(screen.queryByTestId('action-roll')).toBeNull()
    expect(screen.queryByTestId('action-end-turn')).toBeNull()
    expect(screen.queryByTestId('action-resign')).toBeNull()
  })
})

describe('ActionBar click handlers (#17/#18)', () => {
  test('ActionBar_clickRollForFirst_sendsRollForFirst', async () => {
    const onAction = vi.fn()
    render(ActionBar, { props: defaultProps({ rolledForFirst: false, onAction }) })

    await fireEvent.click(screen.getByTestId('action-roll-for-first'))

    expect(onAction).toHaveBeenCalledWith({ type: 'ROLL_FOR_FIRST' })
  })

  test('ActionBar_clickRoll_sendsRoll', async () => {
    const onAction = vi.fn()
    render(ActionBar, {
      props: defaultProps({
        rolledForFirst: true,
        status: 'waitingForRoll',
        turn: 'white',
        myColor: 'white',
        onAction,
      }),
    })

    await fireEvent.click(screen.getByTestId('action-roll'))

    expect(onAction).toHaveBeenCalledWith({ type: 'ROLL' })
  })

  test('ActionBar_clickEndTurn_sendsEndTurn', async () => {
    const onAction = vi.fn()
    render(ActionBar, {
      props: defaultProps({
        rolledForFirst: true,
        status: 'waitingForMove',
        turn: 'white',
        myColor: 'white',
        onAction,
      }),
    })

    await fireEvent.click(screen.getByTestId('action-end-turn'))

    expect(onAction).toHaveBeenCalledWith({ type: 'END_TURN' })
  })

  test('ActionBar_clickResign_sendsResign', async () => {
    const onAction = vi.fn()
    render(ActionBar, { props: defaultProps({ onAction }) })

    await fireEvent.click(screen.getByTestId('action-resign'))

    expect(onAction).toHaveBeenCalledWith({ type: 'RESIGN' })
  })
})

describe('ActionBar waiting for opponent first roll (#51)', () => {
  // Сервер молчит, пока оба не пришлют ROLL_FOR_FIRST (FIRST_ROLL приходит
  // только после второго). Клиент сам помнит факт клика и заменяет кнопку
  // индикатором «ждём соперника», иначе после броска «ничего не происходит».
  test('ActionBar_firstRollPhase_beforeClick_noWaitingIndicator', () => {
    render(ActionBar, { props: defaultProps({ rolledForFirst: false }) })

    expect(screen.queryByTestId('action-roll-for-first')).not.toBeNull()
    expect(screen.queryByTestId('waiting-first-roll')).toBeNull()
  })

  test('ActionBar_clickRollForFirst_hidesButtonShowsWaiting', async () => {
    render(ActionBar, { props: defaultProps({ rolledForFirst: false }) })

    await fireEvent.click(screen.getByTestId('action-roll-for-first'))

    expect(screen.queryByTestId('action-roll-for-first')).toBeNull()
    expect(screen.queryByTestId('waiting-first-roll')).not.toBeNull()
  })

  test('ActionBar_afterRolledForFirst_hidesWaitingIndicator', async () => {
    const { rerender } = render(ActionBar, { props: defaultProps({ rolledForFirst: false }) })
    await fireEvent.click(screen.getByTestId('action-roll-for-first'))
    expect(screen.queryByTestId('waiting-first-roll')).not.toBeNull()

    // Пришёл FIRST_ROLL → розыгрыш состоялся: индикатор уступает место игре.
    await rerender(defaultProps({ rolledForFirst: true }))

    expect(screen.queryByTestId('waiting-first-roll')).toBeNull()
  })

  test('ActionBar_disabledClickRollForFirst_noWaitingIndicator', async () => {
    render(ActionBar, { props: defaultProps({ rolledForFirst: false, disabled: true }) })

    await fireEvent.click(screen.getByTestId('action-roll-for-first'))

    // При реконнекте бросок не ушёл — не притворяемся, что ждём соперника.
    expect(screen.queryByTestId('waiting-first-roll')).toBeNull()
    expect(screen.queryByTestId('action-roll-for-first')).not.toBeNull()
  })
})

describe('ActionBar disabled (#26c)', () => {
  test('ActionBar_disabled_disablesAllButtons', () => {
    render(ActionBar, { props: defaultProps({ rolledForFirst: false, disabled: true }) })

    expect(screen.getByTestId('action-roll-for-first')).toBeDisabled()
    expect(screen.getByTestId('action-resign')).toBeDisabled()
  })

  test('ActionBar_disabled_clickDoesNotFireOnAction', async () => {
    const onAction = vi.fn()
    render(ActionBar, { props: defaultProps({ disabled: true, onAction }) })

    await fireEvent.click(screen.getByTestId('action-resign'))

    expect(onAction).not.toHaveBeenCalled()
  })

  test('ActionBar_notDisabled_buttonsEnabled', () => {
    render(ActionBar, { props: defaultProps({ disabled: false }) })

    expect(screen.getByTestId('action-resign')).toBeEnabled()
  })
})
