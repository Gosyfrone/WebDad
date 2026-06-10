'use client'

import { useCallback, useEffect, useState } from 'react'
import { ArrowLeft, FileText, Loader2, Lock } from 'lucide-react'
import Link from 'next/link'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'
import { useFollow } from '@/lib/use-follow'
import {
  getMyProfil,
  getPublicProfil,
  saveMyProfil,
  subscribeProfilUpdated,
} from '@/lib/profil-client'
import {
  applyProfilUpdateToPosts,
  listByAuthor,
  subscribePostCreated,
  type FeedPost,
} from '@/lib/posts'
import { FOLLOW_CHANGE_EVENT, type FollowChangeDetail } from '@/lib/use-follow'
import { useToast } from '@/hooks/use-toast'
import type { ProfilDetails, ProfilEditableFields } from '@/types'
import { useT } from '@/components/language-provider'
import { PostCard } from '@/components/feed/post-card'
import { ProfilHeader } from '@/components/profil/profil-header'

type ProfilTab = 'posts' | 'replies' | 'likes'

interface ProfilViewProps {
  /** Absent ou vide : profil courant. Présent : profil public par username. */
  username?: string
}

function sortProfilePosts(posts: FeedPost[]): FeedPost[] {
  return [...posts].sort((a, b) => {
    if (a.isPinned !== b.isPinned) return a.isPinned ? -1 : 1
    return profileSortTime(b) - profileSortTime(a)
  })
}

function profileSortTime(post: FeedPost): number {
  return new Date(post.repostedAt || post.createdAt).getTime()
}

function applyPostUpdate(current: FeedPost[], updated: FeedPost): FeedPost[] {
  return sortProfilePosts(
    current.map((post) => {
      if (post.id === updated.id) return updated
      if (updated.isPinned && post.author.id === updated.author.id) {
        return { ...post, isPinned: false, pinnedAt: '' }
      }
      return post
    }),
  )
}

/**
 * Corps de la page profil : en-tête sticky (retour + nb de posts), en-tête de
 * profil éditable, onglets, puis la liste de posts de l'onglet actif.
 *
 * Le profil est chargé via l'API Gateway. Les onglets « Réponses » et
 * « J'aime » restent des placeholders tant que l'API n'expose pas ces flux.
 */
