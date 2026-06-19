'use client'

import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import { ArrowLeft, ImageIcon, Loader2, Search, Users } from 'lucide-react'

import { cn, initialOf } from '@/lib/utils'
import { getAccessToken } from '@/lib/auth-client'
import {
  filterMutedPosts,
  readMutedWords,
  subscribeMutedWords,
} from '@/lib/content-filters'
import { getBlockedUserIds, getFollowingIds } from '@/lib/api'
import { subscribeProfilUpdated } from '@/lib/profil-client'
import { useInfiniteScroll } from '@/lib/use-infinite-scroll'
import {
  applyProfilUpdateToPosts,
  applyStatsToPosts,
  connectFeedRealtime,
  getPostAuthor,
  listFeed,
  listFollowingFeed,
  currentUserId,
  subscribePostCreated,
  type FeedPost,
  type HashtagPostSort,
  type NewPostPing,
  type PostAuthor,
  type PostMedia,
} from '@/lib/posts'
import { usePostStatsPolling } from '@/lib/use-post-stats-polling'
import { ROUTES, hashtagHref, postHref, searchHref } from '@/lib/routes'
import { BLOCK_CHANGE_EVENT, type BlockChangeDetail } from '@/lib/use-block'
import { CreatePost } from '@/components/feed/create-post'
import { PostCard } from '@/components/feed/post-card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useT } from '@/components/language-provider'
import { SearchSuggestionsDropdown } from '@/components/search/search-suggestions-dropdown'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'

type FeedTab = 'for-you' | 'following'
type HashtagTab = 'top' | 'recent' | 'media'

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

/** Fil d'actualité (onglets « Pour toi » / « Abonnements »), paginé au défilement ;
 *  un post fraîchement publié est prépendu sans refetch. */
