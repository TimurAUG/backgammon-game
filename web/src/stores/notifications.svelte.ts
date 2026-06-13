// notifications — стек тостов-уведомлений для UI. Слой UI поверх игрового
// состояния: события (соперник присоединился — из App; «твой бросок» —
// детект перехода в Game) пушат сюда, компонент Notifications рендерит.
// Модульный $state-объект, как gameState/connection: единственная мутация —
// через push/dismiss/reset.

export type NotificationKind = 'opponentJoined' | 'yourRoll' | 'chat' | 'turnSkipped'

export interface Notification {
  id: number
  kind: NotificationKind
  text: string
}

export const notifications = $state<{ items: Notification[] }>({ items: [] })

// Монотонный счётчик id — стабильный ключ для {#each} и адресации в dismiss.
// Без Date/Math.random, чтобы тесты были детерминированы.
let nextId = 1

// Возвращает id добавленного тоста — для адресного dismiss (в т.ч. программного).
export function pushNotification(kind: NotificationKind, text: string): number {
  const id = nextId++
  notifications.items.push({ id, kind, text })
  return id
}

export function dismissNotification(id: number): void {
  notifications.items = notifications.items.filter((n) => n.id !== id)
}

export function resetNotifications(): void {
  notifications.items = []
  nextId = 1
}
