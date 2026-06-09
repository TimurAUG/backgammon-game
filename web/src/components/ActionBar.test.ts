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
    render(
      ActionBar,
      {
        props: defaultProps({
          rolledForFirst: true,
          status: 'waitingForRoll',
          turn: 'white',
          myColor: 'white',
        }),
      },
    )

    expect(screen.queryByTestId('action-roll-for-first')).toBeNull()
    expect(screen.queryByTestId('action-roll')).not.toBeNull()
    expect(screen.queryByTestId('action-end-turn')).toBeNull()
  })

  test('ActionBar_afterRolledForFirst_opponentTurn_hidesRoll', () => {
    render(
      ActionBar,
      {
        props: defaultProps({
          rolledForFirst: true,
          status: 'waitingForRoll',
          turn: 'black',
          myColor: 'white',
        }),
      },
    )

    expect(screen.queryByTestId('action-roll')).toBeNull()
    expect(screen.queryByTestId('action-end-turn')).toBeNull()
    expect(screen.queryByTestId('action-resign')).not.toBeNull()
  })

  test('ActionBar_waitingForMove_myTurn_showsEndTurn', () => {
    render(
      ActionBar,
      {
        props: defaultProps({
          rolledForFirst: true,
          status: 'waitingForMove',
          turn: 'white',
          myColor: 'white',
        }),
      },
    )

    expect(screen.queryByTestId('action-roll')).toBeNull()
    expect(screen.queryByTestId('action-end-turn')).not.toBeNull()
  })

  test('ActionBar_finished_hidesAllActiveButtons', () => {
    render(
      ActionBar,
      {
        props: defaultProps({
          rolledForFirst: true,
          status: 'finished',
          turn: 'white',
          myColor: 'white',
        }),
      },
    )

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
    render(
      ActionBar,
      {
        props: defaultProps({
          rolledForFirst: true,
          status: 'waitingForRoll',
          turn: 'white',
          myColor: 'white',
          onAction,
        }),
      },
    )

    await fireEvent.click(screen.getByTestId('action-roll'))

    expect(onAction).toHaveBeenCalledWith({ type: 'ROLL' })
  })

  test('ActionBar_clickEndTurn_sendsEndTurn', async () => {
    const onAction = vi.fn()
    render(
      ActionBar,
      {
        props: defaultProps({
          rolledForFirst: true,
          status: 'waitingForMove',
          turn: 'white',
          myColor: 'white',
          onAction,
        }),
      },
    )

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
