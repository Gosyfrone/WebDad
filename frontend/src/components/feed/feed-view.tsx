'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { Loader2, Users } from 'lucide-react'

import { cn } from '@/lib/utils'
import {
  filterMutedPosts,
  readMutedWords,
  subscribeMutedWords,
} from '@/lib/content-filters'
import { getMe } from '@/lib/api'
import { useInfiniteScroll } from '@/lib/use-infinite-scroll'
import {
  listFeed,
  listFollowingFeed,
  currentUserId,
  subscribePostCreated,
  type FeedPost,
} from '@/lib/posts'
import { CreatePost } from '@/components/feed/create-post'
import { PostCard } from '@/components/feed/post-card'
import { useT } from '@/components/language-provider'

type FeedTab = 'for-you' | 'following'

const FEED_PAGE = 10

/** Concatène une page en dédupliquant par id (un post prépendu peut revenir). */
function mergeUnique(current: FeedPost[], incoming: FeedPost[]): FeedPost[] {
  const seen = new Set(current.map((p) => p.id))
  return [...current, ...incoming.filter((p) => !seen.has(p.id))]
}

function applyPostUpdate(current: FeedPost[], updated: FeedPost): FeedPost[] {
  return current.map((post) => {
    if (post.id === updated.id) return updated
    if (updated.isPinned && post.author.id === updated.author.id) {
      return { ...post, isPinned: false, pinnedAt: '' }
    }
    return post
  })
}

/**
 * Corps du fil d'actualité : en-tête sticky, onglets « Pour toi » /
 * « Abonnements », zone de composition, puis la liste de l'onglet actif.
 *
 * Les pages sont chargées au défilement (`useInfiniteScroll`). Un post
 * fraîchement publié est prépendu sans refetch (event `post-created`).
 */
export function FeedView() {
  const t = useT()
  const [tab, setTab] = useState<FeedTab>('for-you')
  const [posts, setPosts] = useState<FeedPost[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const [error, setError] = useState('')
  const [mutedWords, setMutedWords] = useState<string[]>([])
  const [viewerUserId, setViewerUserId] = useState('')
  // Nombre d'éléments réellement chargés depuis le serveur (offset de pagination,
  // indépendant des insertions/suppressions locales).
  const offsetRef = useRef(0)

  const fetchPage = useCallback(
    (activeTab: FeedTab, offset: number) =>
      activeTab === 'for-you'
        ? listFeed(FEED_PAGE, offset)
        : listFollowingFeed(FEED_PAGE, offset),
    [],
  )

  // Chargement initial / changement d'onglet.
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    setPosts([])
    offsetRef.current = 0

    fetchPage(tab, 0)
      .then((list) => {
        if (cancelled) return
        setPosts(list)
        offsetRef.current = list.length
        setHasMore(list.length === FEED_PAGE)
      })
      .catch(() => {
        if (!cancelled) setError(t('feed.load_error'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [tab, fetchPage, t])

  const loadMore = useCallback(async () => {
    setLoadingMore(true)
    try {
      const next = await fetchPage(tab, offsetRef.current)
      offsetRef.current += next.length
      setPosts((prev) => mergeUnique(prev, next))
      setHasMore(next.length === FEED_PAGE)
    } catch {
      setHasMore(false)
    } finally {
      setLoadingMore(false)
    }
  }, [tab, fetchPage])

  const sentinelRef = useInfiniteScroll(loadMore, {
    hasMore,
    loading: loading || loadingMore,
  })

  useEffect(() => {
    let cancelled = false
    let unsubscribe = () => {}

    async function loadUserScopedFilters() {
      let userId = currentUserId()
      if (!userId) {
        try {
          userId = (await getMe()).id
        } catch {
          userId = ''
        }
      }

      if (cancelled) return
      setViewerUserId(userId)
      setMutedWords(readMutedWords(userId))
      unsubscribe = subscribeMutedWords(userId, setMutedWords)
    }

    void loadUserScopedFilters()
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  // Un nouveau post (composer inline ou popup sidebar) est prépendu au fil.
  useEffect(
    () =>
      subscribePostCreated((post) =>
        setPosts((prev) =>
          prev.some((p) => p.id === post.id)
            ? applyPostUpdate(prev, post)
            : [post, ...prev],
        ),
      ),
    [],
  )

  const handleDeleted = useCallback((id: string) => {
    setPosts((prev) => prev.filter((p) => p.id !== id))
  }, [])

  const handleUpdated = useCallback((post: FeedPost) => {
    setPosts((prev) => applyPostUpdate(prev, post))
  }, [])

  const visiblePosts = filterMutedPosts(posts, mutedWords, viewerUserId)

  return (
    <div className="flex flex-col">
      {/* En-tête : sticky sur desktop ; sur mobile l'en-tête global (logo) prend le relais */}
      <div className="panel z-10 border-b lg:sticky lg:top-0">
        <h1 className="brand-text hidden px-4 py-3 text-xl font-bold lg:block">
          {t('feed.title')}
        </h1>
        {/* Onglets Pour toi / Abonnements */}
        <div className="flex">
          <TabButton active={tab === 'for-you'} onClick={() => setTab('for-you')}>
            {t('feed.tab_for_you')}
          </TabButton>
          <TabButton active={tab === 'following'} onClick={() => setTab('following')}>
            {t('feed.tab_following')}
          </TabButton>
        </div>
      </div>

      {/* Zone de création de post inline ; le FAB mobile prend le relais quand ce bloc sort de l'écran. */}
      <div id="feed-composer">
        <CreatePost />
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF]" aria-hidden />
        </div>
      ) : error ? (
        <EmptyState title={t('feed.unavailable')} message={error} />
      ) : posts.length === 0 ? (
        tab === 'for-you' ? (
          <EmptyState
            title={t('feed.empty_title')}
            message={t('feed.empty_for_you')}
          />
        ) : (
          <EmptyState
            title={t('feed.empty_title')}
            message={t('feed.empty_following')}
          />
        )
      ) : visiblePosts.length === 0 ? (
        <EmptyState
          title={t('feed.filtered_empty_title')}
          message={t('feed.filtered_empty_msg')}
        />
      ) : (
        <>
          <div className="divide-y divide-border">
            {visiblePosts.map((post) => (
              <PostCard
                key={post.id}
                post={post}
                onDeleted={handleDeleted}
                onUpdated={handleUpdated}
              />
            ))}
          </div>
          {/* Sentinelle de défilement infini + indicateur de chargement. */}
          {hasMore && (
            <div ref={sentinelRef} className="flex justify-center py-6">
              {loadingMore && (
                <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF]" aria-hidden />
              )}
            </div>
          )}
        </>
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

function EmptyState({ title, message }: { title: string; message: string }) {
  return (
    <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
      <Users className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
      <h2 className="text-lg font-bold">{title}</h2>
      <p className="max-w-sm text-sm text-muted-foreground">{message}</p>
    </div>
  )
}
