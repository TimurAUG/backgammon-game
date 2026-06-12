// FRONTEND_SPEC #36 — стор chat: messages/unread/open; applyChat добавляет
// сообщение (++unread, если панель закрыта), applyChatHistory заменяет ленту,
// setChatOpen(true) обнуляет unread, resetChat очищает. Модульный $state, как
// notifications/connection.

import { beforeEach, describe, expect, test } from 'vitest'

import { chat, applyChat, applyChatHistory, setChatOpen, resetChat } from './chat.svelte'

beforeEach(() => {
  resetChat()
})

describe('chat store (#36)', () => {
  test('chat_default_isEmptyClosedNoUnread', () => {
    expect(chat.messages).toEqual([])
    expect(chat.unread).toBe(0)
    expect(chat.open).toBe(false)
  })

  test('chat_applyChat_appendsMessage', () => {
    applyChat({ sender: 'white', text: 'привет' })

    expect(chat.messages).toEqual([{ sender: 'white', text: 'привет' }])
  })

  test('chat_applyChatWhileClosed_incrementsUnread', () => {
    applyChat({ sender: 'black', text: 'раз' })
    applyChat({ sender: 'black', text: 'два' })

    expect(chat.unread).toBe(2)
  })

  test('chat_applyChatWhileOpen_keepsUnreadZero', () => {
    setChatOpen(true)

    applyChat({ sender: 'black', text: 'видно сразу' })

    expect(chat.unread).toBe(0)
  })

  test('chat_applyChatHistory_replacesMessages', () => {
    applyChat({ sender: 'white', text: 'старое' })

    applyChatHistory([
      { sender: 'white', text: 'раз' },
      { sender: 'black', text: 'два' },
    ])

    expect(chat.messages).toEqual([
      { sender: 'white', text: 'раз' },
      { sender: 'black', text: 'два' },
    ])
  })

  test('chat_setChatOpenTrue_clearsUnread', () => {
    applyChat({ sender: 'black', text: 'непрочитанное' })
    expect(chat.unread).toBe(1)

    setChatOpen(true)

    expect(chat.unread).toBe(0)
  })

  test('chat_reset_clearsEverything', () => {
    applyChat({ sender: 'white', text: 'x' })
    setChatOpen(true)

    resetChat()

    expect(chat.messages).toEqual([])
    expect(chat.unread).toBe(0)
    expect(chat.open).toBe(false)
  })
})
