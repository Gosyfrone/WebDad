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
  applyMention,
  detectMentionTyping,
  type MentionCandidate,
  type MentionTypingContext,
} from '@/lib/mentions'

interface UseMentionOptions {
  /** Ref vers le champ contrôlé (textarea ou input). */
  inputRef: RefObject<HTMLTextAreaElement | HTMLInputElement | null>
  /** Applique la nouvelle valeur (le composant est la source de vérité). */
  onChange: (value: string) => void
  /** Recherche des candidats pour un token (`@query`). */
  search: (query: string) => Promise<MentionCandidate[]>
}

/** Contrôleur retourné au composant : état d'affichage + handlers à brancher. */
export interface MentionController {
  open: boolean
  candidates: MentionCandidate[]
  activeIndex: number
  loading: boolean
  /** À appeler après une modification de valeur / déplacement du curseur. */
  sync: () => void
  /** À brancher sur `onKeyDown` du champ (navigation clavier de la liste). */
  onKeyDown: (e: KeyboardEvent<HTMLTextAreaElement | HTMLInputElement>) => void
  /** Survol/sélection d'un candidat (insère et referme). */
  select: (candidate: MentionCandidate) => void
  setActiveIndex: (i: number) => void
  close: () => void
}

const DEBOUNCE_MS = 180

/**
 * Autocomplétion de mentions @handle pour un champ contrôlé. Le composant garde
 * la maîtrise de sa valeur ; ce hook détecte le token sous le curseur (via
 * `sync()` appelé sur onChange/onClick/onKeyUp), recherche les candidats
 * (debounce), gère la navigation clavier et insère `@username ` à la sélection.
 *
 * La pop-up elle-même (positionnement/rendu) est laissée au composant
 * (`MentionAutocomplete`) : ce hook ne gère que l'état et la logique.
 */
export function useMention({ inputRef, onChange, search }: UseMentionOptions): MentionController {
  const [open, setOpen] = useState(false)
  const [candidates, setCandidates] = useState<MentionCandidate[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const [loading, setLoading] = useState(false)

  const ctxRef = useRef<MentionTypingContext | null>(null)
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
            if (reqId !== reqIdRef.current) return // résultat périmé
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
    const ctx = detectMentionTyping(el.value, caret)
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
    (candidate: MentionCandidate) => {
      const el = inputRef.current
      const ctx = ctxRef.current
      if (!el || !ctx) return
      const { value, caret } = applyMention(el.value, ctx, candidate.username)
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
      if (!open || candidates.length === 0) return
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setActiveIndex((i) => (i + 1) % candidates.length)
          break
        case 'ArrowUp':
          e.preventDefault()
          setActiveIndex((i) => (i - 1 + candidates.length) % candidates.length)
          break
        case 'Enter':
        case 'Tab': {
          e.preventDefault()
          const chosen = candidates[activeIndex]
          if (chosen) select(chosen)
          break
        }
        case 'Escape':
          e.preventDefault()
          close()
          break
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
