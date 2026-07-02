// FRONTEND_SPEC #52 — BearOffCounter: счётчик оставшихся к сбросу шашек.
// Показывается для цвета только в фазе сброса (allHome[c]) в формате
// «осталось/всего» = (15 − borneOff[c])/15. Оба цвета — независимо.
// «Все дома» приходит из STATE (домен AllInHome) — клиент не считает правило.

import { render, screen } from '@testing-library/svelte'
import { describe, expect, test } from 'vitest'

import BearOffCounter from './BearOffCounter.svelte'

describe('BearOffCounter (#52)', () => {
  test('BearOffCounter_colorAllHome_showsRemainingOutOf15', () => {
    render(BearOffCounter, {
      props: { borneOff: { white: 4, black: 0 }, allHome: { white: true, black: false } },
    })

    const row = screen.getByTestId('bear-off-remaining-white')
    expect(row).toHaveTextContent('Белые')
    expect(row).toHaveTextContent('11/15') // 15 − 4 сброшенных = 11 осталось
  })

  test('BearOffCounter_colorNotAllHome_hidesItsRow', () => {
    render(BearOffCounter, {
      props: { borneOff: { white: 0, black: 0 }, allHome: { white: false, black: true } },
    })

    expect(screen.queryByTestId('bear-off-remaining-white')).toBeNull()
    expect(screen.queryByTestId('bear-off-remaining-black')).not.toBeNull()
  })

  test('BearOffCounter_neitherAllHome_rendersNothing', () => {
    render(BearOffCounter, {
      props: { borneOff: { white: 0, black: 0 }, allHome: { white: false, black: false } },
    })

    expect(screen.queryByTestId('bear-off-counter')).toBeNull()
  })

  test('BearOffCounter_bothAllHome_showsBothRowsWithOwnCounts', () => {
    render(BearOffCounter, {
      props: { borneOff: { white: 2, black: 7 }, allHome: { white: true, black: true } },
    })

    expect(screen.getByTestId('bear-off-remaining-white')).toHaveTextContent('13/15')
    expect(screen.getByTestId('bear-off-remaining-black')).toHaveTextContent('8/15')
  })
})
