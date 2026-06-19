'use client'

import { useRef } from 'react'

interface LongPressOptions {
  delayMs?: number
  /** Désactive la détection (ex. visiteur). */
  enabled?: boolean
}

/**
 * Appui long (tactile/souris) : `onLongPress` se déclenche après `delayMs`.
 * Brancher `start` sur `onPointerDown`, `cancel` sur `onPointerUp`/`onPointerLeave`.
 * `consume()` rend `true` une fois si le dernier geste était un appui long —
 * à appeler en tête du `onClick` pour neutraliser le clic court qui suit.
 */
export function useLongPress(
  onLongPress: () => void,
  { delayMs = 500, enabled = true }: LongPressOptions = {},
) {
  const fired = useRef(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  function start() {
    if (!enabled) return
    fired.current = false
    timer.current = setTimeout(() => {
      fired.current = true
      onLongPress()
    }, delayMs)
  }

  function cancel() {
    if (timer.current) {
      clearTimeout(timer.current)
      timer.current = null
    }
  }

  function consume(): boolean {
    if (!fired.current) return false
    fired.current = false
    return true
  }

  return { start, cancel, consume }
}
