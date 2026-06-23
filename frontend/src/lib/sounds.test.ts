import { afterEach, describe, expect, it, vi } from 'vitest'

// `sounds` pilote la Web Audio API + des <audio>. On installe des fakes AVANT
// l'import (le module pose des écouteurs au chargement) et on réimporte à neuf
// par test (état module : contexte audio, cooldown, écouteurs).

interface FakeCtx {
  state: string
  currentTime: number
  destination: unknown
  resume: ReturnType<typeof vi.fn>
  createOscillator: ReturnType<typeof vi.fn>
  createGain: ReturnType<typeof vi.fn>
}

let listeners: Map<string, (e?: unknown) => void>
let ctx: FakeCtx
let audioPlay: ReturnType<typeof vi.fn>
let audioThrows: boolean

function osc() {
  return { type: '', frequency: { setValueAtTime: vi.fn() }, connect: vi.fn(), start: vi.fn(), stop: vi.fn() }
}
function gain() {
  return { gain: { setValueAtTime: vi.fn(), exponentialRampToValueAtTime: vi.fn() }, connect: vi.fn() }
}

function setup(opts: { withAudioContext?: boolean; state?: string; playRejects?: boolean; pauseThrows?: boolean } = {}) {
  const { withAudioContext = true, state = 'running', playRejects = false, pauseThrows = false } = opts
  listeners = new Map()
  audioThrows = pauseThrows
  ctx = {
    state,
    currentTime: 1,
    destination: {},
    resume: vi.fn(async () => { ctx.state = 'running' }),
    createOscillator: vi.fn(osc),
    createGain: vi.fn(gain),
  }
  audioPlay = vi.fn(() => (playRejects ? Promise.reject(new Error('blocked')) : Promise.resolve()))
  const AudioCtor = vi.fn(() => ctx)
  const win: Record<string, unknown> = {
    addEventListener: (t: string, h: (e?: unknown) => void) => listeners.set(t, h),
    removeEventListener: (t: string) => listeners.delete(t),
  }
  if (withAudioContext) win.AudioContext = AudioCtor
  vi.stubGlobal('window', win)
  vi.stubGlobal('Audio', class {
    src: string; preload = ''; currentTime = 0
    constructor(src: string) { this.src = src }
    pause() { if (audioThrows) throw new Error('nope') }
    play() { return audioPlay() }
    load() {}
  })
}

async function load() {
  vi.resetModules()
  return import('@/lib/sounds')
}
afterEach(() => vi.unstubAllGlobals())

describe('playAppSound', () => {
  it('joue le fichier audio quand il est disponible', async () => {
    setup()
    const { playAppSound } = await load()
    playAppSound('breeze_posted')
    expect(audioPlay).toHaveBeenCalled()
  })

  it('cooldown : un 2e appel immédiat est ignoré', async () => {
    setup()
    const { playAppSound } = await load()
    playAppSound('message_received')
    playAppSound('message_received')
    expect(audioPlay).toHaveBeenCalledTimes(1)
  })

  it('repli sur le son généré quand la lecture du fichier échoue', async () => {
    setup({ pauseThrows: true })
    const { playAppSound } = await load()
    playAppSound('breeze_posted')
    // playAudioFile renvoie false → playGeneratedSound → oscillateurs créés
    expect(ctx.createOscillator).toHaveBeenCalled()
    expect(ctx.createGain).toHaveBeenCalled()
  })

  it('contexte suspendu → resume avant de générer', async () => {
    setup({ pauseThrows: true, state: 'suspended' })
    const { playAppSound } = await load()
    playAppSound('notification_received')
    expect(ctx.resume).toHaveBeenCalled()
  })

  it('sans Web Audio API → ne génère rien (pas de crash)', async () => {
    setup({ withAudioContext: false, pauseThrows: true })
    const { playAppSound } = await load()
    expect(() => playAppSound('breeze_posted')).not.toThrow()
    expect(ctx.createOscillator).not.toHaveBeenCalled()
  })

  it('l’écouteur de déverrouillage reprend le contexte au 1er geste', async () => {
    setup({ state: 'suspended' })
    await load()
    // un module fraîchement chargé a posé pointerdown/keydown
    listeners.get('pointerdown')?.()
    expect(ctx.resume).toHaveBeenCalled()
  })
})
