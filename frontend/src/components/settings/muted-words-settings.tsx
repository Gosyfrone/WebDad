'use client'

import { FormEvent, useEffect, useMemo, useState } from 'react'
import { Plus, X } from 'lucide-react'

import {
  readMutedWords,
  saveMutedWords,
  subscribeMutedWords,
} from '@/lib/content-filters'
import { getMe } from '@/lib/api'
import { currentUserId } from '@/lib/posts'
import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const MAX_WORDS = 40

export function MutedWordsSettings() {
  const t = useT()
  const [words, setWords] = useState<string[]>([])
  const [draft, setDraft] = useState('')
  const [viewerUserId, setViewerUserId] = useState('')
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let cancelled = false
    let unsubscribe = () => {}

    async function loadUserScopedWords() {
      let userId = currentUserId()
      if (!userId) {
        try {
          userId = (await getMe()).id
        } catch {
          userId = ''
        }
      }

      if (cancelled) return
      setViewerUserId(userId)
      setWords(readMutedWords(userId))
      setReady(true)
      unsubscribe = subscribeMutedWords(userId, setWords)
    }

    void loadUserScopedWords()
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  const normalizedWords = useMemo(
    () => new Set(words.map((word) => word.trim().toLowerCase())),
    [words],
  )

  function addWord(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const value = draft.trim().replace(/\s+/g, ' ')
    if (!value || normalizedWords.has(value.toLowerCase()) || words.length >= MAX_WORDS) return
    setWords(saveMutedWords([...words, value], viewerUserId))
    setDraft('')
  }

  function removeWord(word: string) {
    setWords(saveMutedWords(words.filter((item) => item !== word), viewerUserId))
  }

  return (
    <div className="panel rounded-xl border px-3 py-3 shadow-sm">
      <form onSubmit={addWord} className="flex gap-2">
        <Input
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          placeholder={t('filters.placeholder')}
          aria-label={t('filters.input_aria')}
          maxLength={80}
        />
        <Button
          type="submit"
          size="icon"
          disabled={!ready || !draft.trim() || words.length >= MAX_WORDS}
          aria-label={t('filters.add')}
          className="shrink-0"
        >
          <Plus className="h-4 w-4" aria-hidden />
        </Button>
      </form>

      {words.length > 0 ? (
        <div className="mt-3 flex max-h-[5.75rem] flex-wrap gap-2 overflow-y-auto pr-1">
          {words.map((word) => (
            <span
              key={word}
              className="inline-flex max-w-full items-center gap-1 rounded-full border bg-background/65 px-2.5 py-1 text-xs font-semibold"
            >
              <span className="truncate">{word}</span>
              <button
                type="button"
                onClick={() => removeWord(word)}
                className="rounded-full p-0.5 text-muted-foreground transition hover:bg-accent hover:text-foreground"
                aria-label={t('filters.remove_aria', { word })}
              >
                <X className="h-3.5 w-3.5" aria-hidden />
              </button>
            </span>
          ))}
        </div>
      ) : (
        <p className="mt-3 text-sm text-muted-foreground">{t('filters.empty')}</p>
      )}
    </div>
  )
}
