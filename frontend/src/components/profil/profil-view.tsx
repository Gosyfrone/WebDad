'use client'

import { useState } from 'react'
import { ArrowLeft, FileText } from 'lucide-react'
import Link from 'next/link'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'
import type { ProfilDetails, ProfilEditableFields } from '@/types'
import { PostCard, type PostCardProps } from '@/components/feed/post-card'
import { ProfilHeader } from '@/components/profil/profil-header'

type ProfilTab = 'posts' | 'replies' | 'likes'

interface ProfilViewProps {
  profil: ProfilDetails
  /** Posts de l'utilisateur (onglet « Posts »). */
  posts: PostCardProps[]
  /** Vrai si le profil affiché est celui de l'utilisateur courant. */
  isOwner?: boolean
}

/**
 * Corps de la page profil : en-tête sticky (retour + nb de posts), en-tête de
 * profil éditable, onglets, puis la liste de posts de l'onglet actif.
 *
 * L'état du profil est local : l'édition met l'en-tête à jour immédiatement
 * (optimiste) en attendant le câblage réseau. Les onglets « Réponses » et
 * « J'aime » sont des placeholders tant que l'API n'expose pas ces flux.
 */
export function ProfilView({ profil: initialProfil, posts, isOwner = true }: ProfilViewProps) {
  const [profil, setProfil] = useState(initialProfil)
  const [tab, setTab] = useState<ProfilTab>('posts')

  function handleEdit(fields: ProfilEditableFields) {
    setProfil((prev) => ({ ...prev, ...fields }))
  }

  return (
    <div className="flex flex-col">
      {/* En-tête sticky */}
      <div className="sticky top-0 z-10 flex items-center gap-6 border-b bg-background/80 px-4 py-2 backdrop-blur">
        <Link
          href={ROUTES.feed}
          aria-label="Retour au fil"
          className="rounded-full p-2 transition-colors hover:bg-accent"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div className="flex flex-col">
          <span className="font-bold leading-tight">{profil.displayName}</span>
          <span className="text-xs text-muted-foreground">
            {profil.postsCount} post{profil.postsCount > 1 ? 's' : ''}
          </span>
        </div>
      </div>

      <ProfilHeader profil={profil} isOwner={isOwner} onEdit={handleEdit} />

      {/* Onglets */}
      <div className="flex border-b">
        <TabButton active={tab === 'posts'} onClick={() => setTab('posts')}>
          Posts
        </TabButton>
        <TabButton active={tab === 'replies'} onClick={() => setTab('replies')}>
          Réponses
        </TabButton>
        <TabButton active={tab === 'likes'} onClick={() => setTab('likes')}>
          J&apos;aime
        </TabButton>
      </div>

      {/* Contenu de l'onglet */}
      {tab === 'posts' ? (
        posts.length > 0 ? (
          <div className="divide-y">
            {posts.map((post) => (
              <PostCard key={post.id} {...post} />
            ))}
          </div>
        ) : (
          <EmptyTab message="Vous n'avez pas encore publié de post." />
        )
      ) : (
        <EmptyTab
          message={
            tab === 'replies'
              ? 'Les réponses apparaîtront ici.'
              : "Les posts que vous aimez apparaîtront ici."
          }
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

function EmptyTab({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center gap-2 px-8 py-16 text-center">
      <FileText className="h-10 w-10 text-muted-foreground" aria-hidden />
      <p className="max-w-sm text-sm text-muted-foreground">{message}</p>
    </div>
  )
}
