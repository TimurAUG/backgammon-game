// FRONTEND_SPEC #21 — GameOver.svelte показывается при gameOver != null с winner+kind.
// Card #19 — кнопка «Выйти» зовёт onExit (seat exit), без create-and-enter UI.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { describe, expect, test, vi } from 'vitest'

import type { Color, WinKind } from '../protocol/messages'

import GameOver from './GameOver.svelte'

describe('GameOver visibility (#21)', () => {
  test('GameOver_null_rendersNothing', () => {
    render(GameOver, { props: { gameOver: null, onExit: vi.fn() } })
    expect(screen.queryByTestId('game-over')).toBeNull()
  })

  test('GameOver_setValue_rendersDialog', () => {
    render(GameOver, {
      props: { gameOver: { winner: 'white', kind: 'oin' }, onExit: vi.fn() },
    })
    expect(screen.queryByTestId('game-over')).not.toBeNull()
  })
})

describe('GameOver winner & kind localization (#21)', () => {
  test.each<[Color, string]>([
    ['white', 'Белые'],
    ['black', 'Чёрные'],
  ])('GameOver_winner%s_showsColorLabel', (winner, label) => {
    render(GameOver, {
      props: { gameOver: { winner, kind: 'oin' }, onExit: vi.fn() },
    })
    expect(screen.getByTestId('game-over')).toHaveTextContent(label)
  })

  test.each<[WinKind, string]>([
    ['oin', 'Оин'],
    ['mars', 'Марс'],
    ['koks', 'Кокс'],
  ])('GameOver_kind%s_showsKindLabel', (kind, label) => {
    render(GameOver, {
      props: { gameOver: { winner: 'white', kind }, onExit: vi.fn() },
    })
    expect(screen.getByTestId('game-over')).toHaveTextContent(label)
  })

  test('GameOver_iWon_showsVictoryMessage', () => {
    render(GameOver, {
      props: {
        gameOver: { winner: 'white', kind: 'mars' },
        myColor: 'white',
        onExit: vi.fn(),
      },
    })
    expect(screen.getByTestId('game-over')).toHaveTextContent(/победил/i)
  })

  test('GameOver_iLost_showsDefeatMessage', () => {
    render(GameOver, {
      props: {
        gameOver: { winner: 'white', kind: 'mars' },
        myColor: 'black',
        onExit: vi.fn(),
      },
    })
    expect(screen.getByTestId('game-over')).toHaveTextContent(/проигр/i)
  })
})

describe('GameOver exit (#19 / #22)', () => {
  test('GameOver_showsExitNotNewGame', () => {
    render(GameOver, {
      props: { gameOver: { winner: 'black', kind: 'koks' }, onExit: vi.fn() },
    })

    expect(screen.getByTestId('action-exit')).toHaveTextContent('Выйти')
    expect(screen.queryByRole('button', { name: 'Новая игра' })).toBeNull()
    expect(screen.queryByTestId('action-new-game')).toBeNull()
  })

  test('GameOver_clickExit_callsOnExit', async () => {
    const onExit = vi.fn()
    render(GameOver, {
      props: { gameOver: { winner: 'black', kind: 'koks' }, onExit },
    })

    await fireEvent.click(screen.getByTestId('action-exit'))
    expect(onExit).toHaveBeenCalledOnce()
  })
})
