'use client'

import { useEffect, useRef, useState } from 'react'
import { Loader2, Users } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { RelationKind } from '@/lib/api'
import { listRelations } from '@/lib/api'
import { useFollow } from '@/lib/use-follow'
import type { RelationUser } from '@/types'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { UserListItem } from '@/components/profil/user-list-item'

interface RelationsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Utilisateur dont on consulte les relations. */
  userId: string
  /** Onglet ouvert au premier affichage. */
  initialTab: RelationKind
  /** Compteurs (libellés des onglets). */
  followersCount: number
  followingCount: number
}

/**
 * Modale des relations d'un profil : deux onglets internes (Abonnés /
 * Abonnements) chargés à la demande depuis le user-service (enrichis du
 * décoratif profil-service). L'état des boutons Suivre/Abonné est géré par
 * `useFollow` (chargé seulement quand la modale est ouverte).
 */
export function RelationsDialog({
  open,
  onOpenChange,
  userId,
  initialTab,
  followersCount,
  followingCount,
}: RelationsDialogProps) {
  const [tab, setTab] = useState<RelationKind>(initialTab)

  // Cache par onglet : `undefined` = pas encore chargé.
  const [lists, setLists] = useState<Partial<Record<RelationKind, RelationUser[]>>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { currentUserId, isFollowing, isPending, toggle } = useFollow(open)

  // Garde-fou contre les fetchs concurrents périmés (changement d'onglet rapide).
  const requestId = useRef(0)

  // À l'ouverture : l'onglet initial dépend du compteur cliqué + reset du cache.
  useEffect(() => {
    if (open) {
      setTab(initialTab)
      setLists({})
      setError(null)
    }
  }, [open, initialTab])

  // Charge la liste de l'onglet actif (une seule fois, mise en cache).
  useEffect(() => {
    if (!open || lists[tab]) return
    const id = ++requestId.current
    setLoading(true)
    setError(null)
    listRelations(userId, tab)
      .then((users) => {
        if (id !== requestId.current) return
        setLists((prev) => ({ ...prev, [tab]: users }))
      })
      .catch(() => {
        if (id !== requestId.current) return
        setError('Impossible de charger la liste.')
      })
      .finally(() => {
        if (id === requestId.current) setLoading(false)
      })
  }, [open, tab, userId, lists])

  const current = lists[tab]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="panel gap-0 overflow-hidden border p-0 shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-w-md">
        <DialogHeader className="border-b p-4">
          <DialogTitle className="brand-text">Connexions</DialogTitle>
        </DialogHeader>

        {/* Onglets */}
        <div className="flex border-b">
          <TabButton active={tab === 'followers'} onClick={() => setTab('followers')}>
            {followersCount} Abonnés
          </TabButton>
          <TabButton active={tab === 'following'} onClick={() => setTab('following')}>
            {followingCount} Abonnements
          </TabButton>
        </div>

        {/* Contenu */}
        <div className="max-h-[60vh] min-h-[16rem] overflow-y-auto">
          {loading || !current ? (
            <CenteredState>
              <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
            </CenteredState>
          ) : error ? (
            <CenteredState>
              <p className="text-sm text-muted-foreground">{error}</p>
            </CenteredState>
          ) : current.length === 0 ? (
            <CenteredState>
              <Users className="h-8 w-8 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
              <p className="text-sm text-muted-foreground">
                {tab === 'followers'
                  ? 'Aucun abonné pour le moment.'
                  : 'Aucun abonnement pour le moment.'}
              </p>
            </CenteredState>
          ) : (
            <div className="divide-y divide-border">
              {current.map((user) => (
                <UserListItem
                  key={user.id}
                  user={user}
                  isFollowing={isFollowing(user.id)}
                  isSelf={user.id === currentUserId}
                  pending={isPending(user.id)}
                  onToggleFollow={toggle}
                />
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'flex-1 py-3 text-sm transition-colors hover:bg-accent',
        active
          ? 'border-b-2 border-[#5B6CFF] font-bold text-[#5B6CFF] dark:border-[#9aa6ff] dark:text-[#9aa6ff]'
          : 'font-normal text-muted-foreground',
      )}
    >
      {children}
    </button>
  )
}

function CenteredState({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[16rem] flex-col items-center justify-center gap-2 px-8 text-center">
      {children}
    </div>
  )
}
