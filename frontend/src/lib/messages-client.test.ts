import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// Couche réseau + E2EE de `lib/messages`. On mocke `apiFetch` (routeur par URL)
// et toute la couche cryptographique par des stubs DÉTERMINISTES :
//   - encryptText(ck, t) → { ciphertext: 'ct:'+t, nonce: 'n1' }
//   - decryptText(ck, ct) → ct sans le préfixe 'ct:' (lève si absent)
// → on pilote chiffrement/déchiffrement sans vraies primitives.
const apiFetch = vi.fn()
const getAccessToken = vi.fn(() => 'tok')
vi.mock('@/lib/auth-client', () => ({
  apiFetch: (...a: unknown[]) => apiFetch(...a),
  getAccessToken: () => getAccessToken(),
}))

const u8 = (n: number) => new Uint8Array([n])
vi.mock('@/lib/crypto', () => ({
  toBase64: (x: Uint8Array) => 'b64:' + Array.from(x).join(','),
  fromBase64: (s: string) => new Uint8Array([s.length % 251]),
  generateIdentityKeyPair: () => ({ privateKey: u8(1), publicKey: u8(2) }),
  generateContentKey: () => u8(7),
  derivePublicKey: () => u8(9),
  sealKeyForRecipient: (pub: string, _ck: Uint8Array) => `env(${pub})`,
  openKeyEnvelope: (env: string) => {
    if (env === 'bad') throw new Error('wrong key')
    return u8(5)
  },
  encryptText: (_ck: Uint8Array, text: string) => ({ ciphertext: 'ct:' + text, nonce: 'n1' }),
  decryptText: (_ck: Uint8Array, ct: string) => {
    if (!String(ct).startsWith('ct:')) throw new Error('undecipherable')
    return String(ct).slice(3)
  },
  encryptSymmetric: () => ({ nonce: u8(3), ciphertext: u8(4) }),
  decryptSymmetric: () => u8(8),
}))

type Identity = { privateKey: Uint8Array; publicKey: Uint8Array } | null
const getStoredIdentity = vi.fn(async (..._a: unknown[]): Promise<Identity> => ({ privateKey: u8(1), publicKey: u8(2) }))
const getLegacyIdentity = vi.fn(async (): Promise<Identity> => null)
const setStoredIdentity = vi.fn(async (..._a: unknown[]) => {})
vi.mock('@/lib/key-store', () => ({
  getStoredIdentity: (...a: unknown[]) => getStoredIdentity(...a),
  getLegacyIdentity: () => getLegacyIdentity(),
  setStoredIdentity: (...a: unknown[]) => setStoredIdentity(...a),
}))

const createBackupAsync = vi.fn(async (..._a: unknown[]) => ({ ct: 'x', kdf: 'p' }))
const openBackupAsync = vi.fn(async (..._a: unknown[]) => u8(1))
vi.mock('@/lib/key-backup-async', () => ({
  createBackupAsync: (...a: unknown[]) => createBackupAsync(...a),
  openBackupAsync: (...a: unknown[]) => openBackupAsync(...a),
}))

const sameKDFParams = vi.fn((..._a: unknown[]) => true)
vi.mock('@/lib/key-backup', () => ({
  sameKDFParams: (...a: unknown[]) => sameKDFParams(...a),
  parseKDFParams: () => ({}),
  DEFAULT_KDF_PARAMS: {},
}))

const fetchMediaBytes = vi.fn(async (..._a: unknown[]) => u8(4))
const uploadEncryptedMedia = vi.fn(async (..._a: unknown[]) => ({ id: 'mid' }))
vi.mock('@/lib/media', () => ({
  fetchMediaBytes: (...a: unknown[]) => fetchMediaBytes(...a),
  uploadEncryptedMedia: (...a: unknown[]) => uploadEncryptedMedia(...a),
}))

const currentUserId = vi.fn(() => 'me')
vi.mock('@/lib/session', () => ({ currentUserId: () => currentUserId() }))

import * as m from '@/lib/messages'

