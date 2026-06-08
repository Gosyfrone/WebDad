import { describe, it, expect } from 'vitest'

import { toWebSocketUrl } from './messages'

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
