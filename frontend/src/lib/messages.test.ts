import { describe, it, expect } from 'vitest'

import {
  buildMessagePage,
  computeDivider,
  decodeMessageBody,
  encodeMessageBody,
  mediaKind,
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
    originalText: '',
    media: [],
    decrypted: true,
    createdAt: '2026-06-08T00:00:00Z',
    editedAt: '',
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
  // L'ancre est un curseur de lecture (`lastReadAt`, ISO) : le séparateur se pose
  // avant le 1ᵉʳ message d'autrui POSTÉRIEUR à ce curseur.
  const T1 = '2026-06-08T10:00:00Z'
  const T2 = '2026-06-08T10:01:00Z'
  const T3 = '2026-06-08T10:02:00Z'
  const T4 = '2026-06-08T10:03:00Z'
  function at(id: string, createdAt: string, mine = false): ChatMessage {
    return { ...msg(id, mine), createdAt }
  }

  it('pas d\'ancre (jamais lu) → null', () => {
    expect(computeDivider([at('a', T1), at('b', T2)], null)).toBeNull()
  })

  it('scénario A→B(moi)→C : séparateur avant C', () => {
    // Lu jusqu'à T2 ; B (à moi) ignoré ; C (d'autrui, T3) est nouveau.
    const messages = [at('a', T1), at('b', T2, true), at('c', T3)]
    expect(computeDivider(messages, T2)).toBe('c')
  })

  it('rien de nouveau après l\'ancre → null', () => {
    expect(computeDivider([at('a', T1), at('b', T2)], T2)).toBeNull()
  })

  it('ignore mes propres messages après l\'ancre', () => {
    expect(computeDivider([at('a', T1), at('b', T2, true), at('c', T3, true)], T1)).toBeNull()
  })

  it('place le séparateur avant le 1ᵉʳ message d\'autrui après l\'ancre', () => {
    expect(computeDivider([at('a', T1), at('b', T2, true), at('c', T3), at('d', T4)], T1)).toBe('c')
  })

  it('ancre invalide → null', () => {
    expect(computeDivider([at('a', T1), at('b', T2)], 'pas-une-date')).toBeNull()
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

describe('encodeMessageBody / decodeMessageBody (enveloppe pièces jointes)', () => {
  const att = { id: 'm1', nonce: 'bm9uY2U=', mime: 'image/png', name: 'a.png', size: 12 }

  it('sans média → texte brut (rétrocompatible, pas de JSON)', () => {
    expect(encodeMessageBody('coucou', [])).toBe('coucou')
  })

  it('avec média → enveloppe JSON v1', () => {
    const body = encodeMessageBody('légende', [att])
    expect(JSON.parse(body)).toEqual({ v: 1, text: 'légende', media: [att] })
  })

  it('round-trip avec média', () => {
    const decoded = decodeMessageBody(encodeMessageBody('cap', [att]))
    expect(decoded).toEqual({ text: 'cap', media: [att] })
  })

  it('message texte historique (chaîne brute) → texte, aucun média', () => {
    expect(decodeMessageBody('un vieux message')).toEqual({ text: 'un vieux message', media: [] })
  })

  it("texte ressemblant à du JSON mais sans v:1 → traité comme texte brut", () => {
    expect(decodeMessageBody('{"foo":1}')).toEqual({ text: '{"foo":1}', media: [] })
  })
})

describe('mediaKind', () => {
  it('classe par préfixe MIME', () => {
    expect(mediaKind('image/jpeg')).toBe('image')
    expect(mediaKind('video/mp4')).toBe('video')
    expect(mediaKind('application/pdf')).toBe('file')
  })
})