function json(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  return { ok: init.ok ?? true, status: init.status ?? 200, json: () => Promise.resolve(body) }
}
function routes(table: Array<[(url: string) => boolean, unknown]>) {
  apiFetch.mockImplementation(async (url: string) => {
    for (const [match, res] of table) if (match(url)) return res
    return json({ error: 'not found' }, { ok: false, status: 404 })
  })
}
const path = (p: string) => (url: string) => url.split('?')[0] === p
const starts = (p: string) => (url: string) => url.startsWith(p)

function apiConv(over: Record<string, unknown> = {}) {
  return {
    id: 'cv1',
    type: 'dm',
    member_ids: ['me', 'peer'],
    my_role: 'member',
    my_envelope: 'env-self',
    created_by: 'me',
    created_at: '2026-01-01',
    updated_at: '2026-01-02',
    ...over,
  }
}
/** Conversation front avec clé de contenu présente (E2EE déchiffrable). */
function conv(over: Partial<m.Conversation> = {}): m.Conversation {
  return {
    id: 'cv1',
    type: 'dm',
    memberIds: ['me', 'peer'],
    title: '',
    myRole: 'member',
    createdBy: 'me',
    createdAt: '',
    updatedAt: '',
    pinnedAt: '',
    lastReadAt: '',
    muted: false,
    memberReceipts: [],
    contentKey: u8(5),
    ...over,
  }
}

beforeEach(() => {
  getAccessToken.mockReturnValue('tok')
  currentUserId.mockReturnValue('me')
  getStoredIdentity.mockResolvedValue({ privateKey: u8(1), publicKey: u8(2) })
  getLegacyIdentity.mockResolvedValue(null)
  sameKDFParams.mockReturnValue(true)
})
afterEach(() => {
  apiFetch.mockReset()
  vi.clearAllMocks()
})

// ─── identité E2EE (doit rester en tête : `identityPromise` est mémoïsée) ─────

describe('identité E2EE', () => {
  it('ensureMyKeys lève IdentityLockedError sans clé locale', async () => {
    getStoredIdentity.mockResolvedValue(null)
    routes([])
    await expect(m.ensureMyKeys()).rejects.toBeInstanceOf(m.IdentityLockedError)
  })

  it('getIdentityState: clé locale + sauvegarde → ready', async () => {
    routes([[path('/messages/keys/backup/status'), json({ data: { exists: true } })]])
    expect(await m.getIdentityState()).toBe('ready')
  })

  it('getIdentityState: sauvegarde sans clé locale → unlock', async () => {
    getStoredIdentity.mockResolvedValue(null)
    routes([
      [path('/messages/keys/backup/status'), json({ data: { exists: true } })],
    ])
    expect(await m.getIdentityState()).toBe('unlock')
  })

  it('getIdentityState: pas de sauvegarde → setup', async () => {
    getStoredIdentity.mockResolvedValue(null)
    routes([[path('/messages/keys/backup/status'), json({ data: { exists: false } })]])
    expect(await m.getIdentityState()).toBe('setup')
  })

  it('migrateLegacyIdentity adopte la clé self si la clé publique correspond', async () => {
    getStoredIdentity.mockResolvedValue(null)
    getLegacyIdentity.mockResolvedValue({ privateKey: u8(1), publicKey: u8(2) })
    routes([
      [starts('/messages/keys/me'), json({ data: { user_id: 'me', public_key: 'b64:2' } })],
      [path('/messages/keys/backup/status'), json({ data: { exists: false } })],
    ])
    await m.getIdentityState()
    expect(setStoredIdentity).toHaveBeenCalled()
  })

  it('setupPassphrase uploade la sauvegarde, persiste et publie la clé', async () => {
    routes([
      [path('/messages/keys/backup'), json({ data: {} })],
      [path('/messages/keys'), json({ data: {} })],
    ])
    await m.setupPassphrase('hunter2')
    expect(createBackupAsync).toHaveBeenCalled()
    expect(setStoredIdentity).toHaveBeenCalled()
    const put = apiFetch.mock.calls.find((c) => String(c[0]) === '/messages/keys')!
    expect(put[1].method).toBe('PUT')
  })

  it('unlockWithPassphrase déballe la sauvegarde et persiste l’identité', async () => {
    sameKDFParams.mockReturnValue(true) // pas d'upgrade
    routes([
      [path('/messages/keys/backup'), json({ data: { ct: 'x' } })],
      [path('/messages/keys'), json({ data: {} })],
    ])
    await m.unlockWithPassphrase('hunter2')
    expect(openBackupAsync).toHaveBeenCalled()
    expect(setStoredIdentity).toHaveBeenCalled()
  })

  it('unlockWithPassphrase ré-emballe une sauvegarde aux paramètres lourds', async () => {
    sameKDFParams.mockReturnValue(false) // déclenche maybeUpgradeBackup
    routes([
      [path('/messages/keys/backup'), json({ data: { ct: 'x' } })],
      [path('/messages/keys'), json({ data: {} })],
    ])
    await m.unlockWithPassphrase('hunter2')
    await Promise.resolve() // laisse tourner le fire-and-forget
    expect(createBackupAsync).toHaveBeenCalled()
  })
})

