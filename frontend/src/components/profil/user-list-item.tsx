'use client'

import Link from 'next/link'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'
import type { RelationUser } from '@/types'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'

interface UserListItemProps {
  user: RelationUser
  /** Vrai si l'utilisateur courant suit déjà cette personne. */
  isFollowing: boolean
  /** Vrai si cette ligne est l'utilisateur courant lui-même (pas de bouton). */
  isSelf?: boolean
  /** Désactive le bouton (action en cours). */
  pending?: boolean
  /** Affiche la bio sous le @handle (défaut : oui). Compact en sidebar. */
  showBio?: boolean
  /** Bascule suivre / ne plus suivre. */
  onToggleFollow: (user: RelationUser, next: boolean) => void
}

/**
 * Ligne d'utilisateur réutilisable (modale des relations, suggestions,
 * recherche) : avatar, nom affiché, @handle, bio tronquée et bouton
 * Suivre⇄Abonné.
 *
 * Toute la ligne (zone de survol) est cliquable et mène au profil de la
 * personne (`/profil/<username>`, ou `/profil` pour soi) via un lien « étiré »
 * (overlay `absolute inset-0`). Le bouton Suivre est remonté au-dessus du lien
 * (`z-10`) pour rester actionnable sans déclencher la navigation.
 *
 * Le bouton « Abonné » passe en « Ne plus suivre » au survol (convention X) ;
 * il disparaît sur sa propre ligne (`isSelf`).
 */
export function UserListItem({
  user,
  isFollowing,
  isSelf = false,
  pending = false,
  showBio = true,
  onToggleFollow,
}: UserListItemProps) {
  const initials = (user.displayName.charAt(0) || user.username.charAt(0) || '?').toUpperCase()
  const href = isSelf ? ROUTES.profil : `${ROUTES.profil}/${user.username}`

  return (
    <div className="relative flex items-start gap-3 px-4 py-3 transition-colors hover:bg-accent">
      {/* Lien « étiré » : rend toute la ligne cliquable vers le profil. */}
      <Link
        href={href}
        aria-label={`Voir le profil de ${user.displayName}`}
        className="absolute inset-0 z-0"
      />

      <Avatar className="h-10 w-10 shrink-0">
        {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
        <AvatarFallback>{initials}</AvatarFallback>
      </Avatar>

      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm font-bold text-foreground">{user.displayName}</span>
        <span className="truncate text-sm text-muted-foreground">@{user.username}</span>
        {showBio && user.bio && (
          <p className="mt-0.5 line-clamp-2 text-sm text-muted-foreground">{user.bio}</p>
        )}
      </div>

      {!isSelf && (
        <Button
          size="sm"
          variant={isFollowing ? 'outline' : 'default'}
          disabled={pending}
          onClick={(e) => {
            // Empêche le clic du bouton de déclencher la navigation du lien étiré.
            e.preventDefault()
            e.stopPropagation()
            onToggleFollow(user, !isFollowing)
          }}
          className={cn(
            'group/btn relative z-10 mt-0.5 shrink-0 rounded-full font-bold',
            isFollowing
              ? 'border-white/70 bg-white/80 backdrop-blur hover:border-destructive/40 hover:bg-destructive/10 hover:text-destructive dark:border-white/15 dark:bg-white/10 dark:hover:bg-destructive/20'
              : 'bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white',
          )}
        >
          {isFollowing ? (
            <>
              <span className="group-hover/btn:hidden">Abonné</span>
              <span className="hidden group-hover/btn:inline">Ne plus suivre</span>
            </>
          ) : (
            'Suivre'
          )}
        </Button>
      )}
    </div>
  )
}
