/**
 * Client de la messagerie privée (E2EE) via l'API Gateway.
 *
 * Responsabilités :
 *   - publier la clé publique d'identité de l'appareil (registre `user_keys`) ;
 *   - créer / lister des conversations et **dériver localement** leur clé de
 *     contenu (CK) en ouvrant l'enveloppe scellée pour soi ;
 *   - chiffrer à l'envoi / déchiffrer à la lecture (le serveur ne voit que du
 *     chiffré) ;
 *   - recevoir les nouveaux messages en **temps réel** par WebSocket.
 *
 * Tout passe par `apiFetch` (Bearer + refresh single-flight hérités). Les
 * primitives crypto sont dans `crypto.ts`, la clé privée dans `key-store.ts`.
 *
 * ⚠️ Module CLIENT uniquement (apiFetch, WebSocket, IndexedDB).
 */

import { apiFetch, getAccessToken } from '@/lib/auth-client'
import { API_URL } from '@/lib/config'
import {
  decryptText,
  encryptText,
  fromBase64,
  generateContentKey,
  openKeyEnvelope,
  sealKeyForRecipient,
  toBase64,
  type KeyPair,
} from '@/lib/crypto'
import { loadOrCreateIdentity } from '@/lib/key-store'

export class MessageApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'MessageApiError'
    this.status = status
  }
}

// --- Formes brutes (snake_case) de l'API ------------------------------------

interface ApiUserKey {
  user_id: string
  public_key: string
}

interface ApiConversation {
  id: string
  type: 'dm' | 'group' | 'community'
  member_ids: string[]
  title?: string // groupe : nom CHIFFRÉ (ciphertext base64) ; communauté : nom EN CLAIR
  title_nonce?: string
  my_role: string
  my_envelope: string
  content_key?: string // communauté : clé de contenu (base64) remise par le serveur
  pinned_at?: string // épinglage PAR-UTILISATEUR (absent = non épinglée)
  last_read_at?: string // curseur de lecture PAR-UTILISATEUR (absent = jamais lu)
  muted?: boolean // sourdine PAR-UTILISATEUR (exclue du badge, pas de la liste)
  created_by: string
  created_at: string
  updated_at: string
}

interface ApiMember {
  user_id: string
  role: string
}

interface ApiCommunity {
  id: string
  title: string
  member_count: number
  is_member: boolean
  created_by: string
  created_at: string
  updated_at: string
}

interface ApiMessage {
  id: string
  conversation_id: string
  sender_id: string
  ciphertext: string
  nonce: string
  created_at: string
}

/** Message brut tel que poussé par la WebSocket (chiffré). Le `MessagesProvider`
 *  s'en sert pour le badge (métadonnées : conversation + expéditeur, pas besoin
 *  de déchiffrer) et le redistribue à la vue qui, elle, le déchiffre. */
export type RawMessage = ApiMessage

// --- Types front -------------------------------------------------------------

/** Conversation prête à l'emploi : la clé de contenu (CK) est déjà dérivée. */
export interface Conversation {
  id: string
  type: 'dm' | 'group' | 'community'
  memberIds: string[]
  /** Nom DÉCHIFFRÉ (groupes) ; '' pour un DM ou si la clé manque ici. */
  title: string
  myRole: string
  createdBy: string
  createdAt: string
  updatedAt: string
  /** Date d'épinglage (ISO) PAR CET utilisateur ; '' si non épinglée. */
  pinnedAt: string
  /** Curseur de lecture (ISO) PAR CET utilisateur ; '' si jamais lu. Sert à la
   *  pastille « non-lu » et à l'ancre « Nouveaux messages ». */
  lastReadAt: string
  /** Conversation en sourdine PAR CET utilisateur : exclue du badge non-lu
   *  app-wide, mais toujours affichée « non lue » dans la liste. */
  muted: boolean
  /** Clé de contenu déchiffrée ; null si l'enveloppe ne s'ouvre pas ici
   *  (clé créée sur un autre appareil). */
  contentKey: Uint8Array | null
}

/** Membre d'une conversation (sans enveloppe : on ne voit que id + rôle). */
export interface MemberInfo {
  userId: string
  role: string
}

/** Entrée de l'annuaire public des communautés (nom en clair, jamais la clé). */
export interface CommunitySummary {
  id: string
  title: string
  memberCount: number
  isMember: boolean
  createdBy: string
  createdAt: string
  updatedAt: string
}

