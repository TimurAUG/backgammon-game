// FRONTEND_SPEC #33 — Toast: одна плашка уведомления. Доступна как role=status
// (aria-live), закрывается кнопкой и сама исчезает по таймеру (ttl).

import { fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, test, vi } from 'vitest'

import Toast from './Toast.svelte'

afterEach(() => {
  vi.useRealTimers()
})

describe('Toast (#33)', () => {
  test('Toast_rendersTextAsStatusRegion', () => {
    render(Toast, { props: { text: 'Твой бросок', onClose: vi.fn() } })

    expect(screen.getByRole('status')).toHaveTextContent('Твой бросок')
  })

  test('Toast_closeButtonClick_callsOnClose', async () => {
    const onClose = vi.fn()
    render(Toast, { props: { text: 'Соперник присоединился', onClose } })

    await fireEvent.click(screen.getByRole('button', { name: 'Закрыть' }))

    expect(onClose).toHaveBeenCalledOnce()
  })

  test('Toast_afterTtlElapses_callsOnClose', () => {
    vi.useFakeTimers()
    const onClose = vi.fn()
    render(Toast, { props: { text: 'Твой бросок', onClose, ttl: 5000 } })

    expect(onClose).not.toHaveBeenCalled()
    vi.advanceTimersByTime(5000)

    expect(onClose).toHaveBeenCalledOnce()
  })
})
