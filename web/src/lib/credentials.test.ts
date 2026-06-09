// FRONTEND_SPEC #23 — персистентность gameId+token в localStorage.
//
// Отдельный модуль lib/credentials: его переиспользуют Connect (save),
// App (load для авто-подключения, web#24) и обработка UNAUTHORIZED
// (clear, web#25).

import { beforeEach, describe, expect, test } from 'vitest'

import { clearCredentials, loadCredentials, saveCredentials } from './credentials'

beforeEach(() => {
  localStorage.clear()
})

describe('credentials persistence (#23)', () => {
  test('credentials_roundtrip_returnsSaved', () => {
    saveCredentials({ gameId: 'g-42', token: 'tok-abc' })
    expect(loadCredentials()).toEqual({ gameId: 'g-42', token: 'tok-abc' })
  })

  test('credentials_loadWithoutSaved_returnsNull', () => {
    expect(loadCredentials()).toBeNull()
  })

  test('credentials_loadCorruptValue_returnsNull', () => {
    localStorage.setItem('nardy.credentials', 'не-json')
    expect(loadCredentials()).toBeNull()
  })

  test('credentials_clear_removesSaved', () => {
    saveCredentials({ gameId: 'g-42', token: 'tok-abc' })
    clearCredentials()
    expect(loadCredentials()).toBeNull()
  })
})
