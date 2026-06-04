'use client'

import { useState } from 'react'
import { Users } from 'lucide-react'

import { cn } from '@/lib/utils'
import { CreatePost } from '@/components/feed/create-post'
import { PostCard, type PostCardProps } from '@/components/feed/post-card'

type FeedTab = 'for-you' | 'following'

interface FeedViewProps {
  posts: PostCardProps[]
}

/**
 * Corps du fil d'actualité : en-tête sticky, onglets « Pour toi » / « Abonnements »,
 * zone de composition, puis la liste correspondant à l'onglet actif.
 *
 * L'onglet « Abonnements » est un placeholder : il sera alimenté par
 * l'API (posts des comptes suivis) dans l'issue post/profil.
 */
export function FeedView({ posts }: FeedViewProps) {
  const [tab, setTab] = useState<FeedTab>('for-you')

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

      {/* Contenu selon l'onglet actif */}
      {tab === 'for-you' ? (
        <div className="divide-y divide-border">
          {posts.map((post) => (
            <PostCard key={post.id} {...post} />
          ))}
        </div>
      ) : (
        <FollowingPlaceholder />
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

/** État vide de l'onglet « Abonnements » (en attendant l'API). */
function FollowingPlaceholder() {
  return (
    <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
      <Users className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
      <h2 className="text-lg font-bold">Aucun post pour le moment</h2>
      <p className="max-w-sm text-sm text-muted-foreground">
        Les posts des comptes que vous suivez apparaîtront ici. Abonnez-vous à
        des profils pour personnaliser ce fil.
      </p>
    </div>
  )
}
