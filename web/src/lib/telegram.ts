// Интеграция с Telegram WebApp. Когда игру открывают внутри Telegram (Mini App
// или встроенный браузер-шит на iOS), вертикальный свайп по содержимому
// сворачивает/закрывает окно и перехватывает drag шашек. Telegram даёт API,
// чтобы это отключить — `disableVerticalSwipes` (Bot API 7.7+). Хедер шита всё
// равно остаётся свайпабельным (закрыть окно можно за него) — это by design.
//
// Полный объект `window.Telegram.WebApp` инжектит telegram-web-app.js (подключён
// в index.html). Вне Telegram `platform === 'unknown'` → ничего не делаем.

// Минимальный срез нужного нам API (методы опциональны — на старых клиентах
// disableVerticalSwipes может отсутствовать).
export interface TelegramWebApp {
  platform?: string
  ready?: () => void
  expand?: () => void
  disableVerticalSwipes?: () => void
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp }
  }
}

// setupTelegram разворачивает Mini App на всю высоту и отключает вертикальный
// свайп Telegram. No-op вне Telegram и при отсутствии методов (старый клиент).
export function setupTelegram(webApp: TelegramWebApp | undefined): void {
  if (webApp === undefined) return
  if (webApp.platform === undefined || webApp.platform === 'unknown') return
  webApp.ready?.()
  webApp.expand?.()
  webApp.disableVerticalSwipes?.()
}
