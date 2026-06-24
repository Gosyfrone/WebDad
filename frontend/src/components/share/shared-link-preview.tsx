'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'

import { getPostById, type FeedPost } from '@/lib/posts'
import { getUserByUsername } from '@/lib/api'
import { postHref, profilHref } from '@/lib/routes'
import type { SharedRef } from '@/lib/share'
import type { RelationUser } from '@/types'
import { cn, initialOf } from '@/lib/utils'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { CertificationBadge } from '@/components/profil/certification-badge'

// Caches module-level : un même post/profil partagé apparaît souvent plusieurs
// fois dans un fil de discussion → on évite de le refetcher à chaque rendu.
const postCache = new Map<string, Promise<FeedPost | null>>()
const userCache = new Map<string, Promise<RelationUser | null>>()

function loadPost(id: string): Promise<FeedPost | null> {
  if (!postCache.has(id)) postCache.set(id, getPostById(id).catch(() => null))
  return postCache.get(id)!
}

function loadUser(handle: string): Promise<RelationUser | null> {
  if (!userCache.has(handle)) userCache.set(handle, getUserByUsername(handle).catch(() => null))
  return userCache.get(handle)!
}

const CARD_CLASS =
  'mt-1 block w-full max-w-xs overflow-hidden rounded-2xl border border-border bg-background/70 text-foreground shadow-sm transition-colors hover:bg-background'

/**
 * Aperçu enrichi d'un lien Breezy (post ou profil) partagé dans une conversation.
 * Le lien reste affiché en clair dans la bulle ; cette carte vient EN PLUS,
 * en dessous, et navigue (SPA) vers la ressource au clic. Rien n'est rendu tant
 * que la ressource n'est pas chargée ou si elle est inaccessible (lien seul).
 */
export function SharedLinkPreview({ target }: { target: SharedRef }) {
  if (target.kind === 'post') return <PostPreview id={target.id} />
  return <ProfilePreview handle={target.id} />
}

function PostPreview({ id }: { id: string }) {
  const [post, setPost] = useState<FeedPost | null>(null)

  useEffect(() => {
    let alive = true
    loadPost(id).then((p) => {
      if (alive) setPost(p)
    })
    return () => {
      alive = false
    }
  }, [id])

  if (!post) return null
  const thumb = post.media?.find((m) => m.type === 'image')

  return (
    <Link href={postHref(post.id)} className={CARD_CLASS}>
      {thumb && (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={thumb.url} alt="" className="h-28 w-full object-cover" />
      )}
      <div className="flex flex-col gap-1 p-3">
        <div className="flex items-center gap-2">
          <Avatar className="h-6 w-6 shrink-0">
            {post.author.avatarUrl && <AvatarImage src={post.author.avatarUrl} alt="" />}
            <AvatarFallback className="text-[10px]">
              {initialOf(post.author.displayName, post.author.username)}
            </AvatarFallback>
          </Avatar>
          <span className="flex min-w-0 items-center gap-1 text-xs font-bold">
            <span className="truncate">{post.author.displayName}</span>
            <CertificationBadge userId={post.author.id} certification={post.author.certification} className="h-4 w-4" />
          </span>
          {post.author.username && (
            <span className="truncate text-xs text-muted-foreground">@{post.author.username}</span>
          )}
        </div>
        {post.content && (
          <p className="line-clamp-3 whitespace-pre-wrap break-words text-sm text-muted-foreground">
            {post.content}
          </p>
        )}
      </div>
    </Link>
  )
}

function ProfilePreview({ handle }: { handle: string }) {
  const [user, setUser] = useState<RelationUser | null>(null)

  useEffect(() => {
    let alive = true
    loadUser(handle).then((u) => {
      if (alive) setUser(u)
    })
    return () => {
      alive = false
    }
  }, [handle])

  if (!user) return null

  return (
    <Link href={profilHref(user.username)} className={cn(CARD_CLASS, 'flex items-center gap-3 p-3')}>
      <Avatar className="h-12 w-12 shrink-0">
        {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt="" />}
        <AvatarFallback className="bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white">
          {initialOf(user.displayName, user.username)}
        </AvatarFallback>
      </Avatar>
      <div className="flex min-w-0 flex-col">
        <span className="flex min-w-0 items-center gap-1 text-sm font-bold">
          <span className="truncate">{user.displayName}</span>
          <CertificationBadge userId={user.id} certification={user.certification} className="h-4 w-4" />
        </span>
        <span className="truncate text-xs text-muted-foreground">@{user.username}</span>
      </div>
    </Link>
  )
}
