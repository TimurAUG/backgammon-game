// FRONTEND_SPEC #26a — стор connection: состояние WS-сокета
// (idle|connecting|connected|reconnecting|error); setConnectionState
// мутирует, resetConnectionState возвращает к idle.

import { beforeEach, describe, expect, test } from 'vitest'

import { connection, resetConnectionState, setConnectionState } from './connection.svelte'

beforeEach(() => {
  resetConnectionState()
})

describe('connection store (#26a)', () => {
  test('connection_default_isIdle', () => {
    expect(connection.state).toBe('idle')
  })

  test('connection_setReconnecting_updatesState', () => {
    setConnectionState('reconnecting')

    expect(connection.state).toBe('reconnecting')
  })

  test('connection_reset_returnsToIdle', () => {
    setConnectionState('connected')

    resetConnectionState()

    expect(connection.state).toBe('idle')
  })
})