// ─── DM / groupes ─────────────────────────────────────────────────────────────

describe('startDM / createGroup / invitations', () => {
  it('startDM scelle la clé pour les deux membres et mappe la conversation', async () => {
    routes([
      [path('/messages/keys/peer'), json({ data: { user_id: 'peer', public_key: 'b64:peer' } })],
      [path('/messages/conversations'), json({ data: apiConv() })],
    ])
    const c = await m.startDM('peer')
    expect(c.id).toBe('cv1')
    expect(c.type).toBe('dm')
    const body = JSON.parse(apiFetch.mock.calls.find((x) => x[1])![1].body)
    expect(body.peer_id).toBe('peer')
    expect(Object.keys(body.envelopes)).toEqual(['me', 'peer'])
  })

  it('fetchPeerPublicKey: 404 → PeerKeyMissingError', async () => {
    routes([[path('/messages/keys/peer'), json({ error: 'x' }, { ok: false, status: 404 })]])
    await expect(m.startDM('peer')).rejects.toBeInstanceOf(m.PeerKeyMissingError)
  })

  it('createGroup chiffre le nom et scelle pour chaque membre', async () => {
    routes([
      [path('/messages/keys/a'), json({ data: { user_id: 'a', public_key: 'b64:a' } })],
      [path('/messages/keys/b'), json({ data: { user_id: 'b', public_key: 'b64:b' } })],
      [path('/messages/conversations'), json({ data: apiConv({ type: 'group' }) })],
    ])
    await m.createGroup('Team', ['a', 'b', 'me', ''])
    const body = JSON.parse(apiFetch.mock.calls.find((x) => x[1] && String(x[0]) === '/messages/conversations')![1].body)
    expect(body.type).toBe('group')
    expect(body.title).toBe('ct:Team')
    expect(Object.keys(body.envelopes).sort()).toEqual(['a', 'b', 'me'])
  })

  it('inviteToGroup sans clé locale → 412', async () => {
    await expect(m.inviteToGroup(conv({ contentKey: null }), 'x')).rejects.toMatchObject({ status: 412 })
  })

  it('inviteToGroup scelle la clé et POST le membre', async () => {
    routes([
      [path('/messages/keys/x'), json({ data: { user_id: 'x', public_key: 'b64:x' } })],
      [starts('/messages/conversations/cv1/members'), json({}, { ok: true })],
    ])
    await m.inviteToGroup(conv(), 'x')
    const call = apiFetch.mock.calls.find((c) => String(c[0]) === '/messages/conversations/cv1/members')!
    expect(JSON.parse(call[1].body).user_id).toBe('x')
  })

  it('removeMember / leaveGroup / deleteGroup ciblent les bonnes routes', async () => {
    routes([[starts('/messages/conversations/cv1'), json({}, { ok: true })]])
    await m.removeMember(conv(), 'x')
    await m.leaveGroup(conv())
    await m.deleteGroup(conv())
    const urls = apiFetch.mock.calls.map((c) => String(c[0]))
    expect(urls).toContain('/messages/conversations/cv1/members/x')
    expect(urls).toContain('/messages/conversations/cv1/members/me')
    expect(urls).toContain('/messages/conversations/cv1')
  })

  it('removeMember échec → erreur explicite', async () => {
    routes([[starts('/messages/conversations/cv1'), json({}, { ok: false, status: 403 })]])
    await expect(m.removeMember(conv(), 'x')).rejects.toThrow('Exclusion impossible')
  })

  it('renameGroup sans clé → 412 ; avec clé → PATCH titre chiffré', async () => {
    await expect(m.renameGroup(conv({ contentKey: null }), 'N')).rejects.toMatchObject({ status: 412 })
    routes([[path('/messages/conversations/cv1'), json({ data: apiConv({ type: 'group' }) })]])
    await m.renameGroup(conv(), 'New')
    const call = apiFetch.mock.calls.find((c) => c[1] && String(c[0]) === '/messages/conversations/cv1')!
    expect(JSON.parse(call[1].body).title).toBe('ct:New')
  })

  it('listMembers mappe id + rôle', async () => {
    routes([[path('/messages/conversations/cv1/members'), json({ data: [{ user_id: 'a', role: 'owner' }] })]])
    expect(await m.listMembers('cv1')).toEqual([{ userId: 'a', role: 'owner' }])
  })
})

