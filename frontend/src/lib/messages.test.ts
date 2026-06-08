import { describe, it, expect } from 'vitest'

import { buildMessagePage, toWebSocketUrl, type ChatMessage } from './messages'

function msg(id: string): ChatMessage {
  return {
    id,
    conversationId: 'c',
    senderId: 's',
    text: 'x',
    decrypted: true,
    createdAt: '2026-06-08T00:00:00Z',
    mine: false,
  }
}

describe('buildMessagePage (pagination par curseur)', () => {
  it('page pleine → hasMore=true, oldestId = plus ancien (premier)', () => {
    const page = buildMessagePage([msg('a'), msg('b'), msg('c')], 3)
    expect(page.hasMore).toBe(true)
    expect(page.oldestId).toBe('a')
    expect(page.messages).toHaveLength(3)
  })

  it('page partielle → hasMore=false (haut de la conversation atteint)', () => {
    const page = buildMessagePage([msg('a'), msg('b')], 3)
    expect(page.hasMore).toBe(false)
    expect(page.oldestId).toBe('a')
  })

  it('page vide → oldestId null, hasMore false', () => {
    const page = buildMessagePage([], 30)
    expect(page.oldestId).toBeNull()
    expect(page.hasMore).toBe(false)
  })
})

describe('toWebSocketUrl', () => {
  it('convertit http en ws et ajoute le token', () => {
    expect(toWebSocketUrl('http://localhost:8080', 'abc')).toBe(
      'ws://localhost:8080/messages/ws?access_token=abc',
    )
  })

  it('convertit https en wss', () => {
    expect(toWebSocketUrl('https://api.example.com', 'tok')).toBe(
      'wss://api.example.com/messages/ws?access_token=tok',
    )
  })

  it('ignore un slash final sur la base', () => {
    expect(toWebSocketUrl('http://localhost:8080/', 't')).toBe(
      'ws://localhost:8080/messages/ws?access_token=t',
    )
  })

  it('encode les caractères spéciaux du token', () => {
    expect(toWebSocketUrl('http://h', 'a b/c+d')).toContain('access_token=a%20b%2Fc%2Bd')
  })
})
