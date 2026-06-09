// FRONTEND_SPEC #15 — Dice.svelte рендер двух кубиков (dice.a и dice.b).
// FRONTEND_SPEC #16 — Dice.svelte рендер remaining (при дубле — 4 одинаковых).

import { render, screen } from '@testing-library/svelte'
import { describe, expect, test } from 'vitest'

import type { Dice as DiceValue } from '../protocol/messages'

import Dice from './Dice.svelte'

describe('Dice both die display (#15)', () => {
  test('Dice_showsBothDieValues', () => {
    const dice: DiceValue = { a: 3, b: 5, isDouble: false, remaining: [3, 5] }
    render(Dice, { props: { dice } })

    expect(screen.getByTestId('die-a')).toHaveTextContent('3')
    expect(screen.getByTestId('die-b')).toHaveTextContent('5')
  })

  test('Dice_double_showsSameValueOnBoth', () => {
    const dice: DiceValue = { a: 6, b: 6, isDouble: true, remaining: [6, 6, 6, 6] }
    render(Dice, { props: { dice } })

    expect(screen.getByTestId('die-a')).toHaveTextContent('6')
    expect(screen.getByTestId('die-b')).toHaveTextContent('6')
  })

  test('Dice_nullDice_rendersNoDie', () => {
    render(Dice, { props: { dice: null } })

    expect(screen.queryByTestId('die-a')).toBeNull()
    expect(screen.queryByTestId('die-b')).toBeNull()
  })
})

describe('Dice remaining pips (#16)', () => {
  test('Dice_normalRoll_remainingHasTwoEntries', () => {
    const dice: DiceValue = { a: 3, b: 5, isDouble: false, remaining: [3, 5] }
    render(Dice, { props: { dice } })

    const items = screen.getAllByTestId(/^remaining-/)
    expect(items).toHaveLength(2)
    expect(items[0]).toHaveTextContent('3')
    expect(items[1]).toHaveTextContent('5')
  })

  test('Dice_double_remainingHasFourSameEntries', () => {
    const dice: DiceValue = { a: 4, b: 4, isDouble: true, remaining: [4, 4, 4, 4] }
    render(Dice, { props: { dice } })

    const items = screen.getAllByTestId(/^remaining-/)
    expect(items).toHaveLength(4)
    for (const item of items) {
      expect(item).toHaveTextContent('4')
    }
  })

  test('Dice_emptyRemaining_showsNoPips', () => {
    const dice: DiceValue = { a: 3, b: 5, isDouble: false, remaining: [] }
    render(Dice, { props: { dice } })

    expect(screen.queryAllByTestId(/^remaining-/)).toHaveLength(0)
  })

  test('Dice_partiallyUsedRoll_showsOnlyRemainingPips', () => {
    // Игрок уже использовал 3 из двух пипсов.
    const dice: DiceValue = { a: 3, b: 5, isDouble: false, remaining: [5] }
    render(Dice, { props: { dice } })

    const items = screen.getAllByTestId(/^remaining-/)
    expect(items).toHaveLength(1)
    expect(items[0]).toHaveTextContent('5')
  })
})
