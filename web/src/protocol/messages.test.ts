// FRONTEND_SPEC #1 — ClientMessage сериализация.
//
// Контракт — nardy-protocol § «Клиент → сервер». Тесты фиксируют
// именно JSON-строку (с порядком ключей, который даёт JSON.stringify
// от литерала), чтобы поймать любое расхождение по имени поля или
// нелишнему `optional`-полю.

import { describe, expect, test } from 'vitest'

import { serializeClientMessage, type ClientMessage } from './messages'

describe('serializeClientMessage', () => {
  test.each<[string, ClientMessage, string]>([
    ['JOIN', { type: 'JOIN', gameId: 'g1', token: 'tok' }, '{"type":"JOIN","gameId":"g1","token":"tok"}'],
    ['ROLL_FOR_FIRST', { type: 'ROLL_FOR_FIRST' }, '{"type":"ROLL_FOR_FIRST"}'],
    ['ROLL', { type: 'ROLL' }, '{"type":"ROLL"}'],
    ['MOVE', { type: 'MOVE', from: 24, to: 18 }, '{"type":"MOVE","from":24,"to":18}'],
    ['MOVE bear off (to=0)', { type: 'MOVE', from: 1, to: 0 }, '{"type":"MOVE","from":1,"to":0}'],
    ['END_TURN', { type: 'END_TURN' }, '{"type":"END_TURN"}'],
    ['RESIGN', { type: 'RESIGN' }, '{"type":"RESIGN"}'],
  ])('serializeClientMessage_%s_matchesProtocol', (_label, msg, expected) => {
    expect(serializeClientMessage(msg)).toBe(expected)
  })
})
