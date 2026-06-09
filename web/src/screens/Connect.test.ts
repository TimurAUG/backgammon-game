// FRONTEND_SPEC #23 — Connect.svelte: форма gameId+token, сохранение
// в localStorage, колбек onConnect с введёнными кредами.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { loadCredentials } from '../lib/credentials'

import Connect from './Connect.svelte'

beforeEach(() => {
  localStorage.clear()
})

async function fillAndSubmit(gameId: string, token: string): Promise<void> {
  await fireEvent.input(screen.getByLabelText('ID игры'), { target: { value: gameId } })
  await fireEvent.input(screen.getByLabelText('Токен'), { target: { value: token } })
  await fireEvent.click(screen.getByRole('button', { name: 'Подключиться' }))
}

describe('Connect form (#23)', () => {
  test('Connect_renders_formFields', () => {
    render(Connect, { props: { onConnect: vi.fn() } })
    expect(screen.getByLabelText('ID игры')).toBeInTheDocument()
    expect(screen.getByLabelText('Токен')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Подключиться' })).toBeInTheDocument()
  })

  test('Connect_submitWithFilledFields_callsOnConnect', async () => {
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })

    await fillAndSubmit('g-42', 'tok-abc')

    expect(onConnect).toHaveBeenCalledExactlyOnceWith({ gameId: 'g-42', token: 'tok-abc' })
  })

  test('Connect_submitWithFilledFields_persistsToLocalStorage', async () => {
    render(Connect, { props: { onConnect: vi.fn() } })

    await fillAndSubmit('g-42', 'tok-abc')

    expect(loadCredentials()).toEqual({ gameId: 'g-42', token: 'tok-abc' })
  })

  test('Connect_submitWithEmptyGameId_doesNotCallOnConnect', async () => {
    const onConnect = vi.fn()
    render(Connect, { props: { onConnect } })

    await fillAndSubmit('   ', 'tok-abc')

    expect(onConnect).not.toHaveBeenCalled()
  })
})
