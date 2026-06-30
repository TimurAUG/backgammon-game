// FRONTEND_SPEC #1 — ClientMessage сериализация.
// FRONTEND_SPEC #2 — ServerMessage парсинг.
//
// Контракт — nardy-protocol § «Клиент → сервер» / «Сервер → клиент».
// Сериализационные тесты фиксируют точные JSON-строки. Парсерные
// тесты проверяют, что dictionary-форма парсится в narrow-тип
// и что мусор отбрасывается.

import { describe, expect, test } from 'vitest'

import {
  parseServerMessage,
  serializeClientMessage,
  type ClientMessage,
  type ErrorCode,
  type ServerMessage,
} from './messages'

describe('serializeClientMessage', () => {
  test.each<[string, ClientMessage, string]>([
    ['JOIN', { type: 'JOIN', gameId: 'g1', token: 'tok' }, '{"type":"JOIN","gameId":"g1","token":"tok"}'],
    ['ROLL_FOR_FIRST', { type: 'ROLL_FOR_FIRST' }, '{"type":"ROLL_FOR_FIRST"}'],
    ['ROLL', { type: 'ROLL' }, '{"type":"ROLL"}'],
    ['MOVE', { type: 'MOVE', from: 24, to: 18 }, '{"type":"MOVE","from":24,"to":18}'],
    ['MOVE bear off (to=0)', { type: 'MOVE', from: 1, to: 0 }, '{"type":"MOVE","from":1,"to":0}'],
    ['END_TURN', { type: 'END_TURN' }, '{"type":"END_TURN"}'],
    ['RESIGN', { type: 'RESIGN' }, '{"type":"RESIGN"}'],
    ['CHAT', { type: 'CHAT', text: 'привет' }, '{"type":"CHAT","text":"привет"}'],
  ])('serializeClientMessage_%s_matchesProtocol', (_label, msg, expected) => {
    expect(serializeClientMessage(msg)).toBe(expected)
  })
})

