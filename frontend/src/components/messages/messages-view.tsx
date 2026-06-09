'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { MessagesSquare } from 'lucide-react'

import { cn } from '@/lib/utils'
import {
  clearConversation,
  currentUserId,
  decryptMessage,
  ensureMyKeys,
  getConversation,
  listConversations,
  listMessagesPage,
  muteConversation,
  pinConversation,
  startDM,
  unmuteConversation,
  unpinConversation,
  type ChatMessage,
  type Conversation,
  type RealtimeEvent,
} from '@/lib/messages'
import { textMentionsUser } from '@/lib/mentions'
import { resolveUser } from '@/lib/user-cache'
import { useToast } from '@/hooks/use-toast'
import { useMessages } from '@/components/messages-provider'
import { useLanguage } from '@/components/language-provider'
import { ChatPane } from '@/components/messages/chat-pane'
import { ConversationInfoDialog } from '@/components/messages/conversation-info-dialog'
import { ConversationList, type ConversationPreview } from '@/components/messages/conversation-list'
import { CreateCommunityDialog } from '@/components/messages/create-community-dialog'
import { DiscoverCommunitiesDialog } from '@/components/messages/discover-communities-dialog'
import { NewDMDialog } from '@/components/messages/new-dm-dialog'
import { NewGroupDialog } from '@/components/messages/new-group-dialog'

type DialogKind = 'dm' | 'group' | 'community' | 'discover' | 'info' | null

/**
 * Trie les conversations « épinglées d'abord » (par date d'épinglage
 * décroissante) puis par activité décroissante — même ordre que le serveur, à
 * réappliquer côté client après chaque mutation locale (nouveau message, pin…).
 */
function sortConversations(list: Conversation[]): Conversation[] {
  return [...list].sort((a, b) => {
    const ap = a.pinnedAt !== ''
    const bp = b.pinnedAt !== ''
    if (ap !== bp) return ap ? -1 : 1
    if (ap && bp && a.pinnedAt !== b.pinnedAt) return a.pinnedAt > b.pinnedAt ? -1 : 1
    return a.updatedAt > b.updatedAt ? -1 : 1
  })
}

/** Met à jour la date d'activité d'une conversation puis re-trie (épinglés d'abord). */
function touchAndSort(list: Conversation[], id: string, updatedAt: string): Conversation[] {
  return sortConversations(list.map((c) => (c.id === id ? { ...c, updatedAt } : c)))
}

function toPreview(m: ChatMessage, myUsername: string): ConversationPreview {
  return {
    messageId: m.id,
    text: m.text,
    mine: m.mine,
    decrypted: m.decrypted,
    senderId: m.senderId,
    // « X vous a mentionné » : dernier message d'autrui, déchiffré, citant mon handle.
    mentionsMe: !m.mine && m.decrypted && textMentionsUser(m.text, myUsername),
  }
}

/**
 * Une conversation est « non lue » si son dernier message n'est pas le mien et
 * est postérieur à mon curseur de lecture serveur (`lastReadAt`, '' = jamais lu).
 */
function isConvUnread(lastReadAt: string, last: ChatMessage | null): boolean {
  if (!last || last.mine) return false
  if (!lastReadAt) return true
  return new Date(last.createdAt).getTime() > new Date(lastReadAt).getTime()
}

/**
 * Orchestrateur de la messagerie : charge les conversations (DM / groupes /
 * communautés) + l'aperçu de leur dernier message, s'abonne à la WebSocket
 * UNIQUE du `MessagesProvider` (déchiffrement + routage des messages entrants,
 * aperçus, pastille non-lue, remontée en tête), et pilote les modales.
 *
 * L'état « lu » est porté par le serveur (`lastReadAt` par conversation,
 * multi-appareil) : pastille non-lue dans la liste et ancre de la ligne
 * « Nouveaux messages » capturée à l'ouverture. Le badge app-wide vit dans le
 * provider ; ouvrir une conversation la marque lue (`markRead`).
 */