// ─── communautés ──────────────────────────────────────────────────────────────

describe('communautés', () => {
  it('createCommunity confie la clé au serveur, nom en clair', async () => {
    routes([[path('/messages/conversations'), json({ data: apiConv({ type: 'community', title: 'Devs', content_key: 'b64:7' }) })]])
    const c = await m.createCommunity('Devs')
    expect(c.type).toBe('community')
    expect(c.title).toBe('Devs')
    const body = JSON.parse(apiFetch.mock.calls.find((x) => x[1])![1].body)
    expect(body.type).toBe('community')
    expect(body.content_key).toBeTruthy()
  })

  it('listCommunities mappe et passe q', async () => {
    routes([[starts('/messages/communities'), json({ data: [{ id: 'c1', title: 'T', member_count: 3, is_member: true, created_by: 'me', created_at: '', updated_at: '' }] })]])
    const list = await m.listCommunities('dev')
    expect(list[0]).toMatchObject({ id: 'c1', memberCount: 3, isMember: true })
    expect(String(apiFetch.mock.calls[0][0])).toContain('q=dev')
  })

  it('joinCommunity mappe la conversation rejointe', async () => {
    routes([[path('/messages/conversations/c1/join'), json({ data: apiConv({ id: 'c1', type: 'community', content_key: 'b64:7', title: 'T' }) })]])
    const c = await m.joinCommunity('c1')
    expect(c.id).toBe('c1')
  })

  it('setMemberRole PATCH le rôle', async () => {
    routes([[starts('/messages/conversations/cv1/members/u'), json({}, { ok: true })]])
    await m.setMemberRole(conv(), 'u', 'talker')
    const call = apiFetch.mock.calls[0]
    expect(JSON.parse(call[1].body)).toEqual({ role: 'talker' })
  })

  it('renameCommunity PATCH le nom en clair', async () => {
    routes([[path('/messages/conversations/cv1'), json({ data: apiConv({ type: 'community', title: 'X', content_key: 'b64:7' }) })]])
    await m.renameCommunity(conv(), 'X')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({ title: 'X' })
  })
})

// ─── liste / état des conversations + toConversation ─────────────────────────