/** Message déchiffré pour l'affichage. */
export interface ChatMessage {
  id: string
  conversationId: string
  senderId: string
  text: string
  /** false si le déchiffrement a échoué (clé absente sur cet appareil). */
  decrypted: boolean
  createdAt: string
  mine: boolean
}

// --- Enveloppe d'API ---------------------------------------------------------

async function unwrap<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as { data?: T; error?: string } | null
  if (!res.ok) throw new MessageApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  return (body?.data ?? null) as T
}

async function expectOk(res: Response, message: string): Promise<void> {
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new MessageApiError(body?.error ?? message, res.status)
  }
}

// --- Identité de l'utilisateur courant --------------------------------------

let identityPromise: Promise<KeyPair> | null = null

/**
 * Charge (ou crée) l'identité de l'appareil et publie sa clé publique au
 * service. Mémoïsé : un seul aller IndexedDB + une seule publication par session.
 */
export function ensureMyKeys(): Promise<KeyPair> {
  if (!identityPromise) {
    identityPromise = (async () => {
      const identity = await loadOrCreateIdentity()
      await apiFetch('/messages/keys', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ public_key: toBase64(identity.publicKey) }),
      })
      return identity
    })()
  }
  return identityPromise
}

/** Id de l'utilisateur courant, lu dans les claims du JWT (ou '' sans session). */
export function currentUserId(): string {
  const token = getAccessToken()
  if (!token) return ''
  const [, payload] = token.split('.')
  if (!payload) return ''
  try {
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')
    return (JSON.parse(window.atob(padded)) as { user_id?: string }).user_id ?? ''
  } catch {
    return ''
  }
}

// --- Clés publiques ----------------------------------------------------------

/** Récupère la clé publique (base64) d'un utilisateur. 404 → MessageApiError. */
async function fetchPeerPublicKey(userId: string): Promise<string> {
  const key = await unwrap<ApiUserKey>(await apiFetch(`/messages/keys/${encodeURIComponent(userId)}`))
  return key.public_key
}

// --- Conversations -----------------------------------------------------------

/**
 * Construit une Conversation prête à l'emploi (clé + nom).
 *   - DM / groupe : la clé vient de l'enveloppe scellée pour soi (E2EE), le nom
 *     d'un groupe est chiffré avec cette clé.
 *   - Communauté : la clé est **fournie par le serveur** (`content_key`), le nom
 *     est en **clair** (semi-public).
 */
function toConversation(api: ApiConversation, identity: KeyPair): Conversation {
  let contentKey: Uint8Array | null = null
  let title = ''

  if (api.type === 'community') {
    // Communauté : clé remise par le serveur, nom en clair.
    contentKey = api.content_key ? fromBase64(api.content_key) : null
    title = api.title ?? ''
  } else {
    // DM / groupe : clé dérivée de l'enveloppe scellée pour soi (E2EE).
    try {
      contentKey = api.my_envelope ? openKeyEnvelope(api.my_envelope, identity.privateKey) : null
    } catch {
      contentKey = null // enveloppe créée pour une autre clé (autre appareil)
    }
    if (contentKey && api.title && api.title_nonce) {
      try {
        title = decryptText(contentKey, api.title, api.title_nonce)
      } catch {
        title = ''
      }
    }
  }

  return {
    id: api.id,
    type: api.type,
    memberIds: api.member_ids,
    title,
    myRole: api.my_role,
    createdBy: api.created_by,
    createdAt: api.created_at,
    updatedAt: api.updated_at,
    pinnedAt: api.pinned_at ?? '',
    lastReadAt: api.last_read_at ?? '',
    muted: api.muted ?? false,
    contentKey,
  }
}

/** Emballe une clé de contenu pour un utilisateur (récupère sa clé publique). */
async function sealForUser(userId: string, contentKey: Uint8Array): Promise<string> {
  const pub = await fetchPeerPublicKey(userId)
  return sealKeyForRecipient(pub, contentKey)
}

/**
 * Démarre (ou retrouve) un DM avec `peerId` : génère la clé de contenu, l'emballe
 * pour les deux membres (clés publiques) et crée la conversation côté serveur.
 */
export async function startDM(peerId: string): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const myId = currentUserId()

  const [myPubB64, peerPubB64] = [toBase64(identity.publicKey), await fetchPeerPublicKey(peerId)]
  const contentKey = generateContentKey()

  const envelopes: Record<string, string> = {
    [myId]: sealKeyForRecipient(myPubB64, contentKey),
    [peerId]: sealKeyForRecipient(peerPubB64, contentKey),
  }

  const created = await unwrap<ApiConversation>(
    await apiFetch('/messages/conversations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ peer_id: peerId, envelopes }),
    }),
  )
  return toConversation(created, identity)
}

