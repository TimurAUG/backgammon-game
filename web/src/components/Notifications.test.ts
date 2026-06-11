// FRONTEND_SPEC #33 — Notifications: контейнер, рендерит тосты из стора
// notifications; закрытие тоста убирает его из стора.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, test } from 'vitest'

import { notifications, pushNotification, resetNotifications } from '../stores/notifications.svelte'

import Notifications from './Notifications.svelte'

beforeEach(() => {
  resetNotifications()
})

describe('Notifications (#33)', () => {
  test('Notifications_rendersOneToastPerStoreItem', () => {
    pushNotification('opponentJoined', 'Соперник присоединился')
    pushNotification('yourRoll', 'Твой бросок')

    render(Notifications)

    expect(screen.getByText('Соперник присоединился')).toBeInTheDocument()
    expect(screen.getByText('Твой бросок')).toBeInTheDocument()
  })

  test('Notifications_closeButtonClick_removesToastFromStore', async () => {
    pushNotification('yourRoll', 'Твой бросок')
    render(Notifications)

    await fireEvent.click(screen.getByRole('button', { name: 'Закрыть' }))

    expect(notifications.items).toHaveLength(0)
    expect(screen.queryByText('Твой бросок')).toBeNull()
  })
})
