'use client'

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type RefObject,
} from 'react'

import {
  applyHashtag,
  detectHashtagTyping,
  type HashtagCandidate,
  type HashtagTypingContext,
} from '@/lib/hashtags'

interface UseHashtagOptions {
  inputRef: RefObject<HTMLTextAreaElement | HTMLInputElement | null>
  onChange: (value: string) => void
  search: (query: string) => Promise<HashtagCandidate[]>
}

export interface HashtagController {
  open: boolean
  candidates: HashtagCandidate[]
  activeIndex: number
  loading: boolean
  sync: () => void
  onKeyDown: (e: KeyboardEvent<HTMLTextAreaElement | HTMLInputElement>) => boolean
  select: (candidate: HashtagCandidate) => void
  setActiveIndex: (i: number) => void
  close: () => void
}

const DEBOUNCE_MS = 180

export function useHashtag({ inputRef, onChange, search }: UseHashtagOptions): HashtagController {
  const [open, setOpen] = useState(false)
  const [candidates, setCandidates] = useState<HashtagCandidate[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const [loading, setLoading] = useState(false)

  const ctxRef = useRef<HashtagTypingContext | null>(null)
  const queryRef = useRef<string>('')
  const reqIdRef = useRef(0)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const close = useCallback(() => {
    setOpen(false)
    setCandidates([])
    setActiveIndex(0)
    ctxRef.current = null
    queryRef.current = ''
    if (debounceRef.current) clearTimeout(debounceRef.current)
  }, [])

  const runSearch = useCallback(
    (query: string) => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      const reqId = ++reqIdRef.current
      setLoading(true)
      debounceRef.current = setTimeout(() => {
        search(query)
          .then((results) => {
            if (reqId !== reqIdRef.current) return
            setCandidates(results)
            setActiveIndex(0)
            setOpen(results.length > 0)
          })
          .catch(() => {
            if (reqId !== reqIdRef.current) return
            setCandidates([])
            setOpen(false)
          })
          .finally(() => {
            if (reqId === reqIdRef.current) setLoading(false)
          })
      }, DEBOUNCE_MS)
    },
    [search],
  )

  const sync = useCallback(() => {
    const el = inputRef.current
    if (!el) return
    const caret = el.selectionStart ?? el.value.length
    const ctx = detectHashtagTyping(el.value, caret)
    if (!ctx) {
      close()
      return
    }
    ctxRef.current = ctx
    if (ctx.query === queryRef.current && open) return
    queryRef.current = ctx.query
    runSearch(ctx.query)
  }, [inputRef, open, close, runSearch])

  const select = useCallback(
    (candidate: HashtagCandidate) => {
      const el = inputRef.current
      const ctx = ctxRef.current
      if (!el || !ctx) return
      const { value, caret } = applyHashtag(el.value, ctx, candidate.tag)
      onChange(value)
      close()
      requestAnimationFrame(() => {
        el.focus()
        el.setSelectionRange(caret, caret)
      })
    },
    [inputRef, onChange, close],
  )

  const onKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement | HTMLInputElement>) => {
      if (!open || candidates.length === 0) return false
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setActiveIndex((i) => (i + 1) % candidates.length)
          return true
        case 'ArrowUp':
          e.preventDefault()
          setActiveIndex((i) => (i - 1 + candidates.length) % candidates.length)
          return true
        case 'Enter':
        case 'Tab': {
          e.preventDefault()
          const chosen = candidates[activeIndex]
          if (chosen) select(chosen)
          return true
        }
        case 'Escape':
          e.preventDefault()
          close()
          return true
        default:
          return false
      }
    },
    [open, candidates, activeIndex, select, close],
  )

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [])

  return { open, candidates, activeIndex, loading, sync, onKeyDown, select, setActiveIndex, close }
}
