'use client'

/**
 * Fournit l'état des notifications à toute l'application authentifiée :
 *   - le compteur de non-lues (badge nav, PC + mobile) ;
 *   - la liste vivante (page /notifications), mise à jour en temps réel ;
 *   - une connexion WebSocket UNIQUE (montée ici, partagée).
 *
 * Le badge est amorcé par `getUnreadCount()` (cheap) au montage, puis maintenu
 * en mémoire : une notification entrante non lue dont l'id est inconnu
 * incrémente le compteur (les activités répétées sur un même groupe — p. ex.
 * 300 likes — partagent le même id et ne comptent donc qu'une fois). Ouvrir la
 * page remet le compteur à zéro (`markAllSeen`).
 */

import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'

import { getAccessToken } from '@/lib/auth-client'
import {
  type AppNotification,
  connectNotifications,
  getUnreadCount,
  listNotifications,
  markAllRead,
} from '@/lib/notifications'

interface NotificationsContextValue {
  unreadCount: number
  items: AppNotification[]
  loading: boolean
  hasMore: boolean
  /** Charge la première page (idempotent) — appelé à l'ouverture de la page. */
  loadInitial: () => void
  /** Charge la page suivante (curseur). */
  loadMore: () => void
  /** Marque tout comme lu (badge → 0) — à l'ouverture de la page. */
  markAllSeen: () => void
}

const NotificationsContext = createContext<NotificationsContextValue | null>(null)

export function useNotifications(): NotificationsContextValue {
  const ctx = useContext(NotificationsContext)
  if (!ctx) {
    throw new Error('useNotifications doit être utilisé dans <NotificationsProvider>')
  }
  return ctx
}

const PAGE_SIZE = 20

export function NotificationsProvider({ children }: { children: React.ReactNode }) {
  const [unreadCount, setUnreadCount] = useState(0)
  const [items, setItems] = useState<AppNotification[]>([])
  const [loading, setLoading] = useState(false)
  const [hasMore, setHasMore] = useState(true)

  // Ids des notifications déjà comptées comme non-lues (dédup des incréments).
  const unreadIds = useRef<Set<string>>(new Set())
  const loadedRef = useRef(false)

  const upsert = useCallback((n: AppNotification) => {
    setItems((prev) => [n, ...prev.filter((p) => p.id !== n.id)])
    if (!n.isRead && !unreadIds.current.has(n.id)) {
      unreadIds.current.add(n.id)
      setUnreadCount((c) => c + 1)
    }
  }, [])

  const removeItem = useCallback((id: string) => {
    setItems((prev) => prev.filter((p) => p.id !== id))
    if (unreadIds.current.has(id)) {
      unreadIds.current.delete(id)
      setUnreadCount((c) => Math.max(0, c - 1))
    }
  }, [])

  const refresh = useCallback(async () => {
    try {
      const [count, page] = await Promise.all([getUnreadCount(), listNotifications('', PAGE_SIZE)])
      setUnreadCount(count)
      unreadIds.current = new Set(page.filter((n) => !n.isRead).map((n) => n.id))
      setItems(page)
      setHasMore(page.length === PAGE_SIZE)
      loadedRef.current = true
    } catch {
      // best-effort : on garde l'état courant
    }
  }, [])

  const loadInitial = useCallback(() => {
    if (loadedRef.current) return
    loadedRef.current = true
    setLoading(true)
    void listNotifications('', PAGE_SIZE)
      .then((page) => {
        setItems(page)
        setHasMore(page.length === PAGE_SIZE)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const loadMore = useCallback(() => {
    if (loading || !hasMore) return
    setLoading(true)
    const last = items[items.length - 1]
    void listNotifications(last?.id ?? '', PAGE_SIZE)
      .then((page) => {
        setItems((prev) => {
          const seen = new Set(prev.map((p) => p.id))
          return [...prev, ...page.filter((p) => !seen.has(p.id))]
        })
        setHasMore(page.length === PAGE_SIZE)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [items, loading, hasMore])

  const markAllSeen = useCallback(() => {
    setUnreadCount(0)
    unreadIds.current.clear()
    setItems((prev) => prev.map((n) => (n.isRead ? n : { ...n, isRead: true })))
    void markAllRead().catch(() => {})
  }, [])

  useEffect(() => {
    if (!getAccessToken()) return
    void getUnreadCount().then(setUnreadCount).catch(() => {})

    const handle = connectNotifications({
      onNotification: upsert,
      onDeleted: removeItem,
      onRefresh: () => void refresh(),
    })
    return () => handle.close()
  }, [upsert, removeItem, refresh])

  return (
    <NotificationsContext.Provider
      value={{ unreadCount, items, loading, hasMore, loadInitial, loadMore, markAllSeen }}
    >
      {children}
    </NotificationsContext.Provider>
  )
}
