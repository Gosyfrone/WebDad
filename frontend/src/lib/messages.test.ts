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
    deletedAt: '',
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

import { applyReceipt, planReceipts, type Conversation } from './messages'

function conv(type: Conversation['type'], receipts: Conversation['memberReceipts']): Conversation {
  return {
    id: 'c',
    type,
    memberIds: [],
    title: '',
    myRole: 'talker',
    createdBy: '',
    createdAt: '',
    updatedAt: '',
    pinnedAt: '',
    lastReadAt: '',
    muted: false,
    memberReceipts: receipts,
    contentKey: null,
  }
}

function mineAt(id: string, iso: string): ChatMessage {
  return { ...msg(id, true), createdAt: iso }
}

describe('planReceipts (accusés Envoyé/Ouvert, sous le dernier message)', () => {
  const m1 = mineAt('m1', '2026-06-08T10:00:00Z')
  const m2 = mineAt('m2', '2026-06-08T11:00:00Z')
  const m3 = mineAt('m3', '2026-06-08T12:00:00Z')

  it('DM : la marque suit le dernier message (Envoyé tant que non lu)', () => {
    const c = conv('dm', [{ userId: 'p', deliveredAt: '2026-06-08T11:30:00Z', readAt: '2026-06-08T10:30:00Z' }])
    const marks = planReceipts([m1, m2, m3], c)
    // Seul m3 (le dernier) porte la marque, et il n'est pas encore lu → Envoyé.
    expect(marks.get('m3')).toEqual({ kind: 'delivered' })
    expect(marks.has('m1')).toBe(false)
    expect(marks.has('m2')).toBe(false)
  })

  it('DM : dernier message lu → 2 coches sous le dernier', () => {
    const c = conv('dm', [{ userId: 'p', deliveredAt: '2026-06-08T12:30:00Z', readAt: '2026-06-08T12:30:00Z' }])
    const marks = planReceipts([m1, m2, m3], c)
    expect(marks.get('m3')).toEqual({ kind: 'read-all' })
    expect(marks.size).toBe(1)
  })

  it('DM : pas d’info destinataire → Envoyé sous le dernier', () => {
    const c = conv('dm', [])
    expect(planReceipts([m1, m2], c).get('m2')).toEqual({ kind: 'delivered' })
  })

  it('groupe : personne ne l’a lu → 1 coche grise (Envoyé) sous le dernier', () => {
    const c = conv('group', [
      { userId: 'a', deliveredAt: '2026-06-08T12:30:00Z', readAt: '' },
      { userId: 'b', deliveredAt: '', readAt: '' },
    ])
    expect(planReceipts([m1, m3], c).get('m3')).toEqual({ kind: 'delivered' })
  })

  it('groupe : lu par une partie → 1 coche couleur + nombre', () => {
    const c = conv('group', [
      { userId: 'a', deliveredAt: '2026-06-08T12:30:00Z', readAt: '2026-06-08T12:30:00Z' },
      { userId: 'b', deliveredAt: '2026-06-08T12:30:00Z', readAt: '' },
    ])
    expect(planReceipts([m3], c).get('m3')).toEqual({ kind: 'read-count', count: 1 })
  })

  it('groupe : lu par tous → 2 coches', () => {
    const c = conv('group', [
      { userId: 'a', deliveredAt: '', readAt: '2026-06-08T12:30:00Z' },
      { userId: 'b', deliveredAt: '', readAt: '2026-06-08T12:30:00Z' },
    ])
    expect(planReceipts([m3], c).get('m3')).toEqual({ kind: 'read-all' })
  })

  it('communauté : aucun accusé', () => {
    const c = conv('community', [{ userId: 'a', deliveredAt: '2026-06-08T12:30:00Z', readAt: '2026-06-08T12:30:00Z' }])
    expect(planReceipts([m3], c).size).toBe(0)
  })
})

describe('applyReceipt (mise à jour temps réel)', () => {
  it('avance les curseurs et ne recule jamais', () => {
    const c = conv('dm', [{ userId: 'p', deliveredAt: '2026-06-08T11:00:00Z', readAt: '2026-06-08T10:00:00Z' }])
    const next = applyReceipt(c, 'p', '2026-06-08T12:00:00Z', '2026-06-08T09:00:00Z')
    expect(next.memberReceipts[0].deliveredAt).toBe('2026-06-08T12:00:00Z') // avancé
    expect(next.memberReceipts[0].readAt).toBe('2026-06-08T10:00:00Z') // pas reculé
  })

  it('crée l’entrée si le membre est absent', () => {
    const c = conv('group', [])
    const next = applyReceipt(c, 'x', '2026-06-08T12:00:00Z', '')
    expect(next.memberReceipts).toEqual([{ userId: 'x', deliveredAt: '2026-06-08T12:00:00Z', readAt: '' }])
  })

  it('communauté : inchangée', () => {
    const c = conv('community', [])
    expect(applyReceipt(c, 'x', '2026-06-08T12:00:00Z', '')).toBe(c)
  })
})
