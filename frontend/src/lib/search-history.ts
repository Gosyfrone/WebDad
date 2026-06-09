'use client'

import type { RelationUser } from '@/types'

const MAX_SEARCH_HISTORY = 8
const SEARCH_HISTORY_PREFIX = 'breezy-search-history'

export interface SearchHistoryEntry extends RelationUser {
  visitedAt: string
}

function storageKey(userId: string | null | undefined): string {
  return `${SEARCH_HISTORY_PREFIX}:${userId || 'local'}`
}

function isSearchHistoryEntry(value: unknown): value is SearchHistoryEntry {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<SearchHistoryEntry>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.username === 'string' &&
    typeof candidate.displayName === 'string' &&
    typeof candidate.visitedAt === 'string'
  )
}

export function readSearchHistory(userId: string | null | undefined): SearchHistoryEntry[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = window.localStorage.getItem(storageKey(userId))
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed.filter(isSearchHistoryEntry).slice(0, MAX_SEARCH_HISTORY)
  } catch {
    return []
  }
}

export function addSearchHistoryEntry(
  userId: string | null | undefined,
  user: RelationUser,
): SearchHistoryEntry[] {
  const next: SearchHistoryEntry[] = [
    { ...user, visitedAt: new Date().toISOString() },
    ...readSearchHistory(userId).filter((entry) => entry.id !== user.id),
  ].slice(0, MAX_SEARCH_HISTORY)

  if (typeof window !== 'undefined') {
    try {
      window.localStorage.setItem(storageKey(userId), JSON.stringify(next))
    } catch {
      /* localStorage peut être indisponible en navigation privée stricte. */
    }
  }
  return next
}

export function clearSearchHistory(userId: string | null | undefined): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.removeItem(storageKey(userId))
  } catch {
    /* best-effort */
  }
}

export function removeSearchHistoryEntry(
  userId: string | null | undefined,
  entryId: string,
): SearchHistoryEntry[] {
  const next = readSearchHistory(userId).filter((entry) => entry.id !== entryId)
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
