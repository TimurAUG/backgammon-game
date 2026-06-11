// sound — короткие звуковые сигналы UI через Web Audio. Без бинарных ассетов:
// тон синтезируется на лету. Конструктор AudioContext инжектируется (DI) —
// в проде берётся из window, в тестах подменяется моком.
//
// Любой сбой (нет Web Audio в jsdom, политика автоплея до жеста пользователя,
// нет аудиоустройства) — тихий no-op: звук вторичен, ронять UI из-за него нельзя.

interface OscillatorLike {
  type: string
  frequency: { value: number }
  connect(dest: unknown): void
  start(when?: number): void
  stop(when?: number): void
}

interface GainLike {
  gain: {
    value: number
    setValueAtTime(value: number, startTime: number): void
    exponentialRampToValueAtTime(value: number, endTime: number): void
  }
  connect(dest: unknown): void
}

interface AudioContextLike {
  currentTime: number
  destination: unknown
  createOscillator(): OscillatorLike
  createGain(): GainLike
}

export type AudioContextCtor = new () => AudioContextLike

function resolveAudioContext(): AudioContextCtor | null {
  const w = globalThis as { AudioContext?: AudioContextCtor; webkitAudioContext?: AudioContextCtor }
  return w.AudioContext ?? w.webkitAudioContext ?? null
}

// Короткий «дзынь» (~150мс) с экспоненциальным затуханием, чтобы не было щелчка.
export function playRollCue(Ctor: AudioContextCtor | null = resolveAudioContext()): void {
  if (Ctor === null) return
  try {
    const ctx = new Ctor()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    osc.type = 'sine'
    osc.frequency.value = 880

    const now = ctx.currentTime
    gain.gain.setValueAtTime(0.2, now)
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.15)

    osc.connect(gain)
    gain.connect(ctx.destination)

    osc.start(now)
    osc.stop(now + 0.15)
  } catch {
    // Аудио недоступно — вторичный эффект, молча игнорируем.
  }
}
