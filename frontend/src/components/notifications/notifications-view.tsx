'use client'

import { useEffect } from 'react'
import Link from 'next/link'
import { AtSign, Bell, Heart, MessageCircle, Quote, Repeat2, Reply, Send, Trash2, UserPlus } from 'lucide-react'

import { cn, initialOf, timeAgo } from '@/lib/utils'
import { type AppNotification, type NotificationType, notificationHref } from '@/lib/notifications'
import { acceptFollowRequest, rejectFollowRequest } from '@/lib/api'
import { useLanguage } from '@/components/language-provider'
import { useNotifications } from '@/components/notifications-provider'
import { ProfilLink } from '@/components/profil/profil-link'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'

/** Icône (et couleur) par type de notification. */
const TYPE_ICON: Record<NotificationType, { Icon: React.ElementType; className: string }> = {
  like: { Icon: Heart, className: 'text-rose-500' },
  comment_like: { Icon: Heart, className: 'text-rose-500' },
  comment: { Icon: MessageCircle, className: 'text-[#5B6CFF] dark:text-[#9aa6ff]' },
  reply: { Icon: Reply, className: 'text-[#5B6CFF] dark:text-[#9aa6ff]' },
  mention: { Icon: AtSign, className: 'text-[#47D9FF]' },
  repost: { Icon: Repeat2, className: 'text-emerald-500' },
  quote: { Icon: Quote, className: 'text-[#8D3DFF]' },
  follow: { Icon: UserPlus, className: 'text-emerald-500' },
  message: { Icon: Send, className: 'text-[#5B6CFF] dark:text-[#9aa6ff]' },
  message_mention: { Icon: Send, className: 'text-[#8D3DFF]' },
  follow_request: { Icon: UserPlus, className: 'text-emerald-500' },
  follow_request_accepted: { Icon: UserPlus, className: 'text-emerald-500' },
  follow_request_accept_confirm: { Icon: UserPlus, className: 'text-emerald-500' },
  post_purge_warning: { Icon: Trash2, className: 'text-amber-500' },
}