export function FeedView() {
  const t = useT()
  const { isVisitor } = useAuthGate()
  const router = useRouter()
  const searchParams = useSearchParams()
  // Le feed reste monté en permanence (layout) derrière les overlays des autres
  // sections. Hors `/feed`, on **gèle** sa lecture des query params (`hashtag`,
  // `tab`) : sinon, naviguer vers `/profil` etc. les remettrait à zéro et
  // déclencherait un refetch invisible, perdant le contexte hashtag et la
  // position. On conserve donc la dernière valeur vue sur `/feed`.
  const pathname = usePathname()
  const onFeed = pathname === ROUTES.feed
  const rawHashtag = (searchParams.get('hashtag') ?? '').trim().replace(/^#/, '')
  const rawHashtagTab = normalizeHashtagTab(searchParams.get('tab'))
  const frozenHashtag = useRef(rawHashtag)
  const frozenHashtagTab = useRef(rawHashtagTab)
  if (onFeed) {
    frozenHashtag.current = rawHashtag
    frozenHashtagTab.current = rawHashtagTab
  }
  const selectedHashtag = onFeed ? rawHashtag : frozenHashtag.current
  const hashtagTab = onFeed ? rawHashtagTab : frozenHashtagTab.current
  const [hashtagInput, setHashtagInput] = useState(selectedHashtag ? `#${selectedHashtag}` : '')
  const [tab, setTab] = useState<FeedTab>('for-you')
  const [posts, setPosts] = useState<FeedPost[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const [error, setError] = useState('')
  const [mutedWords, setMutedWords] = useState<string[]>([])
  const [viewerUserId, setViewerUserId] = useState('')
  const [searchFocused, setSearchFocused] = useState(false)
  // Nombre d'éléments réellement chargés depuis le serveur (offset de pagination,
  // indépendant des insertions/suppressions locales).
  const offsetRef = useRef(0)

  // Temps réel : posts publiés par d'AUTRES pendant qu'on consulte le fil. On
  // n'affiche pas leur contenu tout de suite (façon X) ; on accumule un « ping »
  // par post et on présente un bandeau « a posté » cliquable qui révèle les
  // nouveautés et remonte en haut.
  const [pendingPings, setPendingPings] = useState<NewPostPing[]>([])
  const [bannerAuthor, setBannerAuthor] = useState<PostAuthor | null>(null)
  // Ensemble des comptes suivis (filtre du bandeau pour l'onglet « Abonnements »).
  const followingIdsRef = useRef<Set<string>>(new Set())
  const blockedIdsRef = useRef<Set<string>>(new Set())
  // Miroirs des valeurs courantes pour le callback WS (monté une seule fois).
  const tabRef = useRef(tab)
  const selectedHashtagRef = useRef(selectedHashtag)
  const viewerRef = useRef(viewerUserId)
  const postsRef = useRef(posts)
  const pendingRef = useRef(pendingPings)
  tabRef.current = tab
  selectedHashtagRef.current = selectedHashtag
  viewerRef.current = viewerUserId
  postsRef.current = posts
  pendingRef.current = pendingPings

  const fetchPage = useCallback(
    (activeTab: FeedTab, offset: number) => {
      if (selectedHashtag) {
        return listFeed(
          FEED_PAGE,
          offset,
          selectedHashtag,
          hashtagTab === 'top' ? 'top' : 'recent',
        )
      }
      return activeTab === 'for-you'
        ? listFeed(FEED_PAGE, offset, selectedHashtag)
        : listFollowingFeed(FEED_PAGE, offset, selectedHashtag)
    },
    [hashtagTab, selectedHashtag],
  )

  // Chargement initial / changement d'onglet.
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    setPosts([])
    offsetRef.current = 0
    // Nouveau contexte de fil → on repart d'un bandeau vide.
    setPendingPings([])
    setBannerAuthor(null)

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

  const clearHashtagFilter = useCallback(() => {
    router.push(ROUTES.feed)
  }, [router])

  useEffect(() => {
    setHashtagInput(selectedHashtag ? `#${selectedHashtag}` : '')
  }, [selectedHashtag])

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

  // Révèle les posts en attente : refetch la 1re page de l'onglet courant,
  // prépend les nouveautés (dédup) et remonte en haut. Le contenu vient du fil
  // authentifié normal (visibilité server-side), pas du ping WebSocket.
  const revealPending = useCallback(async () => {
    setPendingPings([])
    setBannerAuthor(null)
    try {
      const fresh = await fetchPage(tab, 0)
      setPosts((prev) => {
        const seen = new Set(prev.map((p) => p.id))
        const fresher = fresh.filter((p) => !seen.has(p.id))
        offsetRef.current += fresher.length
        return [...fresher, ...prev]
      })
    } catch {
      // best-effort : on garde le fil courant
    }
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }, [fetchPage, tab])

  // Charge l'ensemble des comptes suivis (filtre du bandeau en onglet
  // « Abonnements » : on n'annonce que les posts d'auteurs suivis).
  useEffect(() => {
    if (!viewerUserId) return
    let cancelled = false
    void Promise.all([getFollowingIds(viewerUserId), getBlockedUserIds()])
      .then(([followingIds, blockedIds]) => {
        if (!cancelled) {
          followingIdsRef.current = followingIds
          blockedIdsRef.current = blockedIds
        }
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [viewerUserId])

  useEffect(() => {
    function onBlockChange(event: Event) {
      const { userId, blocked } = (event as CustomEvent<BlockChangeDetail>).detail
      if (blocked) {
        blockedIdsRef.current.add(userId)
        setPendingPings((prev) => prev.filter((ping) => ping.authorId !== userId))
        setPosts((prev) => prev.filter((post) => post.author.id !== userId))
        if (bannerAuthor?.id === userId) setBannerAuthor(null)
        return
      }
      blockedIdsRef.current.delete(userId)
    }
    window.addEventListener(BLOCK_CHANGE_EVENT, onBlockChange)
    return () => window.removeEventListener(BLOCK_CHANGE_EVENT, onBlockChange)
  }, [bannerAuthor?.id])

  // Connexion WebSocket du fil (montée une seule fois ; visiteur exclu). À chaque
  // « ping », on filtre via les refs (pas mes posts, pas de filtre hashtag actif,
  // pas déjà présent/en attente, et auteur suivi en onglet « Abonnements »), puis
  // on empile le ping et on résout l'auteur du bandeau.
  useEffect(() => {
    if (isVisitor || !getAccessToken()) return
    const handle = connectFeedRealtime((ping: NewPostPing) => {
      if (selectedHashtagRef.current) return
      if (ping.authorId === viewerRef.current) return
      if (blockedIdsRef.current.has(ping.authorId)) return
      if (postsRef.current.some((p) => p.id === ping.postId)) return
      if (pendingRef.current.some((p) => p.postId === ping.postId)) return
      if (tabRef.current === 'following' && !followingIdsRef.current.has(ping.authorId)) return

      setPendingPings((prev) =>
        prev.some((p) => p.postId === ping.postId) ? prev : [ping, ...prev],
      )
      void getPostAuthor(ping.authorId)
        .then(setBannerAuthor)
        .catch(() => {})
    })
    return () => handle.close()
  }, [isVisitor])

  useEffect(() => {
    // Visiteur : `currentUserId()` rend '' → pas de filtres par utilisateur.
    const userId = currentUserId()
    setViewerUserId(userId)
    setMutedWords(readMutedWords(userId))
    return subscribeMutedWords(userId, setMutedWords)
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

  useEffect(
    () =>
      subscribeProfilUpdated((profil) => {
        setPosts((prev) => applyProfilUpdateToPosts(prev, profil))
      }),
    [],
  )

  // Compteurs dynamiques : refetch périodique des likes/commentaires/reposts des
  // posts affichés (façon X), sans toucher l'état « moi » (liked/reposted local).
  usePostStatsPolling(
    () => postsRef.current.map((p) => p.id),
    (stats) => setPosts((prev) => applyStatsToPosts(prev, stats)),
  )

  const handleDeleted = useCallback((id: string) => {
    setPosts((prev) => prev.filter((p) => p.id !== id))
  }, [])

  const handleUpdated = useCallback((post: FeedPost) => {
    setPosts((prev) => applyPostUpdate(prev, post))
  }, [])

  const visiblePosts = filterMutedPosts(posts, mutedWords, viewerUserId)
  const visibleMedia = hashtagTab === 'media' ? mediaFromPosts(visiblePosts) : []

  function submitHashtagSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const query = hashtagInput.trim()
    if (!query) {
      clearHashtagFilter()
      return
    }
    router.push(searchHref(query))
  }

  function pickSearchSuggestion(href: string) {
    setSearchFocused(false)
    router.push(href)
  }

  function setHashtagTab(next: HashtagTab) {
    if (!selectedHashtag) return
    router.push(hashtagHref(selectedHashtag, next))
  }

  return (
    <div className="flex flex-col">
      {/* En-tête : sticky sur desktop ; sur mobile l'en-tête global (logo) prend le relais */}
      <div className="panel z-30 border-b lg:sticky lg:top-0">
        {selectedHashtag ? (
          <>
            <form onSubmit={submitHashtagSearch} className="flex items-center gap-2 px-4 py-3">
              <button
                type="button"
                onClick={clearHashtagFilter}
                className="grid h-10 w-10 shrink-0 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                aria-label={t('feed.hashtag_back')}
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <div className="relative min-w-0 flex-1">
                <Search
                  className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
                  aria-hidden
                />
                <input
                  type="search"
                  value={hashtagInput}
                  onChange={(event) => setHashtagInput(event.target.value)}
                  onFocus={() => setSearchFocused(true)}
                  onBlur={() => setSearchFocused(false)}
                  aria-label={t('feed.hashtag_search_aria')}
                  placeholder={t('feed.hashtag_search_placeholder')}
                  className="glass w-full rounded-full border py-2.5 pl-10 pr-4 text-sm font-semibold outline-none transition focus:border-[#5B6CFF] focus:bg-white dark:focus:bg-white/10"
                />
                <SearchSuggestionsDropdown
                  query={hashtagInput}
                  open={searchFocused}
                  onPick={pickSearchSuggestion}
                />
              </div>
            </form>
            <div className="flex">
              <TabButton active={hashtagTab === 'top'} onClick={() => setHashtagTab('top')}>
                {t('feed.hashtag_tab_top')}
              </TabButton>
              <TabButton active={hashtagTab === 'recent'} onClick={() => setHashtagTab('recent')}>
                {t('feed.hashtag_tab_recent')}
              </TabButton>
              <TabButton active={hashtagTab === 'media'} onClick={() => setHashtagTab('media')}>
                {t('feed.hashtag_tab_media')}
              </TabButton>
            </div>
          </>
        ) : (
          <>
            <h1 className="brand-text hidden px-4 py-3 text-xl font-bold lg:block">
              {t('feed.title')}
            </h1>
            {/* Onglets Pour toi / Abonnements (« Abonnements » requiert une session). */}
            <div className="flex">
              <TabButton active={tab === 'for-you'} onClick={() => setTab('for-you')}>
                {t('feed.tab_for_you')}
              </TabButton>
              {!isVisitor && (
                <TabButton active={tab === 'following'} onClick={() => setTab('following')}>
                  {t('feed.tab_following')}
                </TabButton>
              )}
            </div>
          </>
        )}
      </div>

      {/* Bandeau temps réel « a posté » : flotte sous l'en-tête dès qu'un autre
          utilisateur a publié. Clic → révèle les nouveautés et remonte en haut.
          (wrapper transparent aux clics, seul le bouton les capte). */}
      {!selectedHashtag && bannerAuthor && pendingPings.length > 0 && (
        <div className="pointer-events-none sticky top-2 z-20 flex justify-center lg:top-[60px]">
          <button
            type="button"
            onClick={revealPending}
            className="pointer-events-auto flex items-center gap-2 rounded-full border border-white/20 bg-[#5B6CFF] px-4 py-1.5 text-sm font-semibold text-white shadow-lg transition hover:brightness-110"
          >
            <Avatar className="h-6 w-6">
              {bannerAuthor.avatarUrl && <AvatarImage src={bannerAuthor.avatarUrl} alt="" />}
              <AvatarFallback className="bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-[10px] font-bold text-white">
                {initialOf(bannerAuthor.displayName)}
              </AvatarFallback>
              <ActivityPresenceDot
                userId={bannerAuthor.id}
                initialLastLoginAt={bannerAuthor.lastLoginAt}
                initialIsOnline={bannerAuthor.isOnline}
                className="h-2 w-2 border"
              />
            </Avatar>
            <span>{t('feed.new_posts')}</span>
          </button>
        </div>
      )}

      {/* Zone de création de post inline (masquée pour le visiteur) ; le FAB
          mobile prend le relais quand ce bloc sort de l'écran. */}
      {!isVisitor && !selectedHashtag && (
        <div id="feed-composer">
          <CreatePost />
        </div>
      )}

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
      ) : selectedHashtag && hashtagTab === 'media' ? (
        <>
          {visibleMedia.length === 0 ? (
            <EmptyState title={t('feed.hashtag_no_media_title')} message={t('feed.hashtag_no_media_msg')} />
          ) : (
            <HashtagMediaGrid media={visibleMedia} openLabel={t('feed.hashtag_media_open')} />
          )}
          {hasMore && (
            <div ref={sentinelRef} className="flex justify-center py-6">
              {loadingMore && (
                <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF]" aria-hidden />
              )}
            </div>
          )}
        </>
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

function normalizeHashtagTab(value: string | null): HashtagTab {
  return value === 'recent' || value === 'media' ? value : 'top'
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

function mediaFromPosts(posts: FeedPost[]): Array<PostMedia & { postId: string }> {
  return posts.flatMap((post) => post.media.map((media) => ({ ...media, postId: post.id })))
}

function HashtagMediaGrid({
  media,
  openLabel,
}: {
  media: Array<PostMedia & { postId: string }>
  openLabel: string
}) {
  return (
    <div className="grid grid-cols-3 gap-1 p-1 sm:gap-1.5 sm:p-2">
      {media.map((item, index) => (
        <a
          key={`${item.postId}-${item.url}-${index}`}
          href={postHref(item.postId)}
          className="group relative aspect-square overflow-hidden bg-muted"
          aria-label={openLabel}
        >
          {item.type === 'video' ? (
            <>
              <video src={item.url} muted playsInline className="h-full w-full object-cover" />
              <ImageIcon
                className="absolute right-2 top-2 h-4 w-4 rounded-full bg-background/70 p-0.5 text-foreground"
                aria-hidden
              />
            </>
          ) : (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={item.url}
              alt=""
              loading="lazy"
              className="h-full w-full object-cover transition group-hover:scale-[1.02]"
            />
          )}
        </a>
      ))}
    </div>
  )
}
