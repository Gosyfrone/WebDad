/**
 * Client des notifications temps réel via l'API Gateway.
 *
 * Responsabilités :
 *   - lister / paginer les notifications de l'utilisateur (curseur `before`) ;
 *   - compter les non-lues (badge) et marquer comme lu ;
 *   - résoudre l'acteur affiché (dernier auteur) via user + profil-service,
 *     mémoïsé (même approche que `lib/posts.ts`, cache d'auteur) ;
 *   - recevoir les notifications en **temps réel** par WebSocket.
 *
 * Le serveur AGRÈGE : 300 likes d'un post = une seule notification dont `count`
 * vaut 300. L'affichage « X et N autres » se reconstruit depuis l'acteur résolu
 * + `count`.
 *
 * Tout passe par `apiFetch` (Bearer + refresh single-flight hérités).
 *
 * ⚠️ Module CLIENT uniquement (apiFetch, WebSocket).
 */

import { apiFetch, getAccessToken } from '@/lib/auth-client'
import { API_URL } from '@/lib/config'
import { resolveMediaUrl } from '@/lib/media'

export type NotificationType =
  | 'like'
  | 'comment'
  | 'reply'
  | 'mention'
  | 'repost'
  | 'quote'
  | 'message_mention'

// --- Formes brutes (snake_case) de l'API ------------------------------------

interface ApiNotification {
  id: string
  type: NotificationType
  post_id?: string
  comment_id?: string
  conversation_id?: string
  last_actor_id: string
  count: number
  is_read: boolean
  created_at: string
  updated_at: string
}

interface ApiUser {
  id: string
  username: string
}

interface ApiProfil {
  display_name?: string
  avatar_url?: string
}

// --- Types front -------------------------------------------------------------

/** Acteur affiché d'une notification (le dernier en date). */
export interface NotificationActor {
  id: string
  username: string
  displayName: string
  avatarUrl: string
}

/** Notification prête pour l'affichage. */
export interface AppNotification {
  id: string
  type: NotificationType
  /** Post cible (navigation au clic). */
  postId: string
  commentId: string
  /** Conversation cible (mention en message → navigation vers /messages). */
  conversationId: string
  actor: NotificationActor
  /** Nombre TOTAL d'acteurs/événements agrégés (≥ 1). */
  count: number
  /** count - 1 (les « N autres personnes »). */
  othersCount: number
  isRead: boolean
  createdAt: string
  updatedAt: string
}

async function unwrap<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as { data?: T; error?: string } | null
  if (!res.ok) throw new Error(body?.error ?? `Erreur ${res.status}`)
  return (body?.data ?? null) as T
}

// --- Résolution d'acteur (mémoïsée) -----------------------------------------

const actorCache = new Map<string, Promise<NotificationActor>>()

async function fetchUser(userId: string): Promise<ApiUser | null> {
  const res = await apiFetch(`/users/${userId}`)
  if (!res.ok) return null
  const body = (await res.json().catch(() => null)) as { data?: ApiUser } | null
  return body?.data ?? null
}

async function fetchProfil(userId: string): Promise<ApiProfil | null> {
  const res = await apiFetch(`/profils/${userId}`)
  if (!res.ok) return null
  const body = (await res.json().catch(() => null)) as { data?: ApiProfil } | null
  return body?.data ?? null
}

/** Résout (et cache) l'acteur : username + décoratif, repli si absent. */
function resolveActor(userId: string): Promise<NotificationActor> {
  const cached = actorCache.get(userId)
  if (cached) return cached

  const promise = (async (): Promise<NotificationActor> => {
    const [user, profil] = await Promise.all([fetchUser(userId), fetchProfil(userId)])
    return {
      id: userId,
      username: user?.username ?? '',
      displayName: profil?.display_name?.trim() || user?.username || 'Utilisateur',
      avatarUrl: resolveMediaUrl(profil?.avatar_url),
    }
  })()

  actorCache.set(userId, promise)
  return promise
}

// --- Mapping -----------------------------------------------------------------

