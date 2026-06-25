'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { MessagesSquare } from 'lucide-react'

import { cn } from '@/lib/utils'
import {
  applyReceipt,
  clearConversation,
  currentUserId,
  decryptMessage,
  ensureMyKeys,
  getConversation,
  getIdentityState,
  type IdentityState,
  joinCommunity,
  listConversations,
  listMessagesPage,
  muteConversation,
  PeerKeyMissingError,
  pinConversation,
  startDM,
  unmuteConversation,
  unpinConversation,
  type ChatMessage,
  type Conversation,
  type RawMessage,
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
import { PassphraseGate } from '@/components/messages/passphrase-gate'

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
    hasMedia: m.media.length > 0,
    deleted: Boolean(m.deletedAt),
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
  // État de l'identité E2EE de l'appareil (null = en cours de détection). Tant
  // qu'il n'est pas `ready`, la vue est floutée derrière la PassphraseGate.
  const [identityState, setIdentityState] = useState<IdentityState | null>(null)
  const [liveMessage, setLiveMessage] = useState<ChatMessage | null>(null)
  const [liveUpdatedMessage, setLiveUpdatedMessage] = useState<ChatMessage | null>(null)
  const [dialog, setDialog] = useState<DialogKind>(null)
  const [previews, setPreviews] = useState<Record<string, ConversationPreview>>({})
  const [unread, setUnread] = useState<Record<string, boolean>>({})
  // « En train d'écrire » (éphémère) : conversation → { userId → expiration ms }.
  // Un ping `typing` (re)pose une échéance à +6 s ; un effet périodique purge.
  const [typing, setTyping] = useState<Record<string, Record<string, number>>>({})
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

  // Purge périodique des échéances « en train d'écrire » expirées (re-render léger).
  useEffect(() => {
    const interval = setInterval(() => {
      setTyping((prev) => {
        const now = Date.now()
        let changed = false
        const next: Record<string, Record<string, number>> = {}
        for (const [cid, users] of Object.entries(prev)) {
          const kept: Record<string, number> = {}
          for (const [uid, until] of Object.entries(users)) {
            if (until > now) kept[uid] = until
            else changed = true
          }
          if (Object.keys(kept).length > 0) next[cid] = kept
        }
        return changed ? next : prev
      })
    }, 1500)
    return () => clearInterval(interval)
  }, [])

  // Membres « en train d'écrire » dans la conversation ouverte (hors moi, non expirés).
  const typingUserIds = useMemo(() => {
    if (!selectedId) return [] as string[]
    const now = Date.now()
    return Object.entries(typing[selectedId] ?? {})
      .filter(([, until]) => until > now)
      .map(([uid]) => uid)
  }, [typing, selectedId])

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

  // Chargement initial : détermine d'abord l'état de l'identité E2EE. Si une clé
  // est présente sur cet appareil (`ready`), on publie la clé publique puis on
  // liste les conversations. Sinon (setup/unlock), on laisse la PassphraseGate
  // prendre le relais — pas de chargement tant que l'identité n'est pas dispo.
  useEffect(() => {
    let cancelled = false
    getIdentityState()
      .then(async (state) => {
        if (cancelled) return
        setIdentityState(state)
        if (state === 'ready') {
          await ensureMyKeys()
          await loadConversations()
        }
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [loadConversations])

  // Appelé par la PassphraseGate après définition/déblocage réussi : l'identité
  // est désormais disponible localement → on charge la messagerie.
  const handleUnlocked = useCallback(() => {
    setIdentityState('ready')
    setLoading(true)
    ensureMyKeys()
      .then(() => loadConversations())
      .catch(() => {})
      .finally(() => setLoading(false))
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
        case 'message_updated': {
          const raw = data as unknown as RawMessage
          const conv = conversationsRef.current.find((c) => c.id === raw.conversation_id)
          if (!conv) return
          const msg = decryptMessage(conv, raw, myId)
          setLiveUpdatedMessage(msg)
          setPreviews((prev) =>
            prev[conv.id]?.messageId === msg.id
              ? { ...prev, [conv.id]: toPreview(msg, myUsernameRef.current) }
              : prev,
          )
          break
        }
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
        case 'receipt': {
          // Accusé « remis »/« ouvert » d'un autre membre → met à jour les coches
          // de mes messages dans cette conversation, en temps réel.
          const uid = typeof data.user_id === 'string' ? data.user_id : ''
          if (!uid || uid === myId) break
          const deliveredAt = typeof data.delivered_at === 'string' ? data.delivered_at : ''
          const readAt = typeof data.read_at === 'string' ? data.read_at : ''
          setConversations((prev) =>
            prev.map((c) => (c.id === cid ? applyReceipt(c, uid, deliveredAt, readAt) : c)),
          )
          break
        }
        case 'typing': {
          // Signal éphémère « en train d'écrire » : on (re)pose une échéance à +6 s
          // pour ce membre ; l'effet de purge l'efface à expiration.
          const uid = typeof data.user_id === 'string' ? data.user_id : ''
          if (!uid || uid === myId) break
          setTyping((prev) => ({ ...prev, [cid]: { ...(prev[cid] ?? {}), [uid]: Date.now() + 6000 } }))
          break
        }
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
    // Attend que l'identité soit disponible (déblocage passé) avant d'ouvrir.
    if (!dmTarget || identityState !== 'ready' || handledDmRef.current === dmTarget) return
    handledDmRef.current = dmTarget
    // Nettoie l'URL via le routeur Next (PAS window.history : l'API brute écrase
    // l'état interne de Next et casse la navigation arrière).
    const cleanUrl = () => router.replace('/messages', { scroll: false })
    // Conversation déjà ouverte avec cette personne → on l'ouvre directement (pas
    // besoin de la clé du destinataire pour rejoindre un DM existant).
    const existing = conversationsRef.current.find(
      (c) => c.type === 'dm' && c.memberIds.includes(dmTarget),
    )
    if (existing) {
      upsertAndSelect(existing)
      cleanUrl()
      return
    }
    ensureMyKeys()
      .then(() => startDM(dmTarget))
      .then((conv) => {
        upsertAndSelect(conv)
        cleanUrl()
      })
      .catch((err) => {
        // Le destinataire n'a pas activé sa messagerie → message dédié (pas rouge).
        if (err instanceof PeerKeyMissingError) {
          toast({ title: t('messages.peer_not_activated'), variant: 'brand' })
        } else {
          toast({ title: t('messages.dm_failed'), variant: 'destructive' })
        }
        cleanUrl()
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dmTarget, identityState])

  // Point d'entrée depuis une notification de mention : /messages?conv=<id> →
  // ouvre la conversation existante.
  const convTarget = searchParams.get('conv')
  const handledConvRef = useRef<string | null>(null)
  useEffect(() => {
    if (!convTarget || identityState !== 'ready' || handledConvRef.current === convTarget) return
    handledConvRef.current = convTarget
    ensureMyKeys()
      .then(() => getConversation(convTarget))
      .then((conv) => {
        upsertAndSelect(conv)
        router.replace('/messages', { scroll: false })
      })
      .catch(() => {})
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [convTarget, identityState])

  // Lien d'invitation d'une communauté : /messages?join=<id> → auto-join (le
  // serveur remet la clé) puis ouverture. Un visiteur est déjà redirigé vers
  // /login par le middleware (route protégée), puis revient ici après connexion.
  const joinTarget = searchParams.get('join')
  const handledJoinRef = useRef<string | null>(null)
  useEffect(() => {
    if (!joinTarget || identityState !== 'ready' || handledJoinRef.current === joinTarget) return
    handledJoinRef.current = joinTarget
    ensureMyKeys()
      .then(() => joinCommunity(joinTarget))
      .then((conv) => {
        upsertAndSelect(conv)
        router.replace('/messages', { scroll: false })
      })
      .catch(() => {
        toast({ title: t('messages.join_failed'), variant: 'destructive' })
        router.replace('/messages', { scroll: false })
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [joinTarget, identityState])

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

  const gated = identityState !== null && identityState !== 'ready'

  // Floute/inerte les volets tant que l'identité E2EE n'est pas disponible : la
  // PassphraseGate (positionnée absolue, hors flou) prend alors le relais.
  const blurWhenGated = gated && 'pointer-events-none select-none blur-md'

  return (
    <div className="relative flex min-h-0 flex-1 overflow-hidden">
      {/* Volet liste */}
      <div
        className={cn(
          // Pas de pt-14 ici : l'overlay « wide » démarre déjà à `top-14`
          // (cf. FeedOverlay) → le header global est déjà dégagé. Un pt-14
          // ajouterait un 2ᵉ décalage de 56px (vide sous le header).
          'h-full w-full shrink-0 lg:w-[360px] lg:border-r',
          selectedId ? 'hidden lg:flex lg:flex-col' : 'flex flex-col',
          blurWhenGated,
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

      {/* Volet chat (mobile : occupe la vue dès le haut — l'en-tête global est
          masqué sur une conversation ouverte, le ChatPane porte le sien). */}
      <div
        className={cn(
          'h-full min-w-0 flex-1',
          selectedId ? 'flex flex-col' : 'hidden lg:flex lg:flex-col',
          blurWhenGated,
        )}
      >
        {selected ? (
          <ChatPane
            key={selected.id}
            conversation={selected}
            myId={myId}
            liveMessage={liveMessage}
            liveUpdatedMessage={liveUpdatedMessage}
            typingUserIds={typingUserIds}
            dividerAnchor={anchorForSelected}
            onBack={() => setSelectedId(null)}
            onOpenInfo={() => setDialog('info')}
            onLocalMessage={(msg, edited) => {
              if (!edited) {
                ingest(selected.id, msg, true)
                return
              }
              setPreviews((prev) =>
                prev[selected.id]?.messageId === msg.id
                  ? { ...prev, [selected.id]: toPreview(msg, myUsernameRef.current) }
                  : prev,
              )
            }}
          />
        ) : (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-8 text-center">
            <MessagesSquare className="h-12 w-12 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
            <p className="text-lg font-bold text-foreground">{t('messages.select_title')}</p>
            <p className="max-w-sm text-sm text-muted-foreground">{t('messages.select_desc')}</p>
          </div>
        )}
      </div>

      {/* Protection par phrase de passe (définir / débloquer) — par-dessus le
          contenu flouté tant que l'identité E2EE n'est pas disponible ici. */}
      {gated && identityState && (
        <PassphraseGate
          mode={identityState === 'unlock' ? 'unlock' : 'setup'}
          onUnlocked={handleUnlocked}
        />
      )}

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