export function ProfilView({ username }: ProfilViewProps) {
  const { toast } = useToast()
  const t = useT()
  const [profil, setProfil] = useState<ProfilDetails | null>(null)
  const [posts, setPosts] = useState<FeedPost[]>([])
  const [tab, setTab] = useState<ProfilTab>('posts')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [followOverride, setFollowOverride] = useState<boolean | null>(null)

  const isOwner = !username
  const followState = useFollow(!isOwner && Boolean(profil?.userId))
  const followsProfile = profil ? followOverride ?? followState.isFollowing(profil.userId) : false
  const accessPending =
    Boolean(profil?.userId) &&
    profil?.visibility === 'private' &&
    !isOwner &&
    followOverride === null &&
    !followState.loaded
  const privateContentLocked =
    Boolean(profil?.userId) &&
    profil?.visibility === 'private' &&
    !isOwner &&
    !accessPending &&
    !followsProfile

  useEffect(() => {
    let cancelled = false

    async function loadProfil() {
      setLoading(true)
      setError('')
      try {
        const nextProfil = username
          ? await getPublicProfil(username)
          : await getMyProfil()
        if (!cancelled) setProfil(nextProfil)
      } catch (err) {
        if (!cancelled) {
          setProfil(null)
          setError(err instanceof Error ? err.message : t('profil.not_found'))
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadProfil()
    return () => {
      cancelled = true
    }
  }, [username, t])

  useEffect(() => {
    setFollowOverride(null)
  }, [profil?.userId])

  const loadPosts = useCallback(async () => {
    if (!profil?.userId) return
    try {
      const list = await listByAuthor(profil.userId)
      setPosts(list)
    } catch {
      setPosts([])
    }
  }, [profil?.userId])

  // Posts de l'auteur (onglet « Posts »), chargés une fois le profil connu.
  useEffect(() => {
    if (!profil?.userId) return
    if (accessPending) return
    if (privateContentLocked) {
      setPosts([])
      return
    }
    let cancelled = false
    listByAuthor(profil.userId).then(
      (list) => {
        if (!cancelled) setPosts(list)
      },
      () => {
        if (!cancelled) setPosts([])
      },
    )
    return () => {
      cancelled = true
    }
  }, [accessPending, privateContentLocked, profil?.userId])

  useEffect(() => {
    function handleFollowChange(event: Event) {
      const { followerUserId, followingUserId, following } = (
        event as CustomEvent<FollowChangeDetail>
      ).detail
      const delta = following ? 1 : -1

      setProfil((current) => {
        if (!current) return current

        const updates: Partial<ProfilDetails> = {}
        if (followingUserId === current.userId) {
          updates.followersCount = clampCount(current.followersCount + delta)
        }
        if (followerUserId === current.userId) {
          updates.followingCount = clampCount(current.followingCount + delta)
        }

        return Object.keys(updates).length ? { ...current, ...updates } : current
      })
    }

    window.addEventListener(FOLLOW_CHANGE_EVENT, handleFollowChange)
    return () => window.removeEventListener(FOLLOW_CHANGE_EVENT, handleFollowChange)
  }, [])

  useEffect(() => {
    if (!profil?.userId) return undefined
    return subscribePostCreated((post) => {
      const belongsToProfile =
        post.author.id === profil.userId || post.repostedById === profil.userId
      if (!belongsToProfile) return
      setPosts((prev) =>
        sortProfilePosts(
          prev.some((p) => p.id === post.id)
            ? prev.map((p) => (p.id === post.id ? post : p))
            : [post, ...prev],
        ),
      )
    })
  }, [profil?.userId])

  useEffect(
    () =>
      subscribeProfilUpdated((updatedProfil) => {
        setPosts((prev) => applyProfilUpdateToPosts(prev, updatedProfil))
      }),
    [],
  )

  function handleDeleted(id: string) {
    setPosts((prev) => prev.filter((p) => p.id !== id))
  }

  function handleUpdated(post: FeedPost) {
    setPosts((prev) => applyPostUpdate(prev, post))
  }

  async function handleEdit(fields: ProfilEditableFields) {
    if (!profil) return

    setSaving(true)
    try {
      const updated = await saveMyProfil(fields, profil.profileExists)
      setProfil(updated)
      toast({ title: t('profil.updated') })
    } catch (err) {
      toast({
        title: t('profil.update_failed'),
        description: err instanceof Error ? err.message : t('profil.save_failed'),
        variant: 'destructive',
      })
      throw err
    } finally {
      setSaving(false)
    }
  }

  function handleFollowChanged(following: boolean) {
    if (profil?.visibility !== 'private') return
    setFollowOverride(following)
    if (following) void loadPosts()
    else setPosts([])
  }

  if (loading) {
    return (
      <div className="flex min-h-[45vh] items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF]" aria-hidden />
      </div>
    )
  }

  if (error || !profil) {
    return (
      <div className="mx-4 mt-6 rounded-[24px] border border-white/55 bg-white/72 px-5 py-8 text-center shadow-[0_18px_54px_rgba(91,108,255,0.12)] backdrop-blur-xl">
        <h1 className="text-lg font-bold text-slate-950">{t('profil.unavailable')}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{error}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      {/* En-tête sticky */}
      <div className="panel sticky top-0 z-10 flex items-center gap-6 border-b px-4 py-2">
        <Link
          href={ROUTES.feed}
          aria-label={t('profil.back_aria')}
          className="rounded-full p-2 transition-colors hover:bg-accent hover:text-[#5B6CFF] dark:hover:text-[#9aa6ff]"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div className="flex flex-col">
          <span className="font-bold leading-tight text-foreground">{profil.displayName}</span>
          <span className="text-xs text-muted-foreground">
            {t(posts.length > 1 ? 'profil.posts_count_other' : 'profil.posts_count_one', {
              count: posts.length,
            })}
          </span>
        </div>
      </div>

      <ProfilHeader
        profil={profil}
        isOwner={isOwner}
        saving={saving}
        onEdit={handleEdit}
        onFollowChanged={handleFollowChanged}
      />

      {/* Onglets */}
      <div className="panel flex border-b">
        <TabButton active={tab === 'posts'} onClick={() => setTab('posts')}>
          {t('profil.tab_posts')}
        </TabButton>
        <TabButton active={tab === 'replies'} onClick={() => setTab('replies')}>
          {t('profil.tab_replies')}
        </TabButton>
        <TabButton active={tab === 'likes'} onClick={() => setTab('likes')}>
          {t('profil.tab_likes')}
        </TabButton>
      </div>

      {/* Contenu de l'onglet */}
      {accessPending ? (
        <CenteredTab>
          <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
        </CenteredTab>
      ) : privateContentLocked ? (
        <PrivateTab />
      ) : tab === 'posts' ? (
        posts.length > 0 ? (
          <div className="divide-y divide-border">
            {posts.map((post) => (
              <PostCard
                key={post.id}
                post={post}
                showPinBadge
                onDeleted={handleDeleted}
                onUpdated={handleUpdated}
              />
            ))}
          </div>
        ) : (
          <EmptyTab message={t('profil.empty_posts')} />
        )
      ) : (
        <EmptyTab
          message={tab === 'replies' ? t('profil.empty_replies') : t('profil.empty_likes')}
        />
      )}
    </div>
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

function EmptyTab({ message }: { message: string }) {
  return (
    <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
      <FileText className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
      <p className="max-w-sm text-sm text-muted-foreground">{message}</p>
    </div>
  )
}

function PrivateTab() {
  const t = useT()
  return (
    <CenteredTab>
      <Lock className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
      <div className="max-w-sm space-y-1">
        <p className="text-sm font-semibold text-foreground">{t('profil.private_title')}</p>
        <p className="text-sm text-muted-foreground">{t('profil.private_message')}</p>
      </div>
    </CenteredTab>
  )
}

function CenteredTab({ children }: { children: React.ReactNode }) {
  return (
    <div className="glass mx-4 mt-6 flex min-h-[14rem] flex-col items-center justify-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
      {children}
    </div>
  )
}

function clampCount(value: number): number {
  return Math.max(0, value)
}
