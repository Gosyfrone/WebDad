'use client'

import { useState } from 'react'
import Link from 'next/link'
import { CalendarDays, LinkIcon, Mail, MapPin } from 'lucide-react'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'
import type { RelationKind } from '@/lib/api'
import { useFollow } from '@/lib/use-follow'
import type { ProfilDetails, ProfilEditableFields, RelationUser } from '@/types'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EditProfilDialog } from '@/components/profil/edit-profil-dialog'
import { RelationsDialog } from '@/components/profil/relations-dialog'

interface ProfilHeaderProps {
  profil: ProfilDetails
  /** Vrai si le profil affiché est celui de l'utilisateur courant. */
  isOwner: boolean
  saving?: boolean
  /** Remontée des champs édités (consommée par le parent pour l'affichage live). */
  onEdit: (fields: ProfilEditableFields) => Promise<void>
}

/**
 * En-tête de la page profil : bannière, avatar superposé, identité, bio,
 * date d'inscription et compteurs d'abonnés. Le propriétaire voit le bouton
 * « Éditer le profil » ; un visiteur verrait « Suivre » (à brancher).
 */
export function ProfilHeader({ profil, isOwner, saving = false, onEdit }: ProfilHeaderProps) {
  const { t, locale } = useLanguage()
  const initials = profil.displayName.charAt(0).toUpperCase()
  const [relationsOpen, setRelationsOpen] = useState(false)
  const [relationsTab, setRelationsTab] = useState<RelationKind>('followers')

  // État de suivi (même hook que la recherche / les suggestions). Différé pour
  // le propriétaire (pas de bouton « Suivre » sur son propre profil).
  const { currentUserId, isFollowing, isPending, toggle } = useFollow(!isOwner)
  const canFollow = !isOwner && currentUserId !== null && currentUserId !== profil.userId

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
                  website: profil.website,
                  location: profil.location,
                  birthDate: toDateInputValue(profil.birthDate),
                  gender: profil.gender,
                }}
                birthDateLocked={Boolean(profil.birthDate)}
                genderLocked={Boolean(profil.gender)}
                displayNameChangedAt={profil.displayNameChangedAt}
                saving={saving}
                onSave={onEdit}
              >
                <Button
                  variant="outline"
                  className="rounded-full border-white/70 bg-white/80 font-bold shadow-sm backdrop-blur hover:bg-white dark:border-white/15 dark:bg-white/10 dark:hover:bg-white/20"
                >
                  {t('profil.edit')}
                </Button>
              </EditProfilDialog>
            ) : canFollow ? (
              <div className="flex items-center gap-2">
                <Button
                  asChild
                  variant="outline"
                  size="icon"
                  aria-label={t('messages.message_action')}
                  title={t('messages.message_action')}
                  className="rounded-full border-white/70 bg-white/80 shadow-sm backdrop-blur hover:bg-white dark:border-white/15 dark:bg-white/10 dark:hover:bg-white/20"
                >
                  <Link href={`${ROUTES.messages}?dm=${profil.userId}`}>
                    <Mail className="h-4 w-4" />
                  </Link>
                </Button>
                <FollowButton
                  following={isFollowing(profil.userId)}
                  pending={isPending(profil.userId)}
                  onToggle={(next) => toggle(toRelationUser(profil), next)}
                />
              </div>
            ) : null}
          </div>
        </div>

        {/* Identité */}
        <div className="mt-3 flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-extrabold text-foreground">{profil.displayName}</h1>
            <Badge variant="secondary">{t(`role.${profil.role}`)}</Badge>
          </div>
          <span className="text-sm text-muted-foreground">@{profil.username}</span>
        </div>

        {/* Bio */}
        {profil.bio && (
          <p className="mt-3 whitespace-pre-wrap text-sm leading-relaxed">{profil.bio}</p>
        )}

        <div className="mt-3 flex flex-wrap gap-x-4 gap-y-2 text-sm text-muted-foreground">
          {profil.location && (
            <span className="flex items-center gap-1.5">
              <MapPin className="h-4 w-4" aria-hidden />
              {profil.location}
            </span>
          )}
          {profil.website && (
            <a
              href={toExternalUrl(profil.website)}
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 text-[#5B6CFF] hover:underline"
            >
              <LinkIcon className="h-4 w-4" aria-hidden />
              {profil.website}
            </a>
          )}
          {profil.birthDate && (
            <span className="flex items-center gap-1.5">
              <CalendarDays className="h-4 w-4" aria-hidden />
              {t('profil.born_on', { date: formatFullDate(profil.birthDate, locale) })}
            </span>
          )}
          <span className="flex items-center gap-1.5">
            <CalendarDays className="h-4 w-4" aria-hidden />
            {t('profil.joined', { date: formatJoinedAt(profil.joinedAt, locale) })}
          </span>
        </div>

        {/* Compteurs (cliquables → modale des relations) — même ordre que la modale */}
        <div className="mt-3 flex gap-5 text-sm">
          <Count
            value={profil.followersCount}
            label={t('profil.followers')}
            onClick={() => openRelations('followers')}
          />
          <Count
            value={profil.followingCount}
            label={t('profil.following')}
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

/** Adapte un ProfilDetails vers le RelationUser attendu par `useFollow`. */
function toRelationUser(profil: ProfilDetails): RelationUser {
  return {
    id: profil.userId,
    username: profil.username,
    displayName: profil.displayName,
    bio: profil.bio,
    avatarUrl: profil.avatarUrl,
  }
}

/**
 * Bouton Suivre⇄Abonné du profil — même comportement que `UserListItem`
 * (recherche / suggestions) : « Abonné » bascule en « Ne plus suivre » au survol.
 */
function FollowButton({
  following,
  pending,
  onToggle,
}: {
  following: boolean
  pending: boolean
  onToggle: (next: boolean) => void
}) {
  const { t } = useLanguage()
  return (
    <Button
      disabled={pending}
      onClick={() => onToggle(!following)}
      variant={following ? 'outline' : 'default'}
      className={cn(
        'group/btn rounded-full font-bold',
        following
          ? 'border-white/70 bg-white/80 backdrop-blur hover:border-destructive/40 hover:bg-destructive/10 hover:text-destructive dark:border-white/15 dark:bg-white/10 dark:hover:bg-destructive/20'
          : 'bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white',
      )}
    >
      {following ? (
        <>
          <span className="group-hover/btn:hidden">{t('follow.followed')}</span>
          <span className="hidden group-hover/btn:inline">{t('follow.unfollow')}</span>
        </>
      ) : (
        t('follow.follow')
      )}
    </Button>
  )
}

function toDateInputValue(iso: string): string {
  if (!iso) return ''
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return date.toISOString().slice(0, 10)
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

/** « juin 2026 » (ou « June 2026 » en anglais) à partir d'une date ISO. */
function formatJoinedAt(iso: string, locale: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const intl = locale === 'en' ? 'en-US' : 'fr-FR'
  return new Intl.DateTimeFormat(intl, { month: 'long', year: 'numeric' }).format(date)
}

function formatFullDate(iso: string, locale: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const intl = locale === 'en' ? 'en-US' : 'fr-FR'
  return new Intl.DateTimeFormat(intl, {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(date)
}

function toExternalUrl(url: string): string {
  if (/^https?:\/\//i.test(url)) return url
  return `https://${url}`
}

function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)} M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)} K`
  return String(n)
}
