'use client'

import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import Link from 'next/link'
import { useRouter, useSearchParams } from 'next/navigation'
import { ArrowRight, Hash, Loader2, Search, Sparkles, UserX } from 'lucide-react'

import {
  getCommonFollowers,
  getFollowingIds,
  getSuggestions,
  searchUsers,
} from '@/lib/api'
import {
  currentUserId as readCurrentUserId,
  listFeed,
  listHashtagTrends,
  type FeedPost,
  type HashtagTrend,
} from '@/lib/posts'
import { hashtagHref } from '@/lib/routes'
import { useFollow } from '@/lib/use-follow'
import type { RelationUser } from '@/types'
import { cn } from '@/lib/utils'
import { useExplorerFilters } from '@/components/explorer/explorer-filter-context'
import { MobileFilterMenu } from '@/components/explorer/explorer-filter-controls'
import { useT } from '@/components/language-provider'
import { PostCard } from '@/components/feed/post-card'
import { ProfilLink } from '@/components/profil/profil-link'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'

/**
 * Explorer combine découverte par tendances, suggestions de comptes et
 * publications du feed. La recherche garde deux sections, tendances puis
 * personnes.
 */
export function ExplorerView() {
  const t = useT()
  const router = useRouter()
  const searchParams = useSearchParams()
  const [query, setQuery] = useState(() => searchParams.get('q') ?? '')
  const [submittedQuery, setSubmittedQuery] = useState(() => searchParams.get('q') ?? '')
  const [debounced, setDebounced] = useState('')
  const [trends, setTrends] = useState<HashtagTrend[]>([])
  const [searchTrends, setSearchTrends] = useState<HashtagTrend[]>([])
  const [suggestions, setSuggestions] = useState<RelationUser[]>([])
  const [commonFollowers, setCommonFollowers] = useState<Record<string, RelationUser[]>>({})
  const [searchUsersList, setSearchUsersList] = useState<RelationUser[]>([])
  const [publicationPosts, setPublicationPosts] = useState<FeedPost[]>([])
  const [resultPosts, setResultPosts] = useState<FeedPost[]>([])
  const [loadingResultPosts, setLoadingResultPosts] = useState(false)
  const [loadingDefault, setLoadingDefault] = useState(true)
  const [loadingSearch, setLoadingSearch] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { currentUserId, isFollowing, isPending, toggle } = useFollow()
  const {
    setFilters,
    setSubmittedSearchActive,
    showPublications,
    showUsers,
    togglePublications,
    toggleUsers,
  } = useExplorerFilters()
  const viewerId = currentUserId ?? readCurrentUserId()

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 260)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    const nextQuery = searchParams.get('q') ?? ''
    setQuery(nextQuery)
    setSubmittedQuery(nextQuery)
    setSubmittedSearchActive(Boolean(nextQuery))
  }, [searchParams, setSubmittedSearchActive])

  useEffect(() => {
    return () => setSubmittedSearchActive(false)
  }, [setSubmittedSearchActive])

  useEffect(() => {
    let cancelled = false
    setLoadingDefault(true)
    setError(null)
    Promise.all([
      listHashtagTrends(10).catch(() => []),
      getSuggestions(12).catch(() => []),
      listFeed(10, 0).catch(() => []),
      viewerId
        ? getFollowingIds(viewerId).catch(() => new Set<string>())
        : Promise.resolve(new Set<string>()),
    ])
      .then(([nextTrends, nextSuggestions, nextPosts, followingIds]) => {
        if (cancelled) return
        const visibleSuggestions = nextSuggestions
          .filter((user) => user.id !== viewerId && !followingIds.has(user.id))
          .slice(0, 6)
        setTrends(nextTrends)
        setSuggestions(visibleSuggestions)
        setPublicationPosts(nextPosts)
      })
      .catch(() => {
        if (!cancelled) setError(t('explorer.search_failed_title'))
      })
      .finally(() => {
        if (!cancelled) setLoadingDefault(false)
      })
    return () => {
      cancelled = true
    }
  }, [t, viewerId])

  useEffect(() => {
    if (suggestions.length === 0) {
      setCommonFollowers({})
      return
    }
    let cancelled = false
    Promise.all(
      suggestions.map(async (user) => [user.id, await getCommonFollowers(user.id, 2).catch(() => [])] as const),
    ).then((pairs) => {
      if (cancelled) return
      setCommonFollowers(Object.fromEntries(pairs))
    })
    return () => {
      cancelled = true
    }
  }, [suggestions])

  useEffect(() => {
    if (!debounced) {
      setSearchTrends([])
      setSearchUsersList([])
      setLoadingSearch(false)
      return
    }

    let cancelled = false
    const term = debounced.replace(/^#/, '')
    setLoadingSearch(true)
    setError(null)
    Promise.all([
      listHashtagTrends(3, term).catch(() => []),
      searchUsers(debounced.startsWith('#') ? term : debounced).catch(() => []),
    ])
      .then(([nextTrends, nextUsers]) => {
        if (cancelled) return
        setSearchTrends(nextTrends)
        setSearchUsersList(nextUsers.filter((user) => user.id !== viewerId))
      })
      .catch(() => {
        if (!cancelled) setError(t('explorer.search_failed_title'))
      })
      .finally(() => {
        if (!cancelled) setLoadingSearch(false)
      })
    return () => {
      cancelled = true
    }
  }, [debounced, t, viewerId])

  useEffect(() => {
    if (!submittedQuery) {
      setResultPosts([])
      setLoadingResultPosts(false)
      return
    }
    let cancelled = false
    setLoadingResultPosts(true)
    listFeed(50, 0)
      .then((posts) => {
        if (cancelled) return
        setResultPosts(posts.filter((post) => postMatchesQuery(post, submittedQuery)))
      })
      .catch(() => {
        if (!cancelled) setResultPosts([])
      })
      .finally(() => {
        if (!cancelled) setLoadingResultPosts(false)
      })
    return () => {
      cancelled = true
    }
  }, [submittedQuery])

  function submitSearch(nextValue = query, filters = { publications: true, users: false }) {
    const term = nextValue.trim()
    if (!term) return
    setSubmittedQuery(term)
    setFilters(filters)
    setSubmittedSearchActive(true)
    router.replace(`/explorer?q=${encodeURIComponent(term)}`, { scroll: false })
  }

  function handleSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    submitSearch()
  }

  function handleGoToUser(term: string) {
    const clean = term.trim().replace(/^@/, '')
    const exact = searchUsersList.find((user) => normalizeSearch(user.username) === normalizeSearch(clean))
    if (exact) {
      router.push(`/profil/${exact.username}`)
      return
    }
    submitSearch(clean, { publications: false, users: true })
  }

  function handlePostDeleted(id: string) {
    setPublicationPosts((prev) => prev.filter((post) => post.id !== id))
    setResultPosts((prev) => prev.filter((post) => post.id !== id))
  }

  function handlePostUpdated(post: FeedPost) {
    setPublicationPosts((prev) => prev.map((item) => (item.id === post.id ? post : item)))
    setResultPosts((prev) => prev.map((item) => (item.id === post.id ? post : item)))
  }

  return (
    <div className="flex flex-col">
      <div className="panel z-10 border-b px-4 py-3 lg:sticky lg:top-0">
        <h1 className="brand-text mb-3 text-xl font-bold">{t('nav.explore')}</h1>
        <form onSubmit={handleSearchSubmit} className="relative">
          <div className="relative">
            <Search
              className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
              aria-hidden
            />
            <input
              type="search"
              value={query}
              onChange={(event) => {
                setQuery(event.target.value)
                if (submittedQuery && event.target.value.trim() !== submittedQuery) {
                  setSubmittedQuery('')
                  setSubmittedSearchActive(false)
                }
              }}
              placeholder={t('explorer.search_placeholder')}
              autoFocus
              className="glass w-full rounded-full border py-2.5 pl-10 pr-4 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:bg-white focus:outline-none dark:focus:bg-white/10"
            />
          </div>
          {debounced && !submittedQuery && (
            <LiveSearchMenu
              query={debounced}
              trends={searchTrends}
              users={searchUsersList}
              loading={loadingSearch}
              onSearch={() => submitSearch(debounced)}
              onGoToUser={() => handleGoToUser(debounced)}
            />
          )}
        </form>
        <p className="mt-2 px-1 text-xs text-muted-foreground">
          {t('explorer.hint_before')}{' '}
          <span className="font-bold">@</span> / <span className="font-bold">#</span>{' '}
          {t('explorer.hint_after')}
        </p>
        {submittedQuery && (
          <div className="mt-3 flex justify-end xl:hidden">
            <MobileFilterMenu
              showPublications={showPublications}
              showUsers={showUsers}
              onTogglePublications={togglePublications}
              onToggleUsers={toggleUsers}
            />
          </div>
        )}
      </div>

      {submittedQuery ? (
        <SubmittedSearchResults
          query={submittedQuery}
          posts={resultPosts}
          users={searchUsersList}
          loadingPosts={loadingResultPosts}
          loadingUsers={loadingSearch}
          error={error}
          currentUserId={viewerId}
          showPublications={showPublications}
          showUsers={showUsers}
          isFollowing={isFollowing}
          isPending={isPending}
          onToggleFollow={toggle}
          onDeleted={handlePostDeleted}
          onUpdated={handlePostUpdated}
        />
      ) : loadingDefault ? (
        <LoadingBlock />
      ) : error ? (
        <EmptyState
          icon={<UserX className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
          title={t('explorer.search_failed_title')}
          message={t('explorer.search_failed_msg')}
        />
      ) : (
        <>
          <TrendsSection trends={trends} />
          <SuggestionsSection
            users={suggestions}
            currentUserId={viewerId}
            commonFollowers={commonFollowers}
            isFollowing={isFollowing}
            isPending={isPending}
            onToggleFollow={toggle}
          />
          <PublicationsSection
            posts={publicationPosts}
            onDeleted={handlePostDeleted}
            onUpdated={handlePostUpdated}
          />
        </>
      )}
    </div>
  )
}

