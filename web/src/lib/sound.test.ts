// FRONTEND_SPEC #32 — playRollCue: короткий сигнал через Web Audio.
// Конструктор AudioContext инжектируется; без поддержки и при ошибке — no-op,
// чтобы отсутствие аудио (jsdom, политика автоплея) не валило приложение.

import { describe, expect, test, vi } from 'vitest'

import { playRollCue, type AudioContextCtor } from './sound'

function mockAudio() {
  const osc = {
    type: '',
    frequency: { value: 0 },
    connect: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
  }
  const gain = {
    gain: { value: 0, setValueAtTime: vi.fn(), exponentialRampToValueAtTime: vi.fn() },
    connect: vi.fn(),
  }
  const ctx = {
    currentTime: 0,
    destination: { id: 'destination' },
    createOscillator: () => osc,
    createGain: () => gain,
  }
  const Ctor = vi.fn(() => ctx) as unknown as AudioContextCtor
  return { Ctor, ctx, osc, gain }
}

describe('playRollCue (#32)', () => {
  test('playRollCue_withContext_startsOscillatorWiredToDestination', () => {
    const { Ctor, ctx, osc, gain } = mockAudio()

    playRollCue(Ctor)

    expect(osc.start).toHaveBeenCalledOnce()
    expect(osc.connect).toHaveBeenCalledWith(gain)
    expect(gain.connect).toHaveBeenCalledWith(ctx.destination)
  })

  test('playRollCue_noAudioContext_isNoOpAndDoesNotThrow', () => {
    expect(() => playRollCue(null)).not.toThrow()
  })

  test('playRollCue_contextConstructorThrows_swallowsError', () => {
    const Ctor = vi.fn(() => {
      throw new Error('AudioContext unavailable')
    }) as unknown as AudioContextCtor

    expect(() => playRollCue(Ctor)).not.toThrow()
  })
})
