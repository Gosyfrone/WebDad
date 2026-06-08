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
  my_role: string
  my_envelope: string
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

// --- Types front -------------------------------------------------------------

/** Conversation prête à l'emploi : la clé de contenu (CK) est déjà dérivée. */
export interface Conversation {
  id: string
  type: 'dm' | 'group' | 'community'
  memberIds: string[]
  myRole: string
  createdBy: string
  createdAt: string
  updatedAt: string
  /** Clé de contenu déchiffrée ; null si l'enveloppe ne s'ouvre pas ici
   *  (clé créée sur un autre appareil). */
  contentKey: Uint8Array | null
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

/** Dérive la CK d'une conversation depuis l'enveloppe scellée pour soi. */
function toConversation(api: ApiConversation, identity: KeyPair): Conversation {
  let contentKey: Uint8Array | null = null
  try {
    contentKey = api.my_envelope ? openKeyEnvelope(api.my_envelope, identity.privateKey) : null
  } catch {
    contentKey = null // enveloppe créée pour une autre clé (autre appareil)
  }
  return {
    id: api.id,
    type: api.type,
    memberIds: api.member_ids,
    myRole: api.my_role,
    createdBy: api.created_by,
    createdAt: api.created_at,
    updatedAt: api.updated_at,
    contentKey,
  }
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
 * Ouvre la connexion temps réel et appelle `onMessage` pour chaque nouveau
 * message (brut, chiffré — à déchiffrer via `decryptMessage` avec la bonne
 * conversation). Reconnexion automatique simple en cas de coupure.
 */
export function connectRealtime(onMessage: (raw: ApiMessage) => void): RealtimeHandle {
  let socket: WebSocket | null = null
  let closedByUs = false
  let retry = 0

  const connect = () => {
    const token = getAccessToken()
    if (!token) return
    socket = new WebSocket(toWebSocketUrl(API_URL, token))

    socket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data as string) as { type?: string; data?: ApiMessage }
        if (payload.type === 'message' && payload.data) onMessage(payload.data)
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
