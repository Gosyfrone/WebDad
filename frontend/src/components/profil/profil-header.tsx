'use client'

import { useState } from 'react'
import { CalendarDays } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { RelationKind } from '@/lib/api'
import type { ProfilDetails, ProfilEditableFields } from '@/types'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EditProfilDialog } from '@/components/profil/edit-profil-dialog'
import { RelationsDialog } from '@/components/profil/relations-dialog'

interface ProfilHeaderProps {
  profil: ProfilDetails
  /** Vrai si le profil affiché est celui de l'utilisateur courant. */
  isOwner: boolean
  /** Remontée des champs édités (consommée par le parent pour l'affichage live). */
  onEdit: (fields: ProfilEditableFields) => void
}

/** Libellé lisible pour chaque rôle. */
const ROLE_LABELS: Record<ProfilDetails['role'], string> = {
  user: 'Utilisateur',
  moderator: 'Modérateur',
  administrator: 'Administrateur',
}

/**
 * En-tête de la page profil : bannière, avatar superposé, identité, bio,
 * date d'inscription et compteurs d'abonnés. Le propriétaire voit le bouton
 * « Éditer le profil » ; un visiteur verrait « Suivre » (à brancher).
 */
export function ProfilHeader({ profil, isOwner, onEdit }: ProfilHeaderProps) {
  const initials = profil.displayName.charAt(0).toUpperCase()
  const [relationsOpen, setRelationsOpen] = useState(false)
  const [relationsTab, setRelationsTab] = useState<RelationKind>('followers')

  function openRelations(tab: RelationKind) {
    setRelationsTab(tab)
    setRelationsOpen(true)
  }

  return (
    <header className="panel">
      {/* Bannière */}
      <div
        className={cn(
          'h-36 w-full bg-cover bg-center sm:h-48',
          !profil.bannerUrl &&
            'bg-gradient-to-r from-[#8D3DFF]/35 via-[#EADCFF] to-[#47D9FF]/25 dark:from-[#8D3DFF]/45 dark:via-[#1c1338] dark:to-[#47D9FF]/35',
        )}
        style={profil.bannerUrl ? { backgroundImage: `url(${profil.bannerUrl})` } : undefined}
      />

      <div className="px-4 pb-3">
        {/* Avatar superposé + action */}
        <div className="flex items-end justify-between">
          <Avatar className="-mt-12 h-24 w-24 border-4 border-white shadow-[0_18px_44px_rgba(91,108,255,0.26)] dark:border-[#140c24] sm:-mt-16 sm:h-32 sm:w-32">
            {profil.avatarUrl && (
              <AvatarImage src={profil.avatarUrl} alt={profil.displayName} />
            )}
            <AvatarFallback className="text-3xl">{initials}</AvatarFallback>
          </Avatar>

          <div className="mt-3">
            {isOwner ? (
              <EditProfilDialog
                initial={{
                  displayName: profil.displayName,
                  bio: profil.bio,
                  avatarUrl: profil.avatarUrl,
                  bannerUrl: profil.bannerUrl,
                }}
                onSave={onEdit}
              >
                <Button
                  variant="outline"
                  className="rounded-full border-white/70 bg-white/80 font-bold shadow-sm backdrop-blur hover:bg-white dark:border-white/15 dark:bg-white/10 dark:hover:bg-white/20"
                >
                  Éditer le profil
                </Button>
              </EditProfilDialog>
            ) : (
              // TODO (issue profil) : brancher l'action « suivre »
              <Button
                className="rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white"
                disabled
              >
                Suivre
              </Button>
            )}
          </div>
        </div>

        {/* Identité */}
        <div className="mt-3 flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-extrabold text-foreground">{profil.displayName}</h1>
            <Badge variant="secondary">{ROLE_LABELS[profil.role]}</Badge>
          </div>
          <span className="text-sm text-muted-foreground">@{profil.username}</span>
        </div>

        {/* Bio */}
        {profil.bio && (
          <p className="mt-3 whitespace-pre-wrap text-sm leading-relaxed">{profil.bio}</p>
        )}

        {/* Date d'inscription */}
        <div className="mt-3 flex items-center gap-1.5 text-sm text-muted-foreground">
          <CalendarDays className="h-4 w-4" aria-hidden />
          <span>A rejoint en {formatJoinedAt(profil.joinedAt)}</span>
        </div>

        {/* Compteurs (cliquables → modale des relations) — même ordre que la modale */}
        <div className="mt-3 flex gap-5 text-sm">
          <Count
            value={profil.followersCount}
            label="Abonnés"
            onClick={() => openRelations('followers')}
          />
          <Count
            value={profil.followingCount}
            label="Abonnements"
            onClick={() => openRelations('following')}
          />
        </div>
      </div>

      <RelationsDialog
        open={relationsOpen}
        onOpenChange={setRelationsOpen}
        userId={profil.userId}
        initialTab={relationsTab}
        followersCount={profil.followersCount}
        followingCount={profil.followingCount}
      />
    </header>
  )
}

function Count({
  value,
  label,
  onClick,
}: {
  value: number
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex gap-1 rounded-md transition-colors hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#5B6CFF]/40"
    >
      <span className="font-bold text-foreground">{formatCount(value)}</span>
      <span className="text-muted-foreground">{label}</span>
    </button>
  )
}

/** « juin 2026 » à partir d'une date ISO. */
function formatJoinedAt(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('fr-FR', { month: 'long', year: 'numeric' }).format(date)
}

function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)} M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)} K`
  return String(n)
}