/**
 * Crée un groupe : génère la clé de contenu, chiffre le nom avec, et l'emballe
 * pour chaque membre initial (soi inclus). `memberIds` = les autres membres.
 */
export async function createGroup(name: string, memberIds: string[]): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const myId = currentUserId()
  const contentKey = generateContentKey()

  // Nom chiffré avec la clé de contenu (le serveur ne voit jamais le nom).
  const { ciphertext: title, nonce: title_nonce } = encryptText(contentKey, name)

  // Set de membres (déduit, créateur inclus).
  const others = memberIds.filter((id) => id && id !== myId)
  const envelopes: Record<string, string> = {
    // Mon enveloppe : scellée avec ma clé publique locale (pas d'aller serveur).
    [myId]: sealKeyForRecipient(toBase64(identity.publicKey), contentKey),
  }
  for (const uid of new Set(others)) {
    envelopes[uid] = await sealForUser(uid, contentKey)
  }

  const created = await unwrap<ApiConversation>(
    await apiFetch('/messages/conversations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type: 'group', title, title_nonce, envelopes }),
    }),
  )
  return toConversation(created, identity)
}

/**
 * Invite un utilisateur dans un groupe : emballe la clé de contenu (que l'on
 * détient) pour sa clé publique. N'importe quel membre peut inviter.
 */
export async function inviteToGroup(conv: Conversation, userId: string): Promise<void> {
  if (!conv.contentKey) {
    throw new MessageApiError('clé de conversation indisponible sur cet appareil', 412)
  }
  const envelope = await sealForUser(userId, conv.contentKey)
  await expectOk(
    await apiFetch(`/messages/conversations/${conv.id}/members`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId, envelope }),
    }),
    "Invitation impossible",
  )
}

/** Exclut un membre (owner uniquement côté back). */
export async function removeMember(conv: Conversation, userId: string): Promise<void> {
  await expectOk(
    await apiFetch(`/messages/conversations/${conv.id}/members/${encodeURIComponent(userId)}`, {
      method: 'DELETE',
    }),
    'Exclusion impossible',
  )
}

/** Quitte un groupe (interdit à l'owner côté back : il doit le supprimer). */
export async function leaveGroup(conv: Conversation): Promise<void> {
  await expectOk(
    await apiFetch(`/messages/conversations/${conv.id}/members/me`, { method: 'DELETE' }),
    'Impossible de quitter le groupe',
  )
}

/** Supprime un groupe (owner uniquement côté back). */
export async function deleteGroup(conv: Conversation): Promise<void> {
  await expectOk(
    await apiFetch(`/messages/conversations/${conv.id}`, { method: 'DELETE' }),
    'Suppression impossible',
  )
}

/** Renomme un groupe (owner uniquement) : le nom est re-chiffré avec la CK. */
export async function renameGroup(conv: Conversation, name: string): Promise<Conversation> {
  if (!conv.contentKey) {
    throw new MessageApiError('clé de conversation indisponible sur cet appareil', 412)
  }
  const identity = await ensureMyKeys()
  const { ciphertext: title, nonce: title_nonce } = encryptText(conv.contentKey, name)
  const updated = await unwrap<ApiConversation>(
    await apiFetch(`/messages/conversations/${conv.id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title, title_nonce }),
    }),
  )
  return toConversation(updated, identity)
}

/** Liste les membres d'une conversation (id + rôle). */
export async function listMembers(conversationId: string): Promise<MemberInfo[]> {
  const raw = await unwrap<ApiMember[]>(
    await apiFetch(`/messages/conversations/${conversationId}/members`),
  )
  return (raw ?? []).map((m) => ({ userId: m.user_id, role: m.role }))
}

// --- Communautés (Phase 3, modèle hybride) ----------------------------------

/**
 * Crée une communauté : le client génère la clé de contenu et la **confie au
 * serveur** (`content_key`) pour permettre l'auto-join illimité. Le nom est en
 * CLAIR (semi-public). ⚠️ Conséquence assumée : le serveur peut lire les
 * communautés (contrairement aux DM/groupes).
 */
export async function createCommunity(name: string): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const contentKey = generateContentKey()
  const created = await unwrap<ApiConversation>(
    await apiFetch('/messages/conversations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type: 'community', title: name, content_key: toBase64(contentKey) }),
    }),
  )
  return toConversation(created, identity)
}

/** Annuaire public des communautés (recherche `q` optionnelle). */
export async function listCommunities(q = '', limit = 20, offset = 0): Promise<CommunitySummary[]> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  if (q) params.set('q', q)
  const raw = await unwrap<ApiCommunity[]>(await apiFetch(`/messages/communities?${params.toString()}`))
  return (raw ?? []).map((c) => ({
    id: c.id,
    title: c.title,
    memberCount: c.member_count,
    isMember: c.is_member,
    createdBy: c.created_by,
    createdAt: c.created_at,
    updatedAt: c.updated_at,
  }))
}

/** Rejoint une communauté (en viewer). Le serveur remet la clé de contenu. */
export async function joinCommunity(communityId: string): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const joined = await unwrap<ApiConversation>(
    await apiFetch(`/messages/conversations/${communityId}/join`, { method: 'POST' }),
  )
  return toConversation(joined, identity)
}

/** Promeut/rétrograde un membre d'une communauté (owner only) : talker ↔ viewer. */
export async function setMemberRole(
  conv: Conversation,
  userId: string,
  role: 'talker' | 'viewer',
): Promise<void> {
  await expectOk(
    await apiFetch(`/messages/conversations/${conv.id}/members/${encodeURIComponent(userId)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role }),
    }),
    'Changement de rôle impossible',
  )
}