export function NotificationsView() {
  const { t, locale } = useLanguage()
  const { items, loading, hasMore, loadInitial, loadMore, markAllSeen } = useNotifications()

  // À l'ouverture : charger la première page puis tout marquer comme lu (badge → 0).
  useEffect(() => {
    loadInitial()
    markAllSeen()
  }, [loadInitial, markAllSeen])

  /** Texte agrégé « X (et N autres) … » selon le type. */
  function describe(n: AppNotification): string {
    const name = n.actor.displayName
    const count = n.othersCount
    switch (n.type) {
      case 'like':
        return count > 0
          ? t('notifications.like_other', { name, count })
          : t('notifications.like_one', { name })
      case 'comment_like':
        return count > 0
          ? t('notifications.comment_like_other', { name, count })
          : t('notifications.comment_like_one', { name })
      case 'comment':
        return count > 0
          ? t('notifications.comment_other', { name, count })
          : t('notifications.comment_one', { name })
      case 'reply':
        return count > 0
          ? t('notifications.reply_other', { name, count })
          : t('notifications.reply_one', { name })
      case 'repost':
        return count > 0
          ? t('notifications.repost_other', { name, count })
          : t('notifications.repost_one', { name })
      case 'mention':
        return t('notifications.mention', { name })
      case 'quote':
        return t('notifications.quote', { name })
      case 'message':
        return count > 0
          ? t('notifications.message_other', { name, count })
          : t('notifications.message_one', { name })
      case 'message_mention':
        return count > 0
          ? t('notifications.message_mention_other', { name, count })
          : t('notifications.message_mention_one', { name })
      case 'follow':
        return count > 0
          ? t('notifications.follow_other', { name, count })
          : t('notifications.follow_one', { name })
      case 'follow_request':
        return t('notifications.follow_request', { name })
      case 'follow_request_accepted':
        return t('notifications.follow_request_accepted', { name })
      case 'follow_request_accept_confirm':
        return t('notifications.follow_request_accept_confirm', { name })
      case 'post_purge_warning':
        return t('notifications.post_purge_warning')
      default:
        // Type inconnu (notif legacy) : pas de texte descriptif, mais la ligne
        // reste affichée (acteur + date) sans casser le rendu.
        return ''
    }
  }

  return (
    <div className="flex flex-col">
      {/* En-tête desktop : sur mobile, le titre est porté par l'en-tête global. */}
      <header className="panel sticky top-0 z-20 hidden border-b px-4 py-3 backdrop-blur-2xl lg:block">
        <h1 className="text-xl font-bold">{t('notifications.title')}</h1>
      </header>

      {items.length === 0 && !loading ? (
        <div className="flex flex-col items-center gap-3 px-6 py-16 text-center">
          <Bell className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
          <p className="text-muted-foreground">{t('notifications.empty')}</p>
        </div>
      ) : (
        <ul className="flex flex-col">
          {items.map((n) => {
            // Garde défensive : un type inconnu (notif legacy d'avant l'ajout de
            // l'enum côté base — validé à l'écriture, pas à la lecture) ne doit
            // pas faire planter toute la page. Repli sur une icône neutre.
            const { Icon, className } = TYPE_ICON[n.type] ?? {
              Icon: Bell,
              className: 'text-muted-foreground',
            }
            const fallback = initialOf(n.actor.displayName, n.actor.username)
            // Notification SYSTÈME (sans acteur) : préavis de purge RGPD. Pas de
            // lien profil, pas de navigation (le tweet masqué n'est pas visible).
            const isSystem = n.type === 'post_purge_warning'
            return (
              <li
                key={n.id}
                className={cn(
                  'relative flex items-start gap-3 border-b px-4 py-3 transition hover:bg-accent',
                  !n.isRead && 'bg-[#5B6CFF]/5',
                )}
              >
                {!isSystem && !['follow', 'follow_request', 'follow_request_accepted', 'follow_request_accept_confirm'].includes(n.type) && (
                  <Link
                    href={notificationHref(n)}
                    aria-label={describe(n)}
                    className="absolute inset-0"
                  />
                )}

                {/* Avatar de l'acteur (au-dessus du lien étiré → mène au profil).
                    Pour une notification système, juste l'icône dans une pastille. */}
                <div className="relative z-10">
                  {isSystem ? (
                    <span className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-500/10">
                      <Icon className={cn('h-5 w-5', className)} aria-hidden />
                    </span>
                  ) : (
                    <>
                      <ProfilLink author={n.actor}>
                        <Avatar className="h-10 w-10">
                          {n.actor.avatarUrl && (
                            <AvatarImage src={n.actor.avatarUrl} alt={n.actor.displayName} />
                          )}
                          <AvatarFallback>{fallback}</AvatarFallback>
                          <ActivityPresenceDot userId={n.actor.id} />
                        </Avatar>
                      </ProfilLink>
                      <span className="absolute -bottom-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-background shadow">
                        <Icon className={cn('h-3.5 w-3.5', className)} aria-hidden />
                      </span>
                    </>
                  )}
                </div>

                <div className="min-w-0 flex-1">
                  <p className="text-sm leading-snug">
                    {isSystem ? (
                      <span className="text-foreground/90">{describe(n)}</span>
                    ) : n.type === 'follow_request_accept_confirm' ? (
                      <>
                        <span className="text-foreground/90">
                          {t('notifications.follow_request_accept_confirm_prefix')}{' '}
                        </span>
                        <ProfilLink author={n.actor} className="relative z-10 font-semibold hover:underline">
                          {n.actor.displayName}
                        </ProfilLink>
                      </>
                    ) : (
                      <>
                        <ProfilLink author={n.actor} className="relative z-10 font-semibold hover:underline">
                          {n.actor.displayName}
                        </ProfilLink>{' '}
                        {/* describe() commence toujours par « {name} » → on retire le nom
                            (déjà rendu en lien gras) + l'espace qui suit. */}
                        <span className="text-foreground/90">
                          {describe(n).slice(n.actor.displayName.length + 1)}
                        </span>
                      </>
                    )}
                  </p>
                  <span className="text-xs text-muted-foreground">{timeAgo(n.updatedAt, locale)}</span>
                  {n.type === 'follow_request' && (
                    <div className="relative z-10 mt-2 flex gap-2">
                      <Button
                        size="sm"
                        onClick={() => acceptFollowRequest(n.actor.id).then(loadInitial)}
                      >
                        {t('notifications.accept')}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => rejectFollowRequest(n.actor.id).then(loadInitial)}
                      >
                        {t('notifications.reject')}
                      </Button>
                    </div>
                  )}
                </div>
              </li>
            )
          })}
        </ul>
      )}

      {hasMore && items.length > 0 && (
        <div className="flex justify-center p-4">
          <Button variant="ghost" onClick={loadMore} disabled={loading}>
            {t('notifications.load_more')}
          </Button>
        </div>
      )}
    </div>
  )
}
