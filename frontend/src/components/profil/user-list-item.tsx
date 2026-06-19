'use client'

import { cn, initialOf } from '@/lib/utils'
import type { RelationUser } from '@/types'
import { useT } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { ProfilLink } from '@/components/profil/profil-link'

interface UserListItemProps {
  user: RelationUser
  /** Vrai si l'utilisateur courant suit déjà cette personne. */
  isFollowing: boolean
  /** Vrai si une demande de suivi est en attente (compte privé) → bouton « En attente ». */
  isRequested?: boolean
  /** Vrai si cette ligne est l'utilisateur courant lui-même (pas de bouton). */
  isSelf?: boolean
  /** Désactive le bouton (action en cours). */
  pending?: boolean
  /** Affiche la bio sous le @handle (défaut : oui). Compact en sidebar. */
  showBio?: boolean
  /** Resserre la ligne pour les cartes latérales tout en gardant une marge basse. */
  compact?: boolean
  /** Déclenché quand la ligne ouvre le profil. */
  onProfileOpen?: (user: RelationUser) => void
  /** Action compacte affichée à droite de la ligne (ex. retirer d'un historique). */
  trailingAction?: React.ReactNode
  /** Bascule suivre / ne plus suivre. */
  onToggleFollow: (user: RelationUser, next: boolean) => void
  /** Affiche l'action de retrait d'un abonné à la place du follow. */
  canRemoveFollower?: boolean
  /** Retire cette personne des abonnés de l'utilisateur courant. */
  onRemoveFollower?: (user: RelationUser) => void
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
  isRequested = false,
  isSelf = false,
  pending = false,
  showBio = true,
  compact = false,
  onProfileOpen,
  trailingAction,
  onToggleFollow,
  canRemoveFollower = false,
  onRemoveFollower,
}: UserListItemProps) {
  const t = useT()
  const initials = initialOf(user.displayName, user.username)

  return (
    <div
      className={cn(
        'relative flex gap-3 rounded-[18px] transition-colors hover:bg-accent',
        compact ? 'items-start px-3 py-1.5' : 'items-start px-4 py-3',
      )}
    >
      {/* Lien « étiré » : rend toute la ligne cliquable vers le profil. */}
      <ProfilLink
        author={{ id: user.id, username: user.username }}
        preview={false}
        aria-label={t('list.view_profile_aria', { name: user.displayName })}
        className="absolute inset-0 z-0"
        onClick={() => onProfileOpen?.(user)}
      >
        <span className="sr-only">{t('list.view_profile_aria', { name: user.displayName })}</span>
      </ProfilLink>

      <ProfilLink
        author={{ id: user.id, username: user.username }}
        className="relative z-10 shrink-0"
        onClick={() => onProfileOpen?.(user)}
      >
        <Avatar className={cn(compact ? 'h-9 w-9' : 'h-10 w-10')}>
          {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
          <AvatarFallback>{initials}</AvatarFallback>
          <ActivityPresenceDot userId={user.id} />
        </Avatar>
      </ProfilLink>

      <div className={cn('flex min-w-0 flex-1 flex-col', compact && 'pr-1')}>
        <ProfilLink
          author={{ id: user.id, username: user.username }}
          className={cn(
            'relative z-10 font-bold text-foreground hover:underline',
            compact ? 'line-clamp-2 break-words text-[13px] leading-4' : 'truncate text-sm',
          )}
          onClick={() => onProfileOpen?.(user)}
        >
          {user.displayName}
        </ProfilLink>
        <ProfilLink
          author={{ id: user.id, username: user.username }}
          className={cn(
            'relative z-10 truncate text-muted-foreground hover:underline',
            compact ? 'text-[13px] leading-4' : 'text-sm',
          )}
          onClick={() => onProfileOpen?.(user)}
        >
          @{user.username}
        </ProfilLink>
        {showBio && user.bio && (
          <p className="mt-0.5 line-clamp-2 text-sm text-muted-foreground">{user.bio}</p>
        )}
      </div>

      {trailingAction && <div className="relative z-10 mt-0.5 shrink-0">{trailingAction}</div>}

      {!isSelf && canRemoveFollower && onRemoveFollower ? (
        <Button
          size="sm"
          variant="outline"
          disabled={pending}
          onClick={(e) => {
            // Empêche le clic du bouton de déclencher la navigation du lien étiré.
            e.preventDefault()
            e.stopPropagation()
            onRemoveFollower(user)
          }}
          className={cn(
            'relative z-10 mt-0.5 shrink-0 rounded-full border-white/70 bg-white/80 font-bold backdrop-blur hover:border-destructive/40 hover:bg-destructive/10 hover:text-destructive dark:border-white/15 dark:bg-white/10 dark:hover:bg-destructive/20',
            compact && 'mt-0',
          )}
        >
          {t('follow.remove_follower')}
        </Button>
      ) : !isSelf ? (
        <Button
          size="sm"
          variant={isFollowing ? 'outline' : 'default'}
          disabled={pending || isRequested}
          onClick={(e) => {
            // Empêche le clic du bouton de déclencher la navigation du lien étiré.
            e.preventDefault()
            e.stopPropagation()
            onToggleFollow(user, !isFollowing)
          }}
          className={cn(
            'group/btn relative z-10 shrink-0 rounded-full font-bold',
            compact ? 'mt-0' : 'mt-0.5',
            compact && 'h-8 min-w-[76px] px-3 text-xs',
            isFollowing
              ? 'border-white/70 bg-white/80 backdrop-blur hover:border-destructive/40 hover:bg-destructive/10 hover:text-destructive dark:border-white/15 dark:bg-white/10 dark:hover:bg-destructive/20'
              : 'bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-white',
          )}
        >
          {isRequested ? (
            t('follow.requested')
          ) : isFollowing && !compact ? (
            <>
              <span className="group-hover/btn:hidden">{t('follow.followed')}</span>
              <span className="hidden group-hover/btn:inline">{t('follow.unfollow')}</span>
            </>
          ) : isFollowing ? (
            t('follow.followed')
          ) : (
            t('follow.follow')
          )}
        </Button>
      ) : null}
    </div>
  )
}