/** Renomme une communauté (owner only) — nom en CLAIR (pas de chiffrement). */
export async function renameCommunity(conv: Conversation, name: string): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const updated = await unwrap<ApiConversation>(
    await apiFetch(`/messages/conversations/${conv.id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: name }),
    }),
  )
  return toConversation(updated, identity)
}

/** Liste mes conversations (CK dérivées), de la plus active à la plus ancienne. */
export async function listConversations(): Promise<Conversation[]> {
  const identity = await ensureMyKeys()
  const raw = await unwrap<ApiConversation[]>(await apiFetch('/messages/conversations'))
  return (raw ?? []).map((c) => toConversation(c, identity))
}

/** Détail d'une conversation (CK dérivée). */
export async function getConversation(id: string): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const raw = await unwrap<ApiConversation>(await apiFetch(`/messages/conversations/${id}`))
  return toConversation(raw, identity)
}

/** Épingle une conversation en tête de MA liste (état par-utilisateur). */
export async function pinConversation(conv: Conversation): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const updated = await unwrap<ApiConversation>(
    await apiFetch(`/messages/conversations/${conv.id}/pin`, { method: 'PATCH' }),
  )
  return toConversation(updated, identity)
}

/** Retire l'épinglage d'une conversation. */
export async function unpinConversation(conv: Conversation): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const updated = await unwrap<ApiConversation>(
    await apiFetch(`/messages/conversations/${conv.id}/pin`, { method: 'DELETE' }),
  )
  return toConversation(updated, identity)
}

/** Met une conversation en sourdine : elle n'alimente plus le badge non-lu
 *  (mais reste « non lue » dans la liste). */
export async function muteConversation(conv: Conversation): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const updated = await unwrap<ApiConversation>(
    await apiFetch(`/messages/conversations/${conv.id}/mute`, { method: 'PATCH' }),
  )
  return toConversation(updated, identity)
}

/** Réactive une conversation mise en sourdine. */
export async function unmuteConversation(conv: Conversation): Promise<Conversation> {
  const identity = await ensureMyKeys()
  const updated = await unwrap<ApiConversation>(
    await apiFetch(`/messages/conversations/${conv.id}/mute`, { method: 'DELETE' }),
  )
  return toConversation(updated, identity)
}

/**
 * « Supprime » une conversation côté user : la masque de ma liste et coupe mon
 * historique (les autres membres ne sont pas affectés). Elle réapparaît si un
 * nouveau message arrive, sans l'ancien historique.
 */
export async function clearConversation(conv: Conversation): Promise<void> {
  await expectOk(
    await apiFetch(`/messages/conversations/${conv.id}/me`, { method: 'DELETE' }),
    'Suppression impossible',
  )
}

/**
 * Marque une conversation lue côté serveur (avance `last_read_at` à maintenant).
 * Source de vérité unique du non-lu (badge + pastille), multi-appareil.
 */
export async function markConversationRead(conversationId: string): Promise<void> {
  await expectOk(
    await apiFetch(`/messages/conversations/${conversationId}/read`, { method: 'PUT' }),
    'Marquage lu impossible',
  )
}

/** Nombre de conversations ayant au moins un message non lu (badge app-wide). */
export async function getMessagesUnreadCount(): Promise<number> {
  const data = await unwrap<{ count: number }>(await apiFetch('/messages/unread-count'))
  return data?.count ?? 0
}

// --- Messages ----------------------------------------------------------------

/** Déchiffre un message brut avec la CK d'une conversation (échec → texte vide). */
export function decryptMessage(conv: Conversation, api: ApiMessage, myId: string): ChatMessage {
  let text = ''
  let decrypted = false
  if (conv.contentKey) {
    try {
      text = decryptText(conv.contentKey, api.ciphertext, api.nonce)
      decrypted = true
    } catch {
      decrypted = false
    }
  }
  return {
    id: api.id,
    conversationId: api.conversation_id,
    senderId: api.sender_id,
    text,
    decrypted,
    createdAt: api.created_at,
    mine: api.sender_id === myId,
  }
}

/**
 * Historique d'une conversation (du plus ancien au plus récent), déchiffré.
 * `beforeId` (id de message) pagine vers le haut (scroll).
 */
export async function listMessages(
  conv: Conversation,
  limit = 30,
  beforeId?: string,
): Promise<ChatMessage[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (beforeId) params.set('before', beforeId)
  const raw = await unwrap<ApiMessage[]>(
    await apiFetch(`/messages/conversations/${conv.id}/messages?${params.toString()}`),
  )
  const myId = currentUserId()
  return (raw ?? []).map((m) => decryptMessage(conv, m, myId))
}

/** Page de messages prête pour le défilement infini (curseur + « y a-t-il plus ? »). */
export interface MessagePage {
  /** Messages de la page, du plus ancien au plus récent (à préfixer à la liste). */
  messages: ChatMessage[]
  /** Reste-t-il des messages plus anciens à charger ? (false = haut atteint) */
  hasMore: boolean
  /** Curseur à repasser en `beforeId` pour la page d'avant (null si page vide). */
  oldestId: string | null
}

/**
 * Assemble une page paginée à partir des messages chargés. Logique PURE (testée)
 * : `hasMore` = la page est pleine (donc il en reste probablement) ; `oldestId`
 * = le plus ancien message (les messages sont triés croissant), curseur de la
 * page suivante.
 */
export function buildMessagePage(messages: ChatMessage[], limit: number): MessagePage {
  return {
    messages,
    hasMore: messages.length === limit,
    oldestId: messages.length > 0 ? messages[0].id : null,
  }
}

/**
 * Position de la ligne « Nouveaux messages » dans un fil : id du 1ᵉʳ message
 * non-lu qui n'est PAS le mien, à partir de l'ancre `lastReadAt` (curseur de
 * lecture serveur, ISO, capturé à l'ouverture). Fonction PURE (testée).
 *
 *   - `null` si pas d'ancre (jamais lu) ou si rien de nouveau ;
 *   - sinon : 1ᵉʳ message d'autrui dont la date est strictement postérieure à
 *     l'ancre.
 */
export function computeDivider(messages: ChatMessage[], lastReadAt: string | null): string | null {
  if (!lastReadAt) return null
  const anchorMs = new Date(lastReadAt).getTime()
  if (Number.isNaN(anchorMs)) return null
  for (const m of messages) {
    if (!m.mine && new Date(m.createdAt).getTime() > anchorMs) return m.id
  }
  return null
}

/**
 * Charge une page de messages pour le défilement infini : la plus récente sans
 * `beforeId`, sinon les messages ANTÉRIEURS au curseur (scroll vers le haut).
 * Pagination par CURSEUR (et non offset) : stable même quand de nouveaux
 * messages arrivent en temps réel. Brancher `oldestId` → `beforeId` du prochain
 * appel, et s'arrêter quand `hasMore` est false.
 */
export async function listMessagesPage(
  conv: Conversation,
  limit = 30,
  beforeId?: string,
): Promise<MessagePage> {
  return buildMessagePage(await listMessages(conv, limit, beforeId), limit)
}

/** Chiffre et envoie un message ; renvoie le message (déchiffré localement). */
export async function sendMessage(conv: Conversation, text: string): Promise<ChatMessage> {
  if (!conv.contentKey) {
    throw new MessageApiError('clé de conversation indisponible sur cet appareil', 412)
  }
  const { ciphertext, nonce } = encryptText(conv.contentKey, text)
  const created = await unwrap<ApiMessage>(
    await apiFetch(`/messages/conversations/${conv.id}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ciphertext, nonce }),
    }),
  )
  return decryptMessage(conv, created, currentUserId())
}

