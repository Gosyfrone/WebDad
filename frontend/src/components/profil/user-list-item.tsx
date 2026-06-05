'use client'

import { cn } from '@/lib/utils'
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
 * Ligne d'utilisateur réutilisable (modale des relations, suggestions) :
 * avatar, nom affiché, @handle, bio tronquée et bouton Suivre⇄Abonné.
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

  return (
    <div className="flex items-start gap-3 px-4 py-3 transition-colors hover:bg-accent">
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
          onClick={() => onToggleFollow(user, !isFollowing)}
          className={cn(
            'group/btn mt-0.5 shrink-0 rounded-full font-bold',
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