export function MessagesView() {
  const { t } = useLanguage()
  const { toast } = useToast()
  const { markRead, setActiveConversation, subscribeMessages, subscribeEvents, refresh } =
    useMessages()
  const myId = useMemo(() => currentUserId(), [])
  // Mon handle, résolu une fois : sert à détecter « X vous a mentionné » dans
  // l'aperçu (le dernier message est déjà déchiffré côté destinataire → E2EE).
  const myUsernameRef = useRef('')
  useEffect(() => {
    resolveUser(myId)
      .then((u) => {
        myUsernameRef.current = u.username
      })
      .catch(() => {})
  }, [myId])

  const [conversations, setConversations] = useState<Conversation[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [liveMessage, setLiveMessage] = useState<ChatMessage | null>(null)
  const [dialog, setDialog] = useState<DialogKind>(null)
  const [previews, setPreviews] = useState<Record<string, ConversationPreview>>({})
  const [unread, setUnread] = useState<Record<string, boolean>>({})
  // Ancre de la ligne « Nouveaux messages » (dernier lu CAPTURÉ à l'ouverture).
  const [dividerAnchor, setDividerAnchor] = useState<{ convId: string; anchor: string | null }>({
    convId: '',
    anchor: null,
  })

  const conversationsRef = useRef<Conversation[]>([])
  conversationsRef.current = conversations
  const selectedIdRef = useRef<string | null>(null)
  selectedIdRef.current = selectedId

  const selected = conversations.find((c) => c.id === selectedId) ?? null

  /** Intègre un message (WS, envoi local, ou aperçu) : aperçu + ordre + pastille.
   *  Le marquage « lu » serveur d'une conv ouverte est géré par le provider. */
  const ingest = useCallback((conversationId: string, msg: ChatMessage, seen: boolean) => {
    setPreviews((p) => ({ ...p, [conversationId]: toPreview(msg, myUsernameRef.current) }))
    setConversations((prev) => touchAndSort(prev, conversationId, msg.createdAt))
    if (seen || msg.mine) {
      setUnread((u) => (u[conversationId] ? { ...u, [conversationId]: false } : u))
    } else {
      setUnread((u) => ({ ...u, [conversationId]: true }))
    }
  }, [])

  /** Charge les conversations + l'aperçu (dernier message) de chacune. */
  const loadConversations = useCallback(async () => {
    const list = await listConversations()
    // Préserve une conversation sélectionnée absente de l'instantané (ex. un DM
    // ouvert via ?dm=… et créé à l'instant, pas encore visible par ce GET) →
    // évite que ce chargement écrase la sélection et renvoie vers la liste.
    setConversations((prev) => {
      const sel = selectedIdRef.current
      const keep = sel && !list.some((c) => c.id === sel) ? prev.find((c) => c.id === sel) : null
      return sortConversations(keep ? [keep, ...list] : list)
    })
    const entries = await Promise.all(
      list.map(async (c) => {
        try {
          const page = await listMessagesPage(c, 1)
          return [c.id, page.messages[page.messages.length - 1] ?? null] as const
        } catch {
          return [c.id, null] as const
        }
      }),
    )
    setPreviews((prev) => {
      const next = { ...prev }
      for (const [id, last] of entries) if (last) next[id] = toPreview(last, myUsernameRef.current)
      return next
    })
    const readById = new Map(list.map((c) => [c.id, c.lastReadAt]))
    setUnread((prev) => {
      const next = { ...prev }
      for (const [id, last] of entries) {
        next[id] = isConvUnread(readById.get(id) ?? '', last)
      }
      return next
    })
  }, [])

  // Chargement initial (publie ma clé publique puis liste + aperçus).
  useEffect(() => {
    let cancelled = false
    ensureMyKeys()
      .then(() => loadConversations())
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [loadConversations])

  /**
   * Événements temps réel non-message : resynchronise la vue. Crucial pour le
   * changement de rôle (la personne promue talker doit voir son composer
   * s'activer sans recharger) ; gère aussi renommage / ajout / exclusion /
   * suppression.
   */
  const handleEvent = useCallback(
    (evt: RealtimeEvent) => {
      const data = evt.data ?? {}
      const cid =
        (typeof data.conversation_id === 'string' && data.conversation_id) ||
        (typeof data.id === 'string' && data.id) ||
        ''
      if (!cid) return

      const dropIt = () => {
        setConversations((prev) => prev.filter((c) => c.id !== cid))
        setSelectedId((cur) => (cur === cid ? null : cur))
      }
      // Recharge MA vue de la conversation (mon rôle, le titre…) et la remet à jour.
      const refetch = () => {
        getConversation(cid)
          .then((conv) =>
            setConversations((prev) =>
              sortConversations(prev.map((c) => (c.id === conv.id ? conv : c))),
            ),
          )
          .catch(() => {})
      }

      switch (evt.type) {
        case 'conversation_deleted':
          dropIt()
          break
        case 'member_removed':
          if (data.user_id === myId) dropIt()
          else refetch()
          break
        case 'member_added':
          // Je viens d'être ajouté → la conversation n'est pas encore dans ma liste.
          if (data.user_id === myId) loadConversations().catch(() => {})
          else refetch()
          break
        case 'member_role_changed':
        case 'conversation_updated':
          refetch()
          break
      }
    },
    [myId, loadConversations],
  )

  // Temps réel : on s'abonne à la WebSocket UNIQUE du provider (qui la possède).
  useEffect(() => {
    const unsubMsg = subscribeMessages((raw) => {
      const conv = conversationsRef.current.find((c) => c.id === raw.conversation_id)
      if (!conv) {
        // Conversation inconnue (quelqu'un vient de m'écrire) → on recharge tout.
        loadConversations().catch(() => {})
        return
      }
      const msg = decryptMessage(conv, raw, myId)
      setLiveMessage(msg)
      ingest(conv.id, msg, selectedIdRef.current === conv.id)
    })
    const unsubEvt = subscribeEvents(handleEvent)
    return () => {
      unsubMsg()
      unsubEvt()
    }
  }, [myId, ingest, loadConversations, handleEvent, subscribeMessages, subscribeEvents])

  // Tient le provider informé de la conversation ouverte (pour qu'il marque lus
  // les messages entrants de la conv active et n'enfle pas le badge).
  useEffect(() => {
    setActiveConversation(selectedId)
    return () => setActiveConversation(null)
  }, [selectedId, setActiveConversation])

  // Point d'entrée depuis un profil : /messages?dm=<userId> → ouvre le DM.
  const router = useRouter()
  const searchParams = useSearchParams()
  const dmTarget = searchParams.get('dm')
  const handledDmRef = useRef<string | null>(null)
  useEffect(() => {
    if (!dmTarget || handledDmRef.current === dmTarget) return
    handledDmRef.current = dmTarget
    ensureMyKeys()
      .then(() => startDM(dmTarget))
      .then((conv) => {
        // Sélectionne la conversation PUIS nettoie l'URL via le routeur Next
        // (router.replace, PAS window.history : l'API History brute écrase l'état
        // interne de Next et casse la navigation arrière depuis la recherche).
        upsertAndSelect(conv)
        router.replace('/messages', { scroll: false })
      })
      .catch(() => {})
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dmTarget])

  // Point d'entrée depuis une notification de mention : /messages?conv=<id> →
  // ouvre la conversation existante.
  const convTarget = searchParams.get('conv')
  const handledConvRef = useRef<string | null>(null)
  useEffect(() => {
    if (!convTarget || handledConvRef.current === convTarget) return
    handledConvRef.current = convTarget
    ensureMyKeys()
      .then(() => getConversation(convTarget))
      .then((conv) => {
        upsertAndSelect(conv)
        router.replace('/messages', { scroll: false })
      })
      .catch(() => {})
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [convTarget])

  /** Marque une conversation lue : serveur (`markRead`) + état local optimiste
   *  (curseur `lastReadAt` avancé + pastille effacée), après capture de l'ancre. */
  function markConvRead(id: string) {
    markRead(id)
    const now = new Date().toISOString()
    setConversations((prev) => prev.map((c) => (c.id === id ? { ...c, lastReadAt: now } : c)))
    setUnread((u) => (u[id] ? { ...u, [id]: false } : u))
  }

  /** Sélectionne une conversation : capture l'ancre du séparateur PUIS marque lu. */
  function selectConversation(id: string) {
    const conv = conversationsRef.current.find((c) => c.id === id)
    setDividerAnchor({ convId: id, anchor: conv?.lastReadAt || null })
    setSelectedId(id)
    markConvRead(id)
  }

  function upsertAndSelect(conv: Conversation) {
    setConversations((prev) => sortConversations([conv, ...prev.filter((c) => c.id !== conv.id)]))
    setDividerAnchor({ convId: conv.id, anchor: conv.lastReadAt || null })
    setSelectedId(conv.id)
    markConvRead(conv.id)
    setDialog(null)
  }

  function updateConv(conv: Conversation) {
    setConversations((prev) => sortConversations(prev.map((c) => (c.id === conv.id ? conv : c))))
  }

  function removeConv(id: string) {
    setConversations((prev) => prev.filter((c) => c.id !== id))
    setSelectedId((cur) => (cur === id ? null : cur))
  }

  /** (Dés)épingle une conversation — optimiste, réconcilié avec la réponse serveur. */
  async function togglePin(conv: Conversation) {
    const pin = conv.pinnedAt === ''
    const optimistic = pin ? new Date().toISOString() : ''
    setConversations((prev) =>
      sortConversations(prev.map((c) => (c.id === conv.id ? { ...c, pinnedAt: optimistic } : c))),
    )
    try {
      const updated = pin ? await pinConversation(conv) : await unpinConversation(conv)
      updateConv(updated)
    } catch {
      toast({ title: t('messages.action_failed'), variant: 'destructive' })
      loadConversations().catch(() => {})
    }
  }

  /** (Dé)met en sourdine — optimiste. Le badge app-wide est ré-interrogé via le
   *  provider (`refresh`) car le serveur exclut les sourdines du non-lu. */
  async function toggleMute(conv: Conversation) {
    const mute = !conv.muted
    setConversations((prev) => prev.map((c) => (c.id === conv.id ? { ...c, muted: mute } : c)))
    try {
      const updated = mute ? await muteConversation(conv) : await unmuteConversation(conv)
      updateConv(updated)
      refresh()
    } catch {
      toast({ title: t('messages.action_failed'), variant: 'destructive' })
      loadConversations().catch(() => {})
    }
  }

  /** « Supprime » la conversation côté user (masque + coupe l'historique). */
  async function deleteConversation(conv: Conversation) {
    try {
      await clearConversation(conv)
      removeConv(conv.id)
    } catch {
      toast({ title: t('messages.delete_failed'), variant: 'destructive' })
    }
  }

  const anchorForSelected =
    selected && dividerAnchor.convId === selected.id ? dividerAnchor.anchor : null

  return (
    <div className="flex h-[calc(100dvh-7.5rem)] overflow-hidden lg:h-screen">
      {/* Volet liste */}
      <div
        className={cn(
          'h-full w-full shrink-0 lg:w-[360px] lg:border-r',
          selectedId ? 'hidden lg:flex lg:flex-col' : 'flex flex-col',
        )}
      >
        <ConversationList
          conversations={conversations}
          selectedId={selectedId}
          myId={myId}
          loading={loading}
          previews={previews}
          unread={unread}
          onSelect={selectConversation}
          onTogglePin={togglePin}
          onToggleMute={toggleMute}
          onDelete={deleteConversation}
          onNewDM={() => setDialog('dm')}
          onNewGroup={() => setDialog('group')}
          onNewCommunity={() => setDialog('community')}
          onDiscover={() => setDialog('discover')}
        />
      </div>

      {/* Volet chat */}
      <div
        className={cn(
          'h-full min-w-0 flex-1',
          selectedId ? 'flex flex-col' : 'hidden lg:flex lg:flex-col',
        )}
      >
        {selected ? (
          <ChatPane
            key={selected.id}
            conversation={selected}
            myId={myId}
            liveMessage={liveMessage}
            dividerAnchor={anchorForSelected}
            onBack={() => setSelectedId(null)}
            onOpenInfo={() => setDialog('info')}
            onLocalMessage={(msg) => ingest(selected.id, msg, true)}
          />
        ) : (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-8 text-center">
            <MessagesSquare className="h-12 w-12 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
            <p className="text-lg font-bold text-foreground">{t('messages.select_title')}</p>
            <p className="max-w-sm text-sm text-muted-foreground">{t('messages.select_desc')}</p>
          </div>
        )}
      </div>

      {/* Modales */}
      <NewDMDialog
        open={dialog === 'dm'}
        onOpenChange={(o) => setDialog(o ? 'dm' : null)}
        myId={myId}
        onCreated={upsertAndSelect}
      />
      <NewGroupDialog
        open={dialog === 'group'}
        onOpenChange={(o) => setDialog(o ? 'group' : null)}
        myId={myId}
        onCreated={upsertAndSelect}
      />
      <CreateCommunityDialog
        open={dialog === 'community'}
        onOpenChange={(o) => setDialog(o ? 'community' : null)}
        onCreated={upsertAndSelect}
      />
      <DiscoverCommunitiesDialog
        open={dialog === 'discover'}
        onOpenChange={(o) => setDialog(o ? 'discover' : null)}
        onJoined={upsertAndSelect}
      />
      {selected && (
        <ConversationInfoDialog
          open={dialog === 'info'}
          onOpenChange={(o) => setDialog(o ? 'info' : null)}
          conversation={selected}
          myId={myId}
          onUpdated={updateConv}
          onRemoved={removeConv}
        />
      )}
    </div>
  )
}
