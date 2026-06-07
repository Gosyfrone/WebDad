'use client'

import { useCallback, useEffect, useState } from 'react'
import { Loader2, Users } from 'lucide-react'

import { cn } from '@/lib/utils'
import {
  listFeed,
  listFollowingFeed,
  subscribePostCreated,
  type FeedPost,
} from '@/lib/posts'
import { CreatePost } from '@/components/feed/create-post'
import { PostCard } from '@/components/feed/post-card'

type FeedTab = 'for-you' | 'following'

/**
 * Corps du fil d'actualité : en-tête sticky, onglets « Pour toi » /
 * « Abonnements », zone de composition, puis la liste de l'onglet actif.
 *
 * Les données sont chargées via le post-service (`listFeed` / `listFollowingFeed`).
 * Un post fraîchement publié est prépendu sans refetch (event `post-created`).
 */
export function FeedView() {
  const [tab, setTab] = useState<FeedTab>('for-you')
  const [posts, setPosts] = useState<FeedPost[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')

    const load = tab === 'for-you' ? listFeed() : listFollowingFeed()
    load
      .then((list) => {
        if (!cancelled) setPosts(list)
      })
      .catch(() => {
        if (!cancelled) setError('Impossible de charger le fil.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [tab])

  // Un nouveau post (composer inline ou popup sidebar) est prépendu au fil.
  useEffect(() => subscribePostCreated((post) => setPosts((prev) => [post, ...prev])), [])

  const handleDeleted = useCallback((id: string) => {
    setPosts((prev) => prev.filter((p) => p.id !== id))
  }, [])

  return (
    <div className="flex flex-col">
      {/* En-tête : sticky sur desktop ; sur mobile l'en-tête global (logo) prend le relais */}
      <div className="panel z-10 border-b lg:sticky lg:top-0">
        <h1 className="brand-text hidden px-4 py-3 text-xl font-bold lg:block">
          Fil d&apos;actualité
        </h1>
        {/* Onglets Pour toi / Abonnements */}
        <div className="flex">
          <TabButton active={tab === 'for-you'} onClick={() => setTab('for-you')}>
            Pour toi
          </TabButton>
          <TabButton active={tab === 'following'} onClick={() => setTab('following')}>
            Abonnements
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
        <EmptyState
          title="Fil indisponible"
          message={error}
        />
      ) : posts.length === 0 ? (
        tab === 'for-you' ? (
          <EmptyState
            title="Aucun post pour le moment"
            message="Soyez le premier à publier quelque chose sur Breezy."
          />
        ) : (
          <EmptyState
            title="Aucun post pour le moment"
            message="Les posts des comptes que vous suivez apparaîtront ici. Abonnez-vous à des profils pour personnaliser ce fil."
          />
        )
      ) : (
        <div className="divide-y divide-border">
          {posts.map((post) => (
            <PostCard key={post.id} post={post} onDeleted={handleDeleted} />
          ))}
        </div>
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
