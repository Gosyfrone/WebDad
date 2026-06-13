import { describe, it, expect } from 'vitest'

import {
  buildNotification,
  notificationHref,
  toWebSocketUrl,
  type NotificationActor,
} from './notifications'

const actor: NotificationActor = {
  id: 'u1',
  username: 'zaid',
  displayName: 'Zaid',
  avatarUrl: '',
}

function raw(overrides: Record<string, unknown> = {}) {
  return {
    id: 'n1',
    type: 'like' as const,
    post_id: 'p1',
    last_actor_id: 'u1',
    count: 1,
    is_read: false,
    created_at: '2026-06-08T00:00:00Z',
    updated_at: '2026-06-08T00:00:00Z',
    ...overrides,
  }
}

describe('buildNotification (agrégation → affichage)', () => {
  it('un seul acteur → othersCount = 0', () => {
    const n = buildNotification(raw(), actor)
    expect(n.count).toBe(1)
    expect(n.othersCount).toBe(0)
    expect(n.postId).toBe('p1')
    expect(n.actor.displayName).toBe('Zaid')
  })

  it('300 likes agrégés → othersCount = 299', () => {
    const n = buildNotification(raw({ count: 300 }), actor)
    expect(n.count).toBe(300)
    expect(n.othersCount).toBe(299)
  })

  it('count incohérent (0) → ramené à 1, othersCount 0', () => {
    const n = buildNotification(raw({ count: 0 }), actor)
    expect(n.count).toBe(1)
    expect(n.othersCount).toBe(0)
  })

  it('post_id absent → postId vide', () => {
    const n = buildNotification(raw({ post_id: undefined }), actor)
    expect(n.postId).toBe('')
  })
})

describe('notificationHref', () => {
  it('pointe vers le post quand postId présent', () => {
    expect(notificationHref({ type: 'like', postId: 'abc', commentId: '', conversationId: '' })).toBe(
      '/posts/abc',
    )
  })

  it('repli sur le fil sans postId', () => {
    expect(notificationHref({ type: 'like', postId: '', commentId: '', conversationId: '' })).toBe(
      '/feed',
    )
  })

  it('pointe vers la conversation pour une mention en message', () => {
    expect(
      notificationHref({ type: 'message_mention', postId: '', commentId: '', conversationId: 'cv1' }),
    ).toBe('/messages?conv=cv1')
  })

  it('cible le commentaire pour un commentaire/réponse', () => {
    expect(
      notificationHref({ type: 'comment', postId: 'abc', commentId: 'c1', conversationId: '' }),
    ).toBe('/posts/abc?comment=c1')
    expect(
      notificationHref({ type: 'reply', postId: 'abc', commentId: 'c2', conversationId: '' }),
    ).toBe('/posts/abc?comment=c2')
  })

  it('repli sur le post seul si le commentaire est absent', () => {
    expect(
      notificationHref({ type: 'comment', postId: 'abc', commentId: '', conversationId: '' }),
    ).toBe('/posts/abc')
  })
})

describe('toWebSocketUrl', () => {
  it('convertit http en ws et ajoute le token', () => {
    expect(toWebSocketUrl('http://localhost:8080', 'abc')).toBe(
      'ws://localhost:8080/notifications/ws?access_token=abc',
    )
  })

  it('convertit https en wss et ignore un slash final', () => {
    expect(toWebSocketUrl('https://api.example.com/', 'tok')).toBe(
      'wss://api.example.com/notifications/ws?access_token=tok',
    )
  })

  it('encode les caractères spéciaux du token', () => {
    expect(toWebSocketUrl('http://h', 'a b/c+d')).toContain('access_token=a%20b%2Fc%2Bd')
  })
})
