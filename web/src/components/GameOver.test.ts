// FRONTEND_SPEC #21 — GameOver.svelte показывается при gameOver != null с winner+kind.
// FRONTEND_SPEC #22 — Кнопка «Новая игра» зовёт onNewGame callback.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { describe, expect, test, vi } from 'vitest'

import type { Color, WinKind } from '../protocol/messages'

import GameOver from './GameOver.svelte'

describe('GameOver visibility (#21)', () => {
  test('GameOver_null_rendersNothing', () => {
    render(GameOver, { props: { gameOver: null, onNewGame: vi.fn() } })
    expect(screen.queryByTestId('game-over')).toBeNull()
  })

  test('GameOver_setValue_rendersDialog', () => {
    render(GameOver, {
      props: { gameOver: { winner: 'white', kind: 'oin' }, onNewGame: vi.fn() },
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
      props: { gameOver: { winner, kind: 'oin' }, onNewGame: vi.fn() },
    })
    expect(screen.getByTestId('game-over')).toHaveTextContent(label)
  })

  test.each<[WinKind, string]>([
    ['oin', 'Оин'],
    ['mars', 'Марс'],
    ['koks', 'Кокс'],
  ])('GameOver_kind%s_showsKindLabel', (kind, label) => {
    render(GameOver, {
      props: { gameOver: { winner: 'white', kind }, onNewGame: vi.fn() },
    })
    expect(screen.getByTestId('game-over')).toHaveTextContent(label)
  })

  test('GameOver_iWon_showsVictoryMessage', () => {
    render(GameOver, {
      props: {
        gameOver: { winner: 'white', kind: 'mars' },
        myColor: 'white',
        onNewGame: vi.fn(),
      },
    })
    expect(screen.getByTestId('game-over')).toHaveTextContent(/победил/i)
  })

  test('GameOver_iLost_showsDefeatMessage', () => {
    render(GameOver, {
      props: {
        gameOver: { winner: 'white', kind: 'mars' },
        myColor: 'black',
        onNewGame: vi.fn(),
      },
    })
    expect(screen.getByTestId('game-over')).toHaveTextContent(/проигр/i)
  })
})

describe('GameOver new-game (#22)', () => {
  test('GameOver_clickNewGame_callsCallback', async () => {
    const onNewGame = vi.fn()
    render(GameOver, {
      props: { gameOver: { winner: 'black', kind: 'koks' }, onNewGame },
    })

    await fireEvent.click(screen.getByTestId('action-new-game'))
    expect(onNewGame).toHaveBeenCalledOnce()
  })
})