describe('parseServerMessage', () => {
  test('parseServerMessage_STATE_includesAllFields', () => {
    const raw = JSON.stringify({
      type: 'STATE',
      board: Array(24).fill(0),
      turn: 'white',
      status: 'waitingForRoll',
      borneOff: { white: 0, black: 0 },
      isFirstMove: { white: true, black: true },
      dice: { a: 3, b: 5, isDouble: false, remaining: [3, 5] },
    })

    const msg = parseServerMessage(raw)
    expect(msg.type).toBe('STATE')
    if (msg.type !== 'STATE') return
    expect(msg.board).toHaveLength(24)
    expect(msg.turn).toBe('white')
    expect(msg.status).toBe('waitingForRoll')
    expect(msg.borneOff).toEqual({ white: 0, black: 0 })
    expect(msg.isFirstMove).toEqual({ white: true, black: true })
    expect(msg.dice).toEqual({ a: 3, b: 5, isDouble: false, remaining: [3, 5] })
  })

  test('parseServerMessage_LEGAL_MOVES_parsesMoveArray', () => {
    const raw = JSON.stringify({
      type: 'LEGAL_MOVES',
      moves: [
        { from: 24, to: 18, pip: 6 },
        { from: 24, to: 19, pip: 5 },
      ],
    })

    const msg = parseServerMessage(raw)
    expect(msg.type).toBe('LEGAL_MOVES')
    if (msg.type !== 'LEGAL_MOVES') return
    expect(msg.moves).toHaveLength(2)
    expect(msg.moves[0]).toEqual({ from: 24, to: 18, pip: 6 })
  })

  test('parseServerMessage_LEGAL_MOVES_parsesReachChains', () => {
    const raw = JSON.stringify({
      type: 'LEGAL_MOVES',
      moves: [{ from: 13, to: 11, pip: 2 }],
      reach: [{ from: 13, path: [11, 7], pips: [2, 4] }],
    })

    const msg = parseServerMessage(raw)
    expect(msg.type).toBe('LEGAL_MOVES')
    if (msg.type !== 'LEGAL_MOVES') return
    expect(msg.reach).toEqual([{ from: 13, path: [11, 7], pips: [2, 4] }])
  })

  test('parseServerMessage_TURN_SKIPPED_keepsColorAndDice', () => {
    const raw = JSON.stringify({
      type: 'TURN_SKIPPED',
      color: 'white',
      dice: { a: 1, b: 2, isDouble: false, remaining: [1, 2] },
    })

    const msg = parseServerMessage(raw)
    expect(msg.type).toBe('TURN_SKIPPED')
    if (msg.type !== 'TURN_SKIPPED') return
    expect(msg.color).toBe('white')
    expect(msg.dice.a).toBe(1)
    expect(msg.dice.b).toBe(2)
  })

  test('parseServerMessage_OPPONENT_JOINED_keepsName', () => {
    const msg = parseServerMessage('{"type":"OPPONENT_JOINED","name":"Тимур"}')
    expect(msg.type).toBe('OPPONENT_JOINED')
    if (msg.type !== 'OPPONENT_JOINED') return
    expect(msg.name).toBe('Тимур')
  })

  test('parseServerMessage_OPPONENT_LEFT_parsesBare', () => {
    const msg = parseServerMessage('{"type":"OPPONENT_LEFT"}')
    expect(msg.type).toBe('OPPONENT_LEFT')
  })

  test('parseServerMessage_GAME_OVER_keepsWinnerAndKind', () => {
    const msg = parseServerMessage('{"type":"GAME_OVER","winner":"black","kind":"koks"}')
    expect(msg.type).toBe('GAME_OVER')
    if (msg.type !== 'GAME_OVER') return
    expect(msg.winner).toBe('black')
    expect(msg.kind).toBe('koks')
  })

  test('parseServerMessage_CHAT_keepsSenderAndText', () => {
    const msg = parseServerMessage('{"type":"CHAT","sender":"white","text":"гг"}')
    expect(msg.type).toBe('CHAT')
    if (msg.type !== 'CHAT') return
    expect(msg.sender).toBe('white')
    expect(msg.text).toBe('гг')
  })

  test('parseServerMessage_CHAT_HISTORY_parsesArray', () => {
    const raw = JSON.stringify({
      type: 'CHAT_HISTORY',
      chat: [
        { sender: 'white', text: 'раз' },
        { sender: 'black', text: 'два' },
      ],
    })
    const msg = parseServerMessage(raw)
    expect(msg.type).toBe('CHAT_HISTORY')
    if (msg.type !== 'CHAT_HISTORY') return
    expect(msg.chat).toHaveLength(2)
    expect(msg.chat[0]).toEqual({ sender: 'white', text: 'раз' })
    expect(msg.chat[1]).toEqual({ sender: 'black', text: 'два' })
  })

  test('parseServerMessage_ERROR_keepsCodeAndMessage', () => {
    const msg = parseServerMessage(
      '{"type":"ERROR","code":"RULE_OF_SIX","message":"финальный блок 6+ при пустом доме соперника"}',
    )
    expect(msg.type).toBe('ERROR')
    if (msg.type !== 'ERROR') return
    expect(msg.code).toBe('RULE_OF_SIX')
    expect(msg.message).toContain('блок 6+')
  })

  test('parseServerMessage_JOINED_keepsColor', () => {
    // web#24-prep — JOINED сообщает клиенту его цвет (nardy-protocol).
    const msg = parseServerMessage('{"type":"JOINED","color":"black"}')
    expect(msg.type).toBe('JOINED')
    if (msg.type !== 'JOINED') return
    expect(msg.color).toBe('black')
  })

  test('parseServerMessage_ERROR_acceptsRoomFullCode', () => {
    // web#24-prep — ROOM_FULL шлётся третьему клиенту (FRONTEND_SPEC § 9.2).
    const msg = parseServerMessage('{"type":"ERROR","code":"ROOM_FULL","message":"оба слота заняты"}')
    expect(msg.type).toBe('ERROR')
    if (msg.type !== 'ERROR') return
    const code: ErrorCode = msg.code
    expect(code).toBe('ROOM_FULL')
  })

  test('parseServerMessage_invalidJson_throws', () => {
    expect(() => parseServerMessage('not-json')).toThrow()
  })

  test('parseServerMessage_missingType_throws', () => {
    expect(() => parseServerMessage('{"board":[]}')).toThrow()
  })

  test('parseServerMessage_typeNotString_throws', () => {
    expect(() => parseServerMessage('{"type":42}')).toThrow()
  })

  // type-narrowing — это compile-time проверка, не runtime;
  // тут только гарантируем, что сам тип ServerMessage экспортирован.
  test('ServerMessage_typeIsExported', () => {
    const placeholder: ServerMessage = { type: 'OPPONENT_LEFT' }
    expect(placeholder.type).toBe('OPPONENT_LEFT')
  })
})