describe('conversations (mapping + état)', () => {
  it('listConversations déchiffre le titre des groupes', async () => {
    routes([[path('/messages/conversations'), json({ data: [apiConv({ type: 'group', title: 'ct:Squad', title_nonce: 'n1' })] })]])
    const [c] = await m.listConversations()
    expect(c.title).toBe('Squad')
  })

  it('toConversation: enveloppe pour une autre clé → contentKey null, titre vide', async () => {
    routes([[path('/messages/conversations'), json({ data: [apiConv({ type: 'group', my_envelope: 'bad', title: 'ct:Squad', title_nonce: 'n1' })] })]])
    const [c] = await m.listConversations()
    expect(c.contentKey).toBeNull()
    expect(c.title).toBe('')
  })

  it('toConversation: titre indéchiffrable → vide', async () => {
    routes([[path('/messages/conversations'), json({ data: [apiConv({ type: 'group', title: 'plain', title_nonce: 'n1' })] })]])
    const [c] = await m.listConversations()
    expect(c.title).toBe('')
  })

  it('toConversation mappe les accusés des autres membres', async () => {
    routes([[path('/messages/conversations'), json({ data: [apiConv({ member_receipts: [{ user_id: 'peer', delivered_at: 'd', read_at: 'r' }], pinned_at: 'p', muted: true, last_read_at: 'lr' })] })]])
    const [c] = await m.listConversations()
    expect(c.memberReceipts[0]).toEqual({ userId: 'peer', deliveredAt: 'd', readAt: 'r' })
    expect(c.pinnedAt).toBe('p')
    expect(c.muted).toBe(true)
    expect(c.lastReadAt).toBe('lr')
  })

  it('getConversation mappe un détail', async () => {
    routes([[path('/messages/conversations/cv1'), json({ data: apiConv() })]])
    expect((await m.getConversation('cv1')).id).toBe('cv1')
  })

  it('pin/unpin/mute/unmute ciblent les bonnes routes et méthodes', async () => {
    routes([[starts('/messages/conversations/cv1/'), json({ data: apiConv() })]])
    await m.pinConversation(conv())
    await m.unpinConversation(conv())
    await m.muteConversation(conv())
    await m.unmuteConversation(conv())
    const seen = apiFetch.mock.calls.map((c) => `${c[1].method} ${String(c[0])}`)
    expect(seen).toContain('PATCH /messages/conversations/cv1/pin')
    expect(seen).toContain('DELETE /messages/conversations/cv1/pin')
    expect(seen).toContain('PATCH /messages/conversations/cv1/mute')
    expect(seen).toContain('DELETE /messages/conversations/cv1/mute')
  })

  it('clearConversation / markConversationRead', async () => {
    routes([[starts('/messages/conversations/cv1/'), json({}, { ok: true })], [path('/messages/conversations/cv1/me'), json({}, { ok: true })]])
    await m.clearConversation(conv())
    await m.markConversationRead('cv1')
    const seen = apiFetch.mock.calls.map((c) => `${c[1].method} ${String(c[0])}`)
    expect(seen).toContain('DELETE /messages/conversations/cv1/me')
    expect(seen).toContain('PUT /messages/conversations/cv1/read')
  })

  it('getMessagesUnreadCount renvoie le compteur (0 par défaut)', async () => {
    routes([[path('/messages/unread-count'), json({ data: { count: 4 } })]])
    expect(await m.getMessagesUnreadCount()).toBe(4)
    routes([[path('/messages/unread-count'), json({ data: {} })]])
    expect(await m.getMessagesUnreadCount()).toBe(0)
  })
})

// ─── messages : chiffrement, déchiffrement, envoi ────────────────────────────

describe('decryptMessage', () => {
  function apiMsg(over: Record<string, unknown> = {}) {
    return { id: 'm1', conversation_id: 'cv1', sender_id: 'peer', ciphertext: 'ct:hello', nonce: 'n', created_at: 't', ...over }
  }

  it('déchiffre texte simple', () => {
    const msg = m.decryptMessage(conv(), apiMsg(), 'me')
    expect(msg.text).toBe('hello')
    expect(msg.decrypted).toBe(true)
    expect(msg.mine).toBe(false)
  })

  it('message supprimé (tombstone) → texte vide, decrypted true', () => {
    const msg = m.decryptMessage(conv(), apiMsg({ deleted_at: 'x', deleted_by_moderation: true }), 'me')
    expect(msg.deletedAt).toBe('x')
    expect(msg.deletedByModeration).toBe(true)
    expect(msg.text).toBe('')
  })

  it('clé absente → non déchiffré', () => {
    const msg = m.decryptMessage(conv({ contentKey: null }), apiMsg(), 'me')
    expect(msg.decrypted).toBe(false)
  })

  it('ciphertext illisible → decrypted false', () => {
    const msg = m.decryptMessage(conv(), apiMsg({ ciphertext: 'garbage' }), 'me')
    expect(msg.decrypted).toBe(false)
  })

  it('enveloppe média + version originale (édité)', () => {
    const env = 'ct:' + JSON.stringify({ v: 1, text: 'hi', media: [{ id: 'a', nonce: 'n', mime: 'image/png', name: 'x', size: 1 }] })
    const orig = 'ct:' + JSON.stringify({ v: 1, text: 'old', media: [] })
    const msg = m.decryptMessage(conv(), apiMsg({ ciphertext: env, original_ciphertext: orig, original_nonce: 'n', edited_at: 'e', sender_id: 'me' }), 'me')
    expect(msg.text).toBe('hi')
    expect(msg.media[0].type).toBeDefined()
    expect(msg.originalText).toBe('old')
    expect(msg.editedAt).toBe('e')
    expect(msg.mine).toBe(true)
  })
})

