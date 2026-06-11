// FRONTEND_SPEC #31 — стор notifications: список тостов; pushNotification
// добавляет, dismissNotification убирает по id, resetNotifications очищает.
// Модульный $state, как connection/game.

import { beforeEach, describe, expect, test } from 'vitest'

import {
  notifications,
  pushNotification,
  dismissNotification,
  resetNotifications,
} from './notifications.svelte'

beforeEach(() => {
  resetNotifications()
})

describe('notifications store (#31)', () => {
  test('notifications_default_isEmpty', () => {
    expect(notifications.items).toEqual([])
  })

  test('notifications_push_appendsItemWithKindAndText', () => {
    pushNotification('opponentJoined', 'Соперник присоединился')

    expect(notifications.items).toEqual([
      { id: expect.any(Number), kind: 'opponentJoined', text: 'Соперник присоединился' },
    ])
  })

  test('notifications_pushTwice_keepsOrderWithDistinctIds', () => {
    const firstId = pushNotification('opponentJoined', 'Соперник присоединился')
    const secondId = pushNotification('yourRoll', 'Твой бросок')

    expect(notifications.items.map((n) => n.text)).toEqual([
      'Соперник присоединился',
      'Твой бросок',
    ])
    expect(firstId).not.toBe(secondId)
  })

  test('notifications_dismiss_removesOnlyMatchingId', () => {
    const firstId = pushNotification('opponentJoined', 'Соперник присоединился')
    pushNotification('yourRoll', 'Твой бросок')

    dismissNotification(firstId)

    expect(notifications.items.map((n) => n.text)).toEqual(['Твой бросок'])
  })

  test('notifications_reset_clearsAllItems', () => {
    pushNotification('yourRoll', 'Твой бросок')

    resetNotifications()

    expect(notifications.items).toEqual([])
  })
})