// --- Recherche dans une conversation (côté client, E2EE) --------------------

/** Taille de page et borne de sécurité pour la récupération de l'historique. */
const HISTORY_PAGE = 50
const MAX_HISTORY_PAGES = 40 // ~2000 messages max (garde-fou anti-boucle/charge)

/**
 * Récupère TOUT l'historique déchiffré d'une conversation (du plus ancien au
 * plus récent), en paginant par curseur. Borné par `MAX_HISTORY_PAGES`.
 *
 * Nécessaire car la recherche est **forcément côté client** : le serveur ne voit
 * que du chiffré (DM/groupes) et n'expose aucune recherche de messages.
 */
export async function listAllMessages(conv: Conversation): Promise<ChatMessage[]> {
  let all: ChatMessage[] = []
  let before: string | undefined
  for (let i = 0; i < MAX_HISTORY_PAGES; i++) {
    const page = await listMessagesPage(conv, HISTORY_PAGE, before)
    all = [...page.messages, ...all] // les pages antérieures se préfixent
    if (!page.hasMore || !page.oldestId) break
    before = page.oldestId
  }
  return all
}

/**
 * Filtre des messages par sous-chaîne (insensible à la casse). Ignore les
 * messages non déchiffrés (clé absente). Fonction PURE (testée). Conserve l'ordre
 * d'entrée (l'appelant décide du sens d'affichage).
 */