describe('listMessages / listMessagesPage / listAllMessages', () => {
  function rawMsg(id: string, text: string) {
    return { id, conversation_id: 'cv1', sender_id: 'peer', ciphertext: 'ct:' + text, nonce: 'n', created_at: 't' }
  }

  it('listMessages déchiffre la page', async () => {
    routes([[starts('/messages/conversations/cv1/messages'), json({ data: [rawMsg('m1', 'a'), rawMsg('m2', 'b')] })]])
    const msgs = await m.listMessages(conv())
    expect(msgs.map((x) => x.text)).toEqual(['a', 'b'])
  })

  it('listMessages passe le curseur before', async () => {
    routes([[starts('/messages/conversations/cv1/messages'), json({ data: [] })]])
    await m.listMessages(conv(), 10, 'mX')
    expect(String(apiFetch.mock.calls[0][0])).toContain('before=mX')
  })

  it('listAllMessages pagine jusqu’à épuisement', async () => {
    let call = 0
    apiFetch.mockImplementation(async () => {
      call++
      // 1re page pleine (50) → hasMore, 2e page partielle → stop
      const data = call === 1
        ? Array.from({ length: 50 }, (_, i) => rawMsg('a' + i, 't' + i))
        : [rawMsg('b0', 'old')]
      return json({ data })
    })
    const all = await m.listAllMessages(conv())
    expect(all.length).toBe(51)
    expect(call).toBe(2)
  })
})

describe('sendMessage / editMessage / deleteMessage', () => {
  it('sendMessage sans clé → 412', async () => {
    await expect(m.sendMessage(conv({ contentKey: null }), 'hi')).rejects.toMatchObject({ status: 412 })
  })

  it('sendMessage chiffre le texte et renvoie le message déchiffré', async () => {
    routes([[starts('/messages/conversations/cv1/messages'), json({ data: { id: 'm9', conversation_id: 'cv1', sender_id: 'me', ciphertext: 'ct:hi', nonce: 'n', created_at: 't' } })]])
    const msg = await m.sendMessage(conv(), 'hi', [], ['peer'])
    expect(msg.text).toBe('hi')
    const body = JSON.parse(apiFetch.mock.calls.find((c) => c[1])![1].body)
    expect(body.ciphertext).toBe('ct:hi')
    expect(body.mentioned_member_ids).toEqual(['peer'])
  })

  it('sendMessage chiffre et uploade les pièces jointes', async () => {
    routes([[starts('/messages/conversations/cv1/messages'), json({ data: { id: 'm9', conversation_id: 'cv1', sender_id: 'me', ciphertext: 'ct:doc', nonce: 'n', created_at: 't' } })]])
    const file = new File([new Uint8Array([1, 2, 3])], 'a.png', { type: 'image/png' })
    await m.sendMessage(conv(), 'doc', [file])
    expect(uploadEncryptedMedia).toHaveBeenCalled()
  })

  it('editMessage re-chiffre en conservant les pièces jointes', async () => {
    routes([[starts('/messages/conversations/cv1/messages/m1'), json({ data: { id: 'm1', conversation_id: 'cv1', sender_id: 'me', ciphertext: 'ct:new', nonce: 'n', created_at: 't', edited_at: 'e' } })]])
    const existing = { id: 'm1', media: [{ id: 'a', nonce: 'n', mime: 'image/png', name: 'x', size: 1, type: 'image' as const, url: '' }] } as unknown as m.ChatMessage
    const msg = await m.editMessage(conv(), existing, 'new')
    expect(msg.text).toBe('new')
    expect(apiFetch.mock.calls[0][1].method).toBe('PATCH')
  })

  it('editMessage sans clé → 412', async () => {
    await expect(m.editMessage(conv({ contentKey: null }), { id: 'm1', media: [] } as unknown as m.ChatMessage, 'x')).rejects.toMatchObject({ status: 412 })
  })

  it('deleteMessage renvoie le tombstone déchiffré', async () => {
    routes([[starts('/messages/conversations/cv1/messages/m1'), json({ data: { id: 'm1', conversation_id: 'cv1', sender_id: 'me', ciphertext: '', nonce: '', created_at: 't', deleted_at: 'x' } })]])
    const msg = await m.deleteMessage(conv(), 'm1')
    expect(msg.deletedAt).toBe('x')
  })

  it('moderateDeleteMessage: ok silencieux, échec → erreur', async () => {
    routes([[starts('/messages/moderation/'), json({}, { ok: true })]])
    await expect(m.moderateDeleteMessage('m1')).resolves.toBeUndefined()
    routes([[starts('/messages/moderation/'), json({}, { ok: false, status: 500 })]])
    await expect(m.moderateDeleteMessage('m1')).rejects.toThrow('moderate delete failed: 500')
  })

  it('sendTyping est best-effort (avale les erreurs)', async () => {
    apiFetch.mockRejectedValue(new Error('net'))
    await expect(m.sendTyping('cv1')).resolves.toBeUndefined()
  })
})

