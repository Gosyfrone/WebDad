'use client'

/**
 * Fournit l'état de la messagerie à toute l'application authentifiée :
 *   - le compteur de conversations NON LUES (badge nav, PC + mobile) ;
 *   - UNE connexion WebSocket UNIQUE (montée ici, partagée avec la page
 *     `/messages` qui s'y abonne au lieu d'ouvrir la sienne).
 *
 * Le compteur est la **vérité serveur** (`GET /messages/unread-count`, calculée
 * sans déchiffrer le contenu). On le ré-interroge — de façon coalescée — sur les
 * signaux pertinents (nouveau message d'autrui, marquage lu) plutôt que de tenir
 * un compteur fragile en mémoire : à l'échelle d'une messagerie, le débit
 * d'événements est faible et le résultat est toujours exact + multi-appareil.
 *
 * La connexion appartient au provider (cf. décision : « provider possède la WS
 * unique ») : la vue `MessagesView` s'abonne via `subscribeMessages` /
 * `subscribeEvents` et déchiffre elle-même (elle seule a les clés de contenu).
 */

import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'

import { getAccessToken } from '@/lib/auth-client'
import {
  connectRealtime,
  currentUserId,
  getMessagesUnreadCount,
  markConversationRead,
  type RawMessage,
  type RealtimeEvent,
} from '@/lib/messages'
import { playAppSound } from '@/lib/sounds'

interface MessagesContextValue {
  /** Nombre de conversations ayant au moins un message non lu (le badge). */
  unreadCount: number
  /** Déclare la conversation actuellement ouverte (null en quittant la page) :
   *  ses messages entrants sont marqués lus automatiquement. */
  setActiveConversation: (conversationId: string | null) => void
  /** Marque une conversation lue (serveur) puis rafraîchit le compteur. */
  markRead: (conversationId: string) => void
  /** S'abonne aux messages bruts WS (déchiffrement côté abonné). Renvoie un désabonnement. */
  subscribeMessages: (cb: (raw: RawMessage) => void) => () => void
  /** S'abonne aux événements WS non-message. Renvoie un désabonnement. */
  subscribeEvents: (cb: (evt: RealtimeEvent) => void) => () => void
  /** Force un rafraîchissement du compteur (ex. après une action en masse). */
  refresh: () => void
}

const MessagesContext = createContext<MessagesContextValue | null>(null)

export function useMessages(): MessagesContextValue {
  const ctx = useContext(MessagesContext)
  if (!ctx) {
    throw new Error('useMessages doit être utilisé dans <MessagesProvider>')
  }
  return ctx
}

export function MessagesProvider({ children }: { children: React.ReactNode }) {
  const [unreadCount, setUnreadCount] = useState(0)

  const myId = useRef<string>('')
  const activeConvId = useRef<string | null>(null)
  const msgSubs = useRef<Set<(raw: RawMessage) => void>>(new Set())
  const evtSubs = useRef<Set<(evt: RealtimeEvent) => void>>(new Set())
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  /** Ré-interroge le compteur serveur, coalescé (évite une rafale de requêtes). */
  const refresh = useCallback(() => {
    if (refreshTimer.current) return
    refreshTimer.current = setTimeout(() => {
      refreshTimer.current = null
      void getMessagesUnreadCount()
        .then(setUnreadCount)
        .catch(() => {})
    }, 300)
  }, [])

  const markRead = useCallback(
    (conversationId: string) => {
      void markConversationRead(conversationId)
        .then(() => refresh())
        .catch(() => {})
    },
    [refresh],
  )

  const setActiveConversation = useCallback((conversationId: string | null) => {
    activeConvId.current = conversationId
  }, [])

  const subscribeMessages = useCallback((cb: (raw: RawMessage) => void) => {
    msgSubs.current.add(cb)
    return () => {
      msgSubs.current.delete(cb)
    }
  }, [])

  const subscribeEvents = useCallback((cb: (evt: RealtimeEvent) => void) => {
    evtSubs.current.add(cb)
    return () => {
      evtSubs.current.delete(cb)
    }
  }, [])

  useEffect(() => {
    if (!getAccessToken()) return
    myId.current = currentUserId()
    void getMessagesUnreadCount().then(setUnreadCount).catch(() => {})

    const handle = connectRealtime(
      (raw) => {
        // Redistribue à la vue (qui déchiffre) ; le badge n'a besoin que des
        // métadonnées en clair (expéditeur + conversation).
        msgSubs.current.forEach((cb) => cb(raw))
        if (raw.sender_id === myId.current) return
        playAppSound('message_received')
        if (raw.conversation_id === activeConvId.current) {
          // Conversation ouverte → lue à la volée (n'alourdit pas le badge).
          markRead(raw.conversation_id)
        } else {
          refresh()
        }
      },
      (evt) => {
        evtSubs.current.forEach((cb) => cb(evt))
        // Un changement d'appartenance peut modifier le set de conversations.
        refresh()
      },
    )
    return () => {
      handle.close()
      if (refreshTimer.current) clearTimeout(refreshTimer.current)
    }
  }, [markRead, refresh])

  return (
    <MessagesContext.Provider
      value={{
        unreadCount,
        setActiveConversation,
        markRead,
        subscribeMessages,
        subscribeEvents,
        refresh,
      }}
    >
      {children}
    </MessagesContext.Provider>
  )
}
