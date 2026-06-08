import { describe, it, expect } from 'vitest'

import {
  buildMessagePage,
  computeDivider,
  searchMessages,
  toWebSocketUrl,
  type ChatMessage,
} from './messages'

function msg(id: string, mine = false): ChatMessage {
  return {
    id,
    conversationId: 'c',
    senderId: mine ? 'me' : 's',
    text: 'x',
    decrypted: true,
    createdAt: '2026-06-08T00:00:00Z',
    mine,
  }
}

function textMsg(id: string, text: string, decrypted = true): ChatMessage {
  return { ...msg(id), text, decrypted }
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

describe('computeDivider (ligne « Nouveaux messages »)', () => {
  it('pas d\'ancre (1re ouverture) → null', () => {
    expect(computeDivider([msg('a'), msg('b')], null)).toBeNull()
  })

  it('scénario A→B(moi)→C : séparateur avant C', () => {
    // J\'ai lu A, j\'ai envoyé B (à moi), je reviens avec C (d\'autrui).
    const messages = [msg('a'), msg('b', true), msg('c')]
    expect(computeDivider(messages, 'b')).toBe('c')
  })

  it('rien de nouveau après l\'ancre → null', () => {
    expect(computeDivider([msg('a'), msg('b')], 'b')).toBeNull()
  })

  it('ignore mes propres messages après l\'ancre', () => {
    // Après l\'ancre A, seuls mes messages → pas de « nouveaux messages ».
    expect(computeDivider([msg('a'), msg('b', true), msg('c', true)], 'a')).toBeNull()
  })

  it('place le séparateur avant le 1ᵉʳ message d\'autrui après l\'ancre', () => {
    expect(computeDivider([msg('a'), msg('b', true), msg('c'), msg('d')], 'a')).toBe('c')
  })

  it('ancre hors page → 1ᵉʳ message d\'autrui plus récent que l\'ancre', () => {
    // L\'ancre 'a' n\'est pas dans la page chargée [b,c,d] ; b>a et d\'autrui.
    expect(computeDivider([msg('b'), msg('c'), msg('d')], 'a')).toBe('b')
  })
})

describe('searchMessages (recherche dans une conversation)', () => {
  const all = [
    textMsg('a', 'Bonjour tout le monde'),
    textMsg('b', 'On se retrouve à MIDI'),
    textMsg('c', 'message chiffré illisible', false), // non déchiffré → ignoré
    textMsg('d', 'rendez-vous à midi pile'),
  ]

  it('requête vide → aucun résultat', () => {
    expect(searchMessages(all, '   ')).toHaveLength(0)
  })

  it('insensible à la casse, sous-chaîne', () => {
    const res = searchMessages(all, 'midi')
    expect(res.map((m) => m.id)).toEqual(['b', 'd'])
  })

  it('ignore les messages non déchiffrés', () => {
    expect(searchMessages(all, 'illisible')).toHaveLength(0)
  })

  it('conserve l\'ordre d\'entrée', () => {
    const res = searchMessages(all, 'on') // « bONjour » / « ON se retrouve »
    expect(res.map((m) => m.id)).toEqual(['a', 'b'])
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