describe('decryptAttachment', () => {
  it('télécharge et déchiffre une pièce jointe en Blob', async () => {
    const att = { id: 'a', nonce: 'n', mime: 'image/png', name: 'x', size: 1, type: 'image' as const, url: '' }
    const blob = await m.decryptAttachment(u8(5), att)
    expect(blob.type).toBe('image/png')
    expect(fetchMediaBytes).toHaveBeenCalledWith('a')
  })
})

// ─── temps réel ───────────────────────────────────────────────────────────────

describe('connectRealtime', () => {
  let sockets: FakeWS[]
  class FakeWS {
    url: string
    onmessage: ((e: { data: string }) => void) | null = null
    onopen: (() => void) | null = null
    onclose: (() => void) | null = null
    closed = false
    constructor(url: string) {
      this.url = url
      sockets.push(this)
    }
    close() {
      this.closed = true
    }
  }
  beforeEach(() => {
    sockets = []
    vi.stubGlobal('WebSocket', FakeWS as unknown as typeof WebSocket)
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('sans token → pas de socket', () => {
    getAccessToken.mockReturnValue('')
    m.connectRealtime(() => {})
    expect(sockets).toHaveLength(0)
  })

  it('route les messages vers onMessage et les autres événements vers onEvent', () => {
    const msgs: unknown[] = []
    const evts: m.RealtimeEvent[] = []
    m.connectRealtime((r) => msgs.push(r), (e) => evts.push(e))
    sockets[0].onopen?.()
    sockets[0].onmessage?.({ data: 'oops' }) // non-JSON ignoré
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'message', data: { id: 'm1' } }) })
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'role_changed', data: { role: 'talker' } }) })
    sockets[0].onmessage?.({ data: JSON.stringify({ type: 'x' }) }) // data absent → null
    expect(msgs).toEqual([{ id: 'm1' }])
    expect(evts).toEqual([
      { type: 'role_changed', data: { role: 'talker' } },
      { type: 'x', data: null },
    ])
  })

  it('reconnecte après coupure et close() bloque la reconnexion', () => {
    const h = m.connectRealtime(() => {})
    sockets[0].onclose?.()
    vi.advanceTimersByTime(1000)
    expect(sockets.length).toBe(2)
    h.close()
    expect(sockets[1].closed).toBe(true)
    sockets[1].onclose?.()
    vi.advanceTimersByTime(10000)
    expect(sockets.length).toBe(2)
  })
})
