// chat — стор чата партии. Слой UI поверх протокольных сообщений: App пушит
// сюда входящие CHAT/CHAT_HISTORY, компонент Chat рендерит ленту и бейдж
// непрочитанных. Модульный $state-объект, как gameState/notifications:
// единственная мутация — через функции ниже.
//
// Сервер — источник правды: лента содержит ТОЛЬКО то, что прислал сервер
// (никакого оптимистичного эха своего сообщения — ждём CHAT обратно).

import type { ChatMessage } from '../protocol/messages'

export const chat = $state<{ messages: ChatMessage[]; unread: number; open: boolean }>({
  messages: [],
  unread: 0,
  open: false,
})

// applyChat добавляет входящее сообщение в ленту. Пока панель закрыта —
// растит счётчик непрочитанных (для бейджа); открытая панель показывает
// сообщение сразу, поэтому unread не трогаем.
export function applyChat(message: ChatMessage): void {
  chat.messages.push(message)
  if (!chat.open) chat.unread++
}

// applyChatHistory заменяет ленту присланной историей (при JOIN/реконнекте).
// unread не трогаем: история — это уже бывшая переписка, а не новые
// непрочитанные сообщения.
export function applyChatHistory(history: ChatMessage[]): void {
  chat.messages = [...history]
}

// setChatOpen разворачивает/сворачивает панель. Разворот = «прочитано»:
// обнуляем счётчик непрочитанных.
export function setChatOpen(open: boolean): void {
  chat.open = open
  if (open) chat.unread = 0
}

export function resetChat(): void {
  chat.messages = []
  chat.unread = 0
  chat.open = false
}
