'use client'

import type { FeedPost } from '@/lib/posts'

export const MUTED_WORDS_STORAGE_KEY = 'breezy-muted-words'
const MUTED_WORDS_EVENT = 'breezy:muted-words-updated'
const ANONYMOUS_USER_KEY = 'anonymous'

function storageKeyForUser(userId: string): string {
  return `${MUTED_WORDS_STORAGE_KEY}:${userId || ANONYMOUS_USER_KEY}`
}

function normalizeText(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
}

function normalizeWord(value: string): string {
  return normalizeText(value).trim().replace(/\s+/g, ' ')
}

export function sanitizeMutedWords(words: string[]): string[] {
  const seen = new Set<string>()
  const sanitized: string[] = []

  for (const word of words) {
    const normalized = normalizeWord(word)
    if (!normalized || seen.has(normalized)) continue
    seen.add(normalized)
    sanitized.push(word.trim().replace(/\s+/g, ' '))
  }

  return sanitized
}

export function readMutedWords(userId = ''): string[] {
  if (typeof window === 'undefined') return []

  try {
    const raw = window.localStorage.getItem(storageKeyForUser(userId))
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed)
      ? sanitizeMutedWords(parsed.filter((item): item is string => typeof item === 'string'))
      : []
  } catch {
    return []
  }
}

export function saveMutedWords(words: string[], userId = ''): string[] {
  const next = sanitizeMutedWords(words)
  if (typeof window === 'undefined') return next

  window.localStorage.setItem(storageKeyForUser(userId), JSON.stringify(next))
  window.dispatchEvent(
    new CustomEvent<{ userId: string; words: string[] }>(MUTED_WORDS_EVENT, {
      detail: { userId, words: next },
    }),
  )
  return next
}

export function subscribeMutedWords(
  userId: string,
  onChange: (words: string[]) => void,
): () => void {
  if (typeof window === 'undefined') return () => {}

  function handleLocal(event: Event) {
    const detail = (event as CustomEvent<{ userId: string; words: string[] }>).detail
    if (detail?.userId === userId) onChange(detail.words)
  }

  function handleStorage(event: StorageEvent) {
    if (event.key === storageKeyForUser(userId)) onChange(readMutedWords(userId))
  }

  window.addEventListener(MUTED_WORDS_EVENT, handleLocal)
  window.addEventListener('storage', handleStorage)

  return () => {
    window.removeEventListener(MUTED_WORDS_EVENT, handleLocal)
    window.removeEventListener('storage', handleStorage)
  }
}

export function postMatchesMutedWords(
  post: FeedPost,
  words: string[],
  viewerUserId = '',
): boolean {
  if (words.length === 0) return false
  if (viewerUserId && post.author.id === viewerUserId) return false

  const haystack = normalizeText(
    [post.content, post.quotedPost?.content ?? ''].filter(Boolean).join(' '),
  )

  return words.some((word) => haystack.includes(normalizeWord(word)))
}

export function filterMutedPosts(
  posts: FeedPost[],
  words: string[],
  viewerUserId = '',
): FeedPost[] {
  if (words.length === 0) return posts
  return posts.filter((post) => !postMatchesMutedWords(post, words, viewerUserId))
}
