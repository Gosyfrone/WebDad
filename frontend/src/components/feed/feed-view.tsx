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
      <div className="z-10 border-b bg-background/80 backdrop-blur lg:sticky lg:top-0">
        <h1 className="hidden px-4 py-3 text-xl font-bold lg:block">Fil d&apos;actualité</h1>
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

      {/* Zone de création de post inline (desktop) ; sur mobile, c'est le bouton « + » flottant */}
      <div className="hidden lg:block">
        <CreatePost />
      </div>

      {/* Contenu selon l'onglet actif */}
      {tab === 'for-you' ? (
        <div className="divide-y">
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
        'flex-1 py-3 text-sm transition-colors hover:bg-muted/30',
        active
          ? 'border-b-2 border-primary font-bold'
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
    <div className="flex flex-col items-center gap-2 px-8 py-16 text-center">
      <Users className="h-10 w-10 text-muted-foreground" aria-hidden />
      <h2 className="text-lg font-bold">Aucun post pour le moment</h2>
      <p className="max-w-sm text-sm text-muted-foreground">
        Les posts des comptes que vous suivez apparaîtront ici. Abonnez-vous à
        des profils pour personnaliser ce fil.
      </p>
    </div>
  )
}
