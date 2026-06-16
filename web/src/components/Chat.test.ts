// FRONTEND_SPEC #37 — Chat рендерит ленту из стора; своё/чужое по myColor.
// FRONTEND_SPEC #38 — ввод + отправка (Enter/кнопка) → onAction CHAT, очистка;
//                     пустое/пробельное не шлётся.
// FRONTEND_SPEC #39 — сворачивание + бейдж непрочитанных.
// FRONTEND_SPEC #45 — автоскролл ленты к последнему сообщению (новое сообщение
//                     и разворот панели).
//
// Chat — UI поверх стора chat: лента из chat.messages, отправка через onAction.

import { fireEvent, render, screen } from '@testing-library/svelte'
import { flushSync } from 'svelte'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { applyChat, chat, resetChat, setChatOpen } from '../stores/chat.svelte'

import Chat from './Chat.svelte'

beforeEach(() => {
  resetChat()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Chat messages (#37)', () => {
  test('Chat_open_rendersMessagesFromStore', () => {
    setChatOpen(true)
    applyChat({ sender: 'white', text: 'привет' })
    applyChat({ sender: 'black', text: 'ответ' })

    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    expect(screen.getByText('привет')).toBeInTheDocument()
    expect(screen.getByText('ответ')).toBeInTheDocument()
  })

  test('Chat_ownMessageMarkedMine_opponentNot', () => {
    setChatOpen(true)
    applyChat({ sender: 'white', text: 'моё' })
    applyChat({ sender: 'black', text: 'чужое' })

    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    const mine = screen.getByText('моё').closest('[data-testid="chat-message"]')
    const theirs = screen.getByText('чужое').closest('[data-testid="chat-message"]')
    expect(mine).toHaveClass('mine')
    expect(theirs).not.toHaveClass('mine')
  })
})

describe('Chat input (#38)', () => {
  test('Chat_clickSend_sendsChatActionAndClearsInput', async () => {
    setChatOpen(true)
    const onAction = vi.fn()
    render(Chat, { props: { myColor: 'white', onAction } })

    const input = screen.getByTestId('chat-input') as HTMLInputElement
    await fireEvent.input(input, { target: { value: 'привет' } })
    await fireEvent.click(screen.getByTestId('chat-send'))

    expect(onAction).toHaveBeenCalledWith({ type: 'CHAT', text: 'привет' })
    expect(input.value).toBe('')
  })

  test('Chat_enterKey_sendsChatAction', async () => {
    setChatOpen(true)
    const onAction = vi.fn()
    render(Chat, { props: { myColor: 'white', onAction } })

    const input = screen.getByTestId('chat-input')
    await fireEvent.input(input, { target: { value: 'по энтеру' } })
    await fireEvent.keyDown(input, { key: 'Enter' })

    expect(onAction).toHaveBeenCalledWith({ type: 'CHAT', text: 'по энтеру' })
  })

  test('Chat_blankText_doesNotSend', async () => {
    setChatOpen(true)
    const onAction = vi.fn()
    render(Chat, { props: { myColor: 'white', onAction } })

    const input = screen.getByTestId('chat-input')
    await fireEvent.input(input, { target: { value: '   ' } })
    await fireEvent.click(screen.getByTestId('chat-send'))

    expect(onAction).not.toHaveBeenCalled()
  })
})

describe('Chat collapse + unread badge (#39)', () => {
  test('Chat_closedByDefault_showsToggleNotPanel', () => {
    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    expect(screen.getByTestId('chat-toggle')).toBeInTheDocument()
    expect(screen.queryByTestId('chat-panel')).toBeNull()
  })

  test('Chat_closedWithUnread_showsBadgeCount', () => {
    applyChat({ sender: 'black', text: 'раз' })
    applyChat({ sender: 'black', text: 'два' })

    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    expect(screen.getByTestId('chat-unread')).toHaveTextContent('2')
  })

  test('Chat_noUnread_hidesBadge', () => {
    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    expect(screen.queryByTestId('chat-unread')).toBeNull()
  })

  test('Chat_clickToggle_opensPanelAndMarksRead', async () => {
    applyChat({ sender: 'black', text: 'привет' })
    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    await fireEvent.click(screen.getByTestId('chat-toggle'))

    expect(screen.getByTestId('chat-panel')).toBeInTheDocument()
    expect(chat.open).toBe(true)
    expect(chat.unread).toBe(0)
  })

  test('Chat_clickCloseWhenOpen_collapses', async () => {
    setChatOpen(true)
    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    await fireEvent.click(screen.getByTestId('chat-close'))

    expect(chat.open).toBe(false)
  })
})

describe('Chat autoscroll (#45)', () => {
  // jsdom не делает layout: scrollHeight всегда 0. Мокаем геттер, чтобы «низ
  // ленты» был ненулевым и автоскролл был наблюдаем через scrollTop (его jsdom
  // хранит как обычное свойство). Без реализации scrollTop остаётся 0 — тест падает.
  test('Chat_newMessage_scrollsListToBottom', () => {
    vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue(1000)
    setChatOpen(true)
    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    const list = screen.getByRole('list')
    list.scrollTop = 0 // до нового сообщения лента не прокручена к низу

    applyChat({ sender: 'black', text: 'свежее' })
    flushSync()

    expect(list.scrollTop, 'лента прокручена к низу (scrollHeight) после нового сообщения').toBe(
      1000,
    )
  })

  test('Chat_openPanel_scrollsListToBottom', async () => {
    vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue(1000)
    applyChat({ sender: 'black', text: 'пришло пока чат был свёрнут' })
    render(Chat, { props: { myColor: 'white', onAction: vi.fn() } })

    await fireEvent.click(screen.getByTestId('chat-toggle'))

    const list = screen.getByRole('list')
    expect(list.scrollTop, 'при развороте панели лента сразу прокручена к низу').toBe(1000)
  })
})
