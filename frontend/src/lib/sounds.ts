'use client'

export type AppSound = 'breeze_posted' | 'notification_received' | 'message_received'

type AudioWindow = Window & typeof globalThis & { webkitAudioContext?: typeof AudioContext }

interface ToneStep {
  frequency: number
  start: number
  duration: number
  type?: OscillatorType
  gain?: number
}

const COOLDOWN_MS = 450
const lastPlayed = new Map<AppSound, number>()
let audioCtx: AudioContext | null = null
const audioElements = new Map<AppSound, HTMLAudioElement>()
let unlockInstalled = false

const soundFiles: Partial<Record<AppSound, string>> = {
  breeze_posted: '/sounds/breeze_posted.mp3',
  notification_received: '/sounds/message_received.mp3',
  message_received: '/sounds/message_received.mp3',
}

const patterns: Record<AppSound, ToneStep[]> = {
  breeze_posted: [
    { frequency: 520, start: 0, duration: 0.055, type: 'sine', gain: 0.04 },
    { frequency: 780, start: 0.06, duration: 0.075, type: 'sine', gain: 0.045 },
    { frequency: 1040, start: 0.135, duration: 0.085, type: 'triangle', gain: 0.035 },
  ],
  notification_received: [
    { frequency: 880, start: 0, duration: 0.06, type: 'triangle', gain: 0.04 },
    { frequency: 660, start: 0.075, duration: 0.09, type: 'triangle', gain: 0.035 },
  ],
  message_received: [
    { frequency: 430, start: 0, duration: 0.055, type: 'sine', gain: 0.038 },
    { frequency: 575, start: 0.065, duration: 0.08, type: 'sine', gain: 0.04 },
  ],
}

function getAudioContext(): AudioContext | null {
  if (typeof window === 'undefined') return null
  if (audioCtx) return audioCtx

  const AudioContextCtor = window.AudioContext ?? (window as AudioWindow).webkitAudioContext
  if (!AudioContextCtor) return null

  try {
    audioCtx = new AudioContextCtor()
    return audioCtx
  } catch {
    return null
  }
}

function installUnlockListeners(): void {
  if (unlockInstalled || typeof window === 'undefined') return
  unlockInstalled = true

  const unlock = () => {
    const ctx = getAudioContext()
    if (ctx?.state === 'suspended') void ctx.resume().catch(() => {})
    Object.keys(soundFiles).forEach((sound) => getAudioElement(sound as AppSound)?.load())
    if (ctx?.state === 'running') {
      window.removeEventListener('pointerdown', unlock)
      window.removeEventListener('keydown', unlock)
    }
  }

  window.addEventListener('pointerdown', unlock, { passive: true })
  window.addEventListener('keydown', unlock)
}

function getAudioElement(sound: AppSound): HTMLAudioElement | null {
  if (typeof window === 'undefined') return null
  const src = soundFiles[sound]
  if (!src) return null

  const cached = audioElements.get(sound)
  if (cached) return cached

  const audio = new Audio(src)
  audio.preload = 'auto'
  audioElements.set(sound, audio)
  return audio
}

function playGeneratedSound(sound: AppSound): void {
  const ctx = getAudioContext()
  if (!ctx) return

  if (ctx.state === 'suspended') {
    void ctx.resume().catch(() => {})
  }

  const baseTime = ctx.currentTime + 0.015
  patterns[sound].forEach((step) => playTone(ctx, step, baseTime))
}

function playAudioFile(sound: AppSound): boolean {
  const audio = getAudioElement(sound)
  if (!audio) return false

  try {
    audio.pause()
    audio.currentTime = 0
    void audio.play().catch(() => playGeneratedSound(sound))
    return true
  } catch {
    return false
  }
}

function playTone(ctx: AudioContext, step: ToneStep, baseTime: number): void {
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()
  const startAt = baseTime + step.start
  const endAt = startAt + step.duration

  osc.type = step.type ?? 'sine'
  osc.frequency.setValueAtTime(step.frequency, startAt)
  gain.gain.setValueAtTime(0.0001, startAt)
  gain.gain.exponentialRampToValueAtTime(step.gain ?? 0.04, startAt + 0.012)
  gain.gain.exponentialRampToValueAtTime(0.0001, endAt)

  osc.connect(gain)
  gain.connect(ctx.destination)
  osc.start(startAt)
  osc.stop(endAt + 0.02)
}

export function playAppSound(sound: AppSound): void {
  installUnlockListeners()

  const now = Date.now()
  if (now - (lastPlayed.get(sound) ?? 0) < COOLDOWN_MS) return
  lastPlayed.set(sound, now)

  if (playAudioFile(sound)) return

  playGeneratedSound(sound)
}

installUnlockListeners()
