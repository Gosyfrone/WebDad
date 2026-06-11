'use client'

const MAX_SUGGESTION_HISTORY = 8
const SUGGESTION_HISTORY_PREFIX = 'breezy-suggestion-history'

export type SuggestionHistoryKind = 'hashtag' | 'profile'

export interface SuggestionHistoryEntry {
  id: string
  kind: SuggestionHistoryKind
  label: string
  subtitle: string
  href: string
  avatarUrl?: string
  visitedAt: string
}

function storageKey(userId: string | null | undefined): string {
  return `${SUGGESTION_HISTORY_PREFIX}:${userId || 'local'}`
}

function isHistoryEntry(value: unknown): value is SuggestionHistoryEntry {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<SuggestionHistoryEntry>
  return (
    typeof candidate.id === 'string' &&
    (candidate.kind === 'hashtag' || candidate.kind === 'profile') &&
    typeof candidate.label === 'string' &&
    typeof candidate.subtitle === 'string' &&
    typeof candidate.href === 'string' &&
    typeof candidate.visitedAt === 'string'
  )
}

export function readSuggestionHistory(userId: string | null | undefined): SuggestionHistoryEntry[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = window.localStorage.getItem(storageKey(userId))
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed.filter(isHistoryEntry).slice(0, MAX_SUGGESTION_HISTORY)
  } catch {
    return []
  }
}

export function addSuggestionHistoryEntry(
  userId: string | null | undefined,
  entry: Omit<SuggestionHistoryEntry, 'visitedAt'>,
): SuggestionHistoryEntry[] {
  const next: SuggestionHistoryEntry[] = [
    { ...entry, visitedAt: new Date().toISOString() },
    ...readSuggestionHistory(userId).filter((item) => item.id !== entry.id),
  ].slice(0, MAX_SUGGESTION_HISTORY)

  if (typeof window !== 'undefined') {
    try {
      window.localStorage.setItem(storageKey(userId), JSON.stringify(next))
    } catch {
      /* best-effort */
    }
  }
  return next
}

export function clearSuggestionHistory(userId: string | null | undefined): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.removeItem(storageKey(userId))
  } catch {
    /* best-effort */
  }
}

export function removeSuggestionHistoryEntry(
  userId: string | null | undefined,
  entryId: string,
): SuggestionHistoryEntry[] {
  const next = readSuggestionHistory(userId).filter((entry) => entry.id !== entryId)
  if (typeof window !== 'undefined') {
    try {
      if (next.length > 0) {
        window.localStorage.setItem(storageKey(userId), JSON.stringify(next))
      } else {
        window.localStorage.removeItem(storageKey(userId))
      }
    } catch {
      /* best-effort */
    }
  }
  return next
}