function LiveSearchMenu({
  query,
  trends,
  users,
  loading,
  onSearch,
  onGoToUser,
}: {
  query: string
  trends: HashtagTrend[]
  users: RelationUser[]
  loading: boolean
  onSearch: () => void
  onGoToUser: () => void
}) {
  const t = useT()
  const hasResults = trends.length > 0 || users.length > 0
  return (
    <div className="absolute left-0 right-0 top-[calc(100%+0.35rem)] z-30 overflow-hidden rounded-[24px] border border-border bg-background shadow-[0_24px_80px_rgba(0,0,0,0.35)] ring-1 ring-black/5 dark:bg-[#05070b] dark:shadow-[0_24px_90px_rgba(0,0,0,0.75)]">
      {loading ? (
        <div className="flex justify-center py-5">
          <Loader2 className="h-5 w-5 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
        </div>
      ) : (
        <>
          {!hasResults && (
            <>
              <button
                type="button"
                onClick={onSearch}
                className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-accent"
              >
                <Search className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
                <span className="min-w-0">
                  <span className="block font-bold text-foreground">
                    {t('explorer.action_search', { q: query })}
                  </span>
                  <span className="block truncate text-sm text-muted-foreground">
                    {t('explorer.action_search_hint')}
                  </span>
                </span>
              </button>
              <button
                type="button"
                onClick={onGoToUser}
                className="flex w-full items-center gap-3 border-t border-border px-4 py-3 text-left transition-colors hover:bg-accent"
              >
                <ArrowRight className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
                <span className="min-w-0">
                  <span className="block font-bold text-foreground">
                    {t('explorer.action_go_user', { q: query.replace(/^@/, '') })}
                  </span>
                  <span className="block truncate text-sm text-muted-foreground">
                    {t('explorer.action_go_user_hint')}
                  </span>
                </span>
              </button>
            </>
          )}
          {trends.length > 0 && (
            <div className={cn(users.length > 0 && 'border-b border-border')}>
              <p className="px-4 pt-3 text-xs font-bold uppercase tracking-wide text-muted-foreground">
                {t('explorer.search_trends')}
              </p>
              {trends.map((trend) => (
                <Link
                  key={trend.tag}
                  href={hashtagHref(trend.tag, 'top')}
                  className="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-accent"
                >
                  <Hash className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
                  <span className="min-w-0">
                    <span className="block truncate font-bold text-foreground">#{trend.tag}</span>
                    <span className="block text-sm text-muted-foreground">
                      {t(trend.count > 1 ? 'trends.posts_other' : 'trends.posts_one', {
                        count: formatTrendCount(trend.count),
                      })}
                    </span>
                  </span>
                </Link>
              ))}
            </div>
          )}
          {users.length > 0 && (
            <div>
              <p className="px-4 pt-3 text-xs font-bold uppercase tracking-wide text-muted-foreground">
                {t('explorer.search_people')}
              </p>
              {users.slice(0, 5).map((user) => (
                <ProfilLink
                  key={user.id}
                  author={{ id: user.id, username: user.username }}
                  className="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-accent"
                >
                  <Avatar className="h-10 w-10">
                    {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
                    <AvatarFallback>{initials(user)}</AvatarFallback>
                  </Avatar>
                  <span className="min-w-0">
                    <span className="block truncate font-bold text-foreground">{user.displayName}</span>
                    <span className="block truncate text-sm text-muted-foreground">@{user.username}</span>
                  </span>
                </ProfilLink>
              ))}
            </div>
          )}
          {!hasResults && (
            <p className="border-t border-border px-4 py-3 text-sm text-muted-foreground">
              {t('explorer.no_results_for', { q: query })}
            </p>
          )}
        </>
      )}
    </div>
  )
}

function SubmittedSearchResults({
  query,
  posts,
  users,
  loadingPosts,
  loadingUsers,
  error,
  currentUserId,
  showPublications,
  showUsers,
  isFollowing,
  isPending,
  onToggleFollow,
  onDeleted,
  onUpdated,
}: {
  query: string
  posts: FeedPost[]
  users: RelationUser[]
  loadingPosts: boolean
  loadingUsers: boolean
  error: string | null
  currentUserId: string | null
  showPublications: boolean
  showUsers: boolean
  isFollowing: (id: string) => boolean
  isPending: (id: string) => boolean
  onToggleFollow: (user: RelationUser, next: boolean) => void
  onDeleted: (id: string) => void
  onUpdated: (post: FeedPost) => void
}) {
  const t = useT()
  if (error) {
    return (
      <EmptyState
        icon={<UserX className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
        title={t('explorer.search_failed_title')}
        message={t('explorer.search_failed_msg')}
      />
    )
  }
  const visibleUsers = users.filter((user) => user.id !== currentUserId)
  const nothingVisible =
    (!showPublications || (!loadingPosts && posts.length === 0)) &&
    (!showUsers || (!loadingUsers && visibleUsers.length === 0))
  return (
    <>
      <ResultsHeader query={query} />
      {showPublications && (
        <Section title={t('explorer.filter_publications')}>
          {loadingPosts ? (
            <LoadingBlock />
          ) : posts.length === 0 ? (
            <p className="px-4 py-3 text-sm text-muted-foreground">
              {t('explorer.no_publications_for', { q: query })}
            </p>
          ) : (
            <div className="divide-y divide-border">
              {posts.map((post) => (
                <PostCard key={post.id} post={post} onDeleted={onDeleted} onUpdated={onUpdated} />
              ))}
            </div>
          )}
        </Section>
      )}
      {showUsers && (
        <Section title={t('explorer.filter_users')}>
          {loadingUsers ? (
            <LoadingBlock />
          ) : visibleUsers.length === 0 ? (
            <p className="px-4 py-3 text-sm text-muted-foreground">
              {t('explorer.no_users_for', { q: query })}
            </p>
          ) : (
            visibleUsers.map((user) => (
              <ExplorerUserRow
                key={user.id}
                user={user}
                isSelf={user.id === currentUserId}
                isFollowing={isFollowing(user.id)}
                pending={isPending(user.id)}
                onToggleFollow={onToggleFollow}
              />
            ))
          )}
        </Section>
      )}
      {nothingVisible && (
        <EmptyState
          icon={<Search className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
          title={t('explorer.no_results')}
          message={t('explorer.no_results_for', { q: query })}
        />
      )}
    </>
  )
}

function ResultsHeader({
  query,
}: {
  query: string
}) {
  const t = useT()
  return (
    <section className="border-b border-border px-4 py-3">
      <div className="flex items-center gap-2">
        <Sparkles className="h-4 w-4 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
        <p className="brand-text truncate text-lg font-bold">
          {t('explorer.results_for', { q: query })}
        </p>
      </div>
    </section>
  )
}

function TrendsSection({
  trends,
  compact = false,
  title,
}: {
  trends: HashtagTrend[]
  compact?: boolean
  title?: string
}) {
  const t = useT()
  return (
    <Section title={title ?? t('explorer.trends_title')}>
      {trends.length === 0 ? (
        <p className="px-4 py-3 text-sm text-muted-foreground">{t('trends.empty')}</p>
      ) : (
        <div className={cn('divide-y divide-border', compact && 'border-t border-border/60')}>
          {trends.map((trend, index) => (
            <Link
              key={trend.tag}
              href={hashtagHref(trend.tag, 'top')}
              className="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-accent"
            >
              <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-[#5B6CFF]/10 text-[#5B6CFF] dark:bg-[#9aa6ff]/15 dark:text-[#9aa6ff]">
                <Hash className="h-5 w-5" aria-hidden />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block text-xs text-muted-foreground">
                  {t('explorer.trend_rank', { rank: String(index + 1) })}
                </span>
                <span className="grid grid-cols-[minmax(0,1fr)_auto] items-baseline gap-x-3">
                  <span className="min-w-0 truncate font-bold text-foreground">#{trend.tag}</span>
                  <span className="justify-self-end whitespace-nowrap text-xs text-muted-foreground">
                    {t(trend.count > 1 ? 'trends.posts_other' : 'trends.posts_one', {
                      count: formatTrendCount(trend.count),
                    })}
                  </span>
                </span>
              </span>
            </Link>
          ))}
        </div>
      )}
    </Section>
  )
}

function SuggestionsSection({
  users,
  currentUserId,
  commonFollowers,
  isFollowing,
  isPending,
  onToggleFollow,
}: {
  users: RelationUser[]
  currentUserId: string | null
  commonFollowers: Record<string, RelationUser[]>
  isFollowing: (id: string) => boolean
  isPending: (id: string) => boolean
  onToggleFollow: (user: RelationUser, next: boolean) => void
}) {
  const t = useT()
  return (
    <Section title={t('explorer.suggestions_title')}>
      {users.length === 0 ? (
        <p className="px-4 py-3 text-sm text-muted-foreground">{t('who.empty')}</p>
      ) : (
        <div className="grid gap-3 p-4 sm:grid-cols-2">
          {users.map((user) => (
            <SuggestionCard
              key={user.id}
              user={user}
              common={commonFollowers[user.id] ?? []}
              isSelf={user.id === currentUserId}
              isFollowing={isFollowing(user.id)}
              pending={isPending(user.id)}
              onToggleFollow={onToggleFollow}
            />
          ))}
        </div>
      )}
    </Section>
  )
}

function SuggestionCard({
  user,
  common,
  isSelf,
  isFollowing,
  pending,
  onToggleFollow,
}: {
  user: RelationUser
  common: RelationUser[]
  isSelf: boolean
  isFollowing: boolean
  pending: boolean
  onToggleFollow: (user: RelationUser, next: boolean) => void
}) {
  const t = useT()
  const commonLabel =
    common.length > 0
      ? t('explorer.followed_by', { names: common.map((u) => u.displayName).join(', ') })
      : t('explorer.suggested_profile')

  return (
    <div className="glass relative overflow-hidden rounded-[24px] border p-4 backdrop-blur-xl">
      <div className="mb-2 text-xs font-semibold text-muted-foreground">{commonLabel}</div>
      <div className="flex items-start gap-3">
        <ProfilLink author={{ id: user.id, username: user.username }} className="shrink-0">
          <Avatar className="h-14 w-14">
            {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
            <AvatarFallback>{initials(user)}</AvatarFallback>
          </Avatar>
        </ProfilLink>
        <div className="min-w-0 flex-1">
          <ProfilLink
            author={{ id: user.id, username: user.username }}
            className="block truncate font-bold text-foreground hover:underline"
          >
            {user.displayName}
          </ProfilLink>
          <ProfilLink
            author={{ id: user.id, username: user.username }}
            className="block truncate text-sm text-muted-foreground hover:underline"
          >
            @{user.username}
          </ProfilLink>
          {user.bio && <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{user.bio}</p>}
        </div>
        {!isSelf && (
          <Button
            size="sm"
            variant={isFollowing ? 'outline' : 'default'}
            disabled={pending}
            onClick={() => onToggleFollow(user, !isFollowing)}
            className={cn(
              'shrink-0 rounded-full font-bold',
              !isFollowing && 'bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white',
            )}
          >
            {isFollowing ? t('follow.followed') : t('follow.follow')}
          </Button>
        )}
      </div>
    </div>
  )
}

function PublicationsSection({
  posts,
  onDeleted,
  onUpdated,
}: {
  posts: FeedPost[]
  onDeleted: (id: string) => void
  onUpdated: (post: FeedPost) => void
}) {
  const t = useT()
  return (
    <Section title={t('explorer.publications_title')}>
      {posts.length === 0 ? (
        <p className="px-4 py-3 text-sm text-muted-foreground">{t('explorer.publications_empty')}</p>
      ) : (
        <div className="divide-y divide-border">
          {posts.map((post) => (
            <PostCard key={post.id} post={post} onDeleted={onDeleted} onUpdated={onUpdated} />
          ))}
        </div>
      )}
    </Section>
  )
}

function ExplorerUserRow({
  user,
  isSelf,
  isFollowing,
  pending,
  onToggleFollow,
}: {
  user: RelationUser
  isSelf: boolean
  isFollowing: boolean
  pending: boolean
  onToggleFollow: (user: RelationUser, next: boolean) => void
}) {
  const t = useT()
  return (
    <div className="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-accent">
      <ProfilLink author={{ id: user.id, username: user.username }} className="shrink-0">
        <Avatar className="h-11 w-11">
          {user.avatarUrl && <AvatarImage src={user.avatarUrl} alt={user.displayName} />}
          <AvatarFallback>{initials(user)}</AvatarFallback>
        </Avatar>
      </ProfilLink>
      <div className="min-w-0 flex-1">
        <ProfilLink
          author={{ id: user.id, username: user.username }}
          className="block truncate font-bold text-foreground hover:underline"
        >
          {user.displayName}
        </ProfilLink>
        <ProfilLink
          author={{ id: user.id, username: user.username }}
          className="block truncate text-sm text-muted-foreground hover:underline"
        >
          @{user.username}
        </ProfilLink>
      </div>
      {!isSelf && (
        <Button
          size="sm"
          variant={isFollowing ? 'outline' : 'default'}
          disabled={pending}
          onClick={() => onToggleFollow(user, !isFollowing)}
          className={cn(
            'shrink-0 rounded-full font-bold',
            !isFollowing && 'bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white',
          )}
        >
          {isFollowing ? t('follow.followed') : t('follow.follow')}
        </Button>
      )}
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="border-b border-border">
      <div className="flex items-center gap-2 px-4 py-3">
        <Sparkles className="h-4 w-4 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
        <h2 className="brand-text text-lg font-bold">{title}</h2>
      </div>
      {children}
    </section>
  )
}

function LoadingBlock() {
  return (
    <div className="flex justify-center py-16">
      <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
    </div>
  )
}

function EmptyState({
  icon,
  title,
  message,
}: {
  icon: React.ReactNode
  title: string
  message: string
}) {
  return (
    <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
      {icon}
      <h2 className="text-lg font-bold text-foreground">{title}</h2>
      <p className="max-w-sm text-sm text-muted-foreground">{message}</p>
    </div>
  )
}

function initials(user: RelationUser): string {
  return (user.displayName.charAt(0) || user.username.charAt(0) || '?').toUpperCase()
}

function postMatchesQuery(post: FeedPost, query: string): boolean {
  const needle = normalizeSearch(query.replace(/^#/, ''))
  if (!needle) return false
  const content = normalizeSearch(post.content)
  const hashtags = post.hashtags.map(normalizeSearch)
  const author = normalizeSearch(`${post.author.displayName} ${post.author.username}`)
  return (
    content.includes(needle) ||
    author.includes(needle) ||
    hashtags.some((tag) => tag.includes(needle))
  )
}

function normalizeSearch(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
}

function formatTrendCount(count: number): string {
  if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`
  if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`
  return String(count)
}
