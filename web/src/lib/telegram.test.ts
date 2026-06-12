// Настройка Telegram WebApp: внутри Telegram-шита свайп вниз сворачивает/закрывает
// окно и перехватывает drag шашек. setupTelegram разворачивает на всю высоту и
// отключает вертикальный свайп. Вне Telegram (platform 'unknown') — no-op.

import { describe, expect, test } from 'vitest'

import { setupTelegram, type TelegramWebApp } from './telegram'

describe('setupTelegram', () => {
  test('setupTelegram_insideTelegram_callsReadyExpandDisableSwipes', () => {
    const calls: string[] = []
    const webApp: TelegramWebApp = {
      platform: 'ios',
      ready: () => calls.push('ready'),
      expand: () => calls.push('expand'),
      disableVerticalSwipes: () => calls.push('disableVerticalSwipes'),
    }

    setupTelegram(webApp)

    expect(calls).toEqual(['ready', 'expand', 'disableVerticalSwipes'])
  })

  test('setupTelegram_outsideTelegram_platformUnknown_isNoop', () => {
    let touched = false
    setupTelegram({ platform: 'unknown', expand: () => (touched = true) })
    expect(touched).toBe(false)
  })

  test('setupTelegram_noWebApp_doesNotThrow', () => {
    expect(() => setupTelegram(undefined)).not.toThrow()
  })

  test('setupTelegram_oldClientWithoutDisableSwipes_stillExpands', () => {
    // Клиент старее Bot API 7.7 — метода disableVerticalSwipes нет, не падаем.
    const calls: string[] = []
    const webApp: TelegramWebApp = {
      platform: 'android',
      ready: () => calls.push('ready'),
      expand: () => calls.push('expand'),
    }

    setupTelegram(webApp)

    expect(calls).toEqual(['ready', 'expand'])
  })
})
