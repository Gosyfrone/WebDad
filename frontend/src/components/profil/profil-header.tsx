'use client'

import { CalendarDays } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { ProfilDetails, ProfilEditableFields } from '@/types'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EditProfilDialog } from '@/components/profil/edit-profil-dialog'

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

  return (
    <header>
      {/* Bannière */}
      <div
        className={cn(
          'h-36 w-full bg-cover bg-center sm:h-48',
          !profil.bannerUrl && 'bg-gradient-to-r from-primary/40 to-primary/10',
        )}
        style={profil.bannerUrl ? { backgroundImage: `url(${profil.bannerUrl})` } : undefined}
      />

      <div className="px-4 pb-3">
        {/* Avatar superposé + action */}
        <div className="flex items-end justify-between">
          <Avatar className="-mt-12 h-24 w-24 border-4 border-background sm:-mt-16 sm:h-32 sm:w-32">
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
                <Button variant="outline" className="rounded-full font-bold">
                  Éditer le profil
                </Button>
              </EditProfilDialog>
            ) : (
              // TODO (issue profil) : brancher l'action « suivre »
              <Button className="rounded-full font-bold" disabled>
                Suivre
              </Button>
            )}
          </div>
        </div>

        {/* Identité */}
        <div className="mt-3 flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-extrabold">{profil.displayName}</h1>
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

        {/* Compteurs */}
        <div className="mt-3 flex gap-5 text-sm">
          <Count value={profil.followingCount} label="Abonnements" />
          <Count value={profil.followersCount} label="Abonnés" />
        </div>
      </div>
    </header>
  )
}

function Count({ value, label }: { value: number; label: string }) {
  return (
    <span className="flex gap-1">
      <span className="font-bold">{formatCount(value)}</span>
      <span className="text-muted-foreground">{label}</span>
    </span>
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
