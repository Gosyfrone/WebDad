'use client'

import { useCallback, useEffect, useState } from 'react'
import { ArrowLeft, FileText, Loader2, Lock, MessageCircle, ShieldAlert } from 'lucide-react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'

import { cn } from '@/lib/utils'
import { ROUTES, postHref } from '@/lib/routes'
import { useFollow } from '@/lib/use-follow'
import {
  getMyProfil,
  getPublicProfil,
  saveMyProfil,
  subscribeProfilUpdated,
} from '@/lib/profil-client'
import {
  applyProfilUpdateToPosts,
  applyStatsToPost,
  applyStatsToPosts,
  listByAuthor,
  listCommentsByAuthor,
  listLikedByUser,
  subscribePostCreated,
  type FeedPost,
  type ReplyContext,
} from '@/lib/posts'
import { usePostStatsPolling } from '@/lib/use-post-stats-polling'
import { FOLLOW_CHANGE_EVENT, type FollowChangeDetail } from '@/lib/use-follow'
import { useToast } from '@/hooks/use-toast'
import type { ProfilDetails, ProfilEditableFields } from '@/types'
import { useT } from '@/components/language-provider'
import { PostCard } from '@/components/feed/post-card'
import { CommentRow } from '@/components/feed/comment-section'
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
  const [replies, setReplies] = useState<ReplyContext[]>([])
  const [likedPosts, setLikedPosts] = useState<FeedPost[]>([])
  const [likesPrivateLocked, setLikesPrivateLocked] = useState(false)
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

  // Réponses de l'auteur (onglet « Réponses »), chargées au premier clic.
  useEffect(() => {
    if (tab !== 'replies') return
    if (!profil?.userId) return
    if (accessPending) return
    if (privateContentLocked) {
      setReplies([])
      return
    }
    let cancelled = false
    listCommentsByAuthor(profil.userId).then(
      (list) => {
        if (!cancelled) setReplies(list)
      },
      () => {
        if (!cancelled) setReplies([])
      },
    )
    return () => {
      cancelled = true
    }
  }, [tab, accessPending, privateContentLocked, profil?.userId])

  // Likes de l'auteur (onglet « J'aime »), chargés au premier clic.
  useEffect(() => {
    if (tab !== 'likes') return
    if (!profil?.userId) return
    if (accessPending) return
    if (privateContentLocked) {
      setLikedPosts([])
      return
    }
    let cancelled = false
    setLikesPrivateLocked(false)
    listLikedByUser(profil.userId).then(
      (list) => {
        if (!cancelled) setLikedPosts(list)
      },
      (err: unknown) => {
        if (cancelled) return
        if (err instanceof Error && err.message === 'likes_private') {
          setLikesPrivateLocked(true)
        }
        setLikedPosts([])
      },
    )
    return () => {
      cancelled = true
    }
  }, [tab, accessPending, privateContentLocked, profil?.userId])

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

  // Compteurs dynamiques : refetch périodique des likes/commentaires/reposts des
  // posts affichés (onglets Posts / J'aime / Réponses), façon X, sans toucher
  // l'état « moi ». Un lot couvre l'union des trois listes.
  usePostStatsPolling(
    () => [
      ...posts.map((p) => p.id),
      ...likedPosts.map((p) => p.id),
      ...replies.flatMap((r) => (r.parentPost ? [r.parentPost.id] : [])),
    ],
    (stats) => {
      setPosts((prev) => applyStatsToPosts(prev, stats))
      setLikedPosts((prev) => applyStatsToPosts(prev, stats))
      setReplies((prev) => {
        let changed = false
        const next = prev.map((r) => {
          if (!r.parentPost) return r
          const updated = applyStatsToPost(r.parentPost, stats)
          if (updated === r.parentPost) return r
          changed = true
          return { ...r, parentPost: updated }
        })
        return changed ? next : prev
      })
    },
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

  // Compte banni : un visiteur (pas le propriétaire) ne voit ni le profil ni
  // les posts, seulement un état « compte banni » (le back masque déjà le
  // compte des listes ; ici on couvre l'accès direct par URL).
  if (!isOwner && !profil.isActive) {
    return (
      <div className="flex flex-col">
        <div className="panel sticky top-0 z-10 flex items-center gap-6 border-b px-4 py-2">
          <Link
            href={ROUTES.feed}
            scroll={false}
            aria-label={t('profil.back_aria')}
            className="rounded-full p-2 transition-colors hover:bg-accent hover:text-[#5B6CFF] dark:hover:text-[#9aa6ff]"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <span className="font-bold leading-tight text-foreground">@{profil.username}</span>
        </div>
        <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
          <ShieldAlert className="h-10 w-10 text-muted-foreground" aria-hidden />
          <h1 className="text-lg font-bold text-foreground">{t('profil.banned_title')}</h1>
          <p className="max-w-sm text-sm text-muted-foreground">{t('profil.banned_desc')}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      {/* En-tête sticky */}
      <div className="panel sticky top-0 z-10 flex items-center gap-6 border-b px-4 py-2">
        <Link
          href={ROUTES.feed}
          scroll={false}
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
      ) : tab === 'replies' ? (
        replies.length > 0 ? (
          <div className="flex flex-col gap-3 py-3">
            {replies.map((reply) => (
              <ReplyCard key={reply.comment.id} reply={reply} />
            ))}
          </div>
        ) : (
          <EmptyTab message={t('profil.empty_replies')} />
        )
      ) : likesPrivateLocked ? (
        <LikesPrivateTab username={profil.username} />
      ) : likedPosts.length > 0 ? (
        <div className="divide-y divide-border">
          {likedPosts.map((post) => (
            <PostCard key={post.id} post={post} onDeleted={() => {}} onUpdated={() => {}} />
          ))}
        </div>
      ) : (
        <EmptyTab message={t('profil.empty_likes')} />
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

/**
 * Onglet « Réponses » (façon Twitter) : le post parent complet et interactif en
 * haut, puis une carte encadrée façon feed contenant — si la réponse répond à un
 * autre commentaire — le commentaire parent, et enfin la réponse de
 * l'utilisateur (avatar + identité + contenu via `CommentRow`). Cliquer la carte
 * ouvre le thread et défile directement sur le commentaire.
 */
function ReplyCard({ reply }: { reply: ReplyContext }) {
  const t = useT()
  const router = useRouter()
  const { comment, parentPostId, parentPostAuthor, parentPost, parentComment } = reply
  // On répond au commentaire parent s'il existe, sinon au post.
  const replyTarget = parentComment?.author.username || parentPostAuthor.username || '…'
  const href = `${postHref(parentPostId)}?comment=${encodeURIComponent(comment.id)}`
  return (
    <div className="glass group mx-3 flex flex-col rounded-[24px] border backdrop-blur-xl transition hover:bg-white/85 hover:shadow-[0_20px_56px_rgba(91,108,255,0.16)] dark:hover:bg-[#1f1633]/80">
      {parentPost ? (
        <PostCard
          post={parentPost}
          embedded
          onDeleted={() => {}}
          onUpdated={() => {}}
          onCommentClick={() => router.push(postHref(parentPostId))}
        />
      ) : (
        <p className="flex items-center gap-1 px-4 pt-3 text-sm text-muted-foreground">
          <MessageCircle className="h-3.5 w-3.5" aria-hidden />
          {t('profil.replies_deleted_parent')}
        </p>
      )}

      {/* Carte encadrée façon feed : commentaire parent (optionnel) + réponse. */}
      <div
        role="link"
        tabIndex={0}
        onClick={() => router.push(href)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') router.push(href)
        }}
        className="mx-3 mb-3 cursor-pointer rounded-2xl border border-border/70 bg-card/70 p-3 shadow-sm transition-colors hover:bg-card/90"
      >
        <p className="mb-2 flex items-center gap-1 text-xs text-muted-foreground">
          <MessageCircle className="h-3 w-3" aria-hidden />
          {t('profil.replies_in_reply_to', { username: replyTarget })}
        </p>

        {parentComment && (
          <CommentRow comment={parentComment} />
        )}
        <div className={parentComment ? 'ml-5 border-l border-border pl-3 pt-2' : ''}>
          <CommentRow comment={comment} />
        </div>
      </div>
    </div>
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

function LikesPrivateTab({ username }: { username: string }) {
  const t = useT()
  return (
    <CenteredTab>
      <Lock className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
      <div className="max-w-sm space-y-1">
        <p className="text-sm font-semibold text-foreground">
          {t('profil.likes_private_title', { username })}
        </p>
        <p className="text-sm text-muted-foreground">{t('profil.likes_private_message')}</p>
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