export function searchMessages(messages: ChatMessage[], query: string): ChatMessage[] {
  const q = query.trim().toLowerCase()
  if (!q) return []
  return messages.filter((m) => m.decrypted && m.text.toLowerCase().includes(q))
}

// --- Temps réel (WebSocket) --------------------------------------------------

/**
 * Construit l'URL WebSocket vers la gateway. Le token transite en query param
 * (le navigateur n'autorise pas d'en-tête sur un upgrade WS). Fonction PURE
 * (testée). `http(s)://` → `ws(s)://`.
 */
export function toWebSocketUrl(httpBase: string, token: string): string {
  const wsBase = httpBase.replace(/^http/, 'ws').replace(/\/+$/, '')
  return `${wsBase}/messages/ws?access_token=${encodeURIComponent(token)}`
}

/** Poignée de connexion temps réel : permet de fermer proprement. */
export interface RealtimeHandle {
  close(): void
}

/**
 * Événement temps réel NON-message (changement de rôle, renommage, ajout/retrait
 * de membre, suppression de conversation). `data` est un objet libre dont les
 * champs dépendent du type (`conversation_id`/`id`, `user_id`, `role`…).
 */
export interface RealtimeEvent {
  type: string
  data: Record<string, unknown> | null
}

/**
 * Ouvre la connexion temps réel. `onMessage` reçoit chaque nouveau message (brut,
 * chiffré — à déchiffrer via `decryptMessage`). `onEvent` (optionnel) reçoit les
 * autres événements (rôle/membres/conversation) pour resynchroniser la vue —
 * sans lui, la personne promue ne verrait pas son rôle changer avant un reload.
 * Reconnexion automatique simple en cas de coupure.
 */
export function connectRealtime(
  onMessage: (raw: ApiMessage) => void,
  onEvent?: (evt: RealtimeEvent) => void,
): RealtimeHandle {
  let socket: WebSocket | null = null
  let closedByUs = false
  let retry = 0

  const connect = () => {
    const token = getAccessToken()
    if (!token) return
    socket = new WebSocket(toWebSocketUrl(API_URL, token))

    socket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data as string) as { type?: string; data?: unknown }
        if (payload.type === 'message' && payload.data) {
          onMessage(payload.data as ApiMessage)
          return
        }
        if (payload.type) {
          onEvent?.({ type: payload.type, data: (payload.data as Record<string, unknown>) ?? null })
        }
      } catch {
        // message non-JSON : ignoré
      }
    }

    socket.onopen = () => {
      retry = 0
    }

    socket.onclose = () => {
      if (closedByUs) return
      // Reconnexion avec backoff borné (1s → 10s max).
      retry = Math.min(retry + 1, 10)
      setTimeout(connect, retry * 1000)
    }
  }

  connect()

  return {
    close() {
      closedByUs = true
      socket?.close()
    },
  }
}