/** Construit la notification d'affichage (acteur résolu). Logique PURE (testée). */
export function buildNotification(api: ApiNotification, actor: NotificationActor): AppNotification {
  const count = api.count > 0 ? api.count : 1
  return {
    id: api.id,
    type: api.type,
    postId: api.post_id ?? '',
    commentId: api.comment_id ?? '',
    conversationId: api.conversation_id ?? '',
    actor,
    count,
    othersCount: Math.max(0, count - 1),
    isRead: api.is_read,
    createdAt: api.created_at,
    updatedAt: api.updated_at,
  }
}

async function toAppNotification(api: ApiNotification): Promise<AppNotification> {
  return buildNotification(api, await resolveActor(api.last_actor_id))
}

/** Lien de navigation d'une notification : la conversation (mention en message)
 *  ou le post concerné. */
export function notificationHref(n: {
  type: NotificationType
  postId: string
  conversationId: string
}): string {
  if (n.type === 'message_mention') {
    return n.conversationId ? `/messages?conv=${encodeURIComponent(n.conversationId)}` : '/messages'
  }
  return n.postId ? `/posts/${n.postId}` : '/feed'
}

// --- API REST ----------------------------------------------------------------

/** Liste paginée par curseur. `before` = id de la dernière notification chargée. */
export async function listNotifications(before = '', limit = 20): Promise<AppNotification[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (before) params.set('before', before)
  const raw = await unwrap<ApiNotification[]>(await apiFetch(`/notifications?${params.toString()}`))
  return Promise.all((raw ?? []).map(toAppNotification))
}

/** Nombre de notifications non lues (badge). */
export async function getUnreadCount(): Promise<number> {
  const res = await apiFetch('/notifications/unread-count')
  if (!res.ok) return 0
  const data = await unwrap<{ count: number }>(res)
  return data?.count ?? 0
}

/** Marque toutes les notifications comme lues. */
export async function markAllRead(): Promise<void> {
  await apiFetch('/notifications/read', { method: 'POST' })
}

/** Marque une notification comme lue. */
export async function markRead(id: string): Promise<void> {
  await apiFetch(`/notifications/${id}/read`, { method: 'POST' })
}

// --- Temps réel (WebSocket) --------------------------------------------------

/**
 * Construit l'URL WebSocket vers la gateway (token en query param, le navigateur
 * n'autorise pas d'en-tête sur un upgrade WS). Fonction PURE (testée).
 */
export function toWebSocketUrl(httpBase: string, token: string): string {
  const wsBase = httpBase.replace(/^http/, 'ws').replace(/\/+$/, '')
  return `${wsBase}/notifications/ws?access_token=${encodeURIComponent(token)}`
}

/** Poignée de connexion temps réel : permet de fermer proprement. */
export interface RealtimeHandle {
  close(): void
}

/** Callbacks de la connexion temps réel. */
export interface NotificationHandlers {
  /** Nouvelle / mise à jour d'une notification (acteur déjà résolu). */
  onNotification: (n: AppNotification) => void
  /** Une notification a été supprimée (unlike total, post supprimé…). */
  onDeleted: (id: string) => void
  /** Le serveur demande un rechargement complet (purge en cascade). */
  onRefresh: () => void
}

/**
 * Ouvre la connexion temps réel. Résout l'acteur des notifications entrantes
 * avant de les remonter. Reconnexion automatique avec backoff borné.
 */
export function connectNotifications(handlers: NotificationHandlers): RealtimeHandle {
  let socket: WebSocket | null = null
  let closedByUs = false
  let retry = 0

  const connect = () => {
    const token = getAccessToken()
    if (!token) return
    socket = new WebSocket(toWebSocketUrl(API_URL, token))

    socket.onmessage = (event) => {
      let payload: { type?: string; data?: ApiNotification | { id: string } }
      try {
        payload = JSON.parse(event.data as string)
      } catch {
        return
      }
      if (payload.type === 'notification' && payload.data && 'last_actor_id' in payload.data) {
        void toAppNotification(payload.data as ApiNotification).then(handlers.onNotification)
      } else if (payload.type === 'notification_deleted' && payload.data) {
        handlers.onDeleted((payload.data as { id: string }).id)
      } else if (payload.type === 'notification_refresh') {
        handlers.onRefresh()
      }
    }

    socket.onopen = () => {
      retry = 0
    }

    socket.onclose = () => {
      if (closedByUs) return
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
