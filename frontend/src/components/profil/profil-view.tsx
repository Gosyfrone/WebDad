'use client'

import { useEffect, useState } from 'react'
import { ArrowLeft, FileText, Loader2 } from 'lucide-react'
import Link from 'next/link'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'
import { getMyProfil, getPublicProfil, saveMyProfil } from '@/lib/profil-client'
import { listByAuthor, type FeedPost } from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import type { ProfilDetails, ProfilEditableFields } from '@/types'
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
    return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  })
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
  const [profil, setProfil] = useState<ProfilDetails | null>(null)
  const [posts, setPosts] = useState<FeedPost[]>([])
  const [tab, setTab] = useState<ProfilTab>('posts')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const isOwner = !username

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
          setError(err instanceof Error ? err.message : 'Profil introuvable.')
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadProfil()
    return () => {
      cancelled = true
    }
  }, [username])

  // Posts de l'auteur (onglet « Posts »), chargés une fois le profil connu.
  useEffect(() => {
    if (!profil?.userId) return
    let cancelled = false
    listByAuthor(profil.userId)
      .then((list) => {
        if (!cancelled) setPosts(list)
      })
      .catch(() => {
        if (!cancelled) setPosts([])
      })
    return () => {
      cancelled = true
    }
  }, [profil?.userId])

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
      toast({ title: 'Profil mis à jour' })
    } catch (err) {
      toast({
        title: 'Mise à jour impossible',
        description:
          err instanceof Error ? err.message : 'Le profil n’a pas pu être enregistré.',
        variant: 'destructive',
      })
      throw err
    } finally {
      setSaving(false)
    }
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
        <h1 className="text-lg font-bold text-slate-950">Profil indisponible</h1>
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
          aria-label="Retour au fil"
          className="rounded-full p-2 transition-colors hover:bg-accent hover:text-[#5B6CFF] dark:hover:text-[#9aa6ff]"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div className="flex flex-col">
          <span className="font-bold leading-tight text-foreground">{profil.displayName}</span>
          <span className="text-xs text-muted-foreground">
            {posts.length} post{posts.length > 1 ? 's' : ''}
          </span>
        </div>
      </div>

      <ProfilHeader profil={profil} isOwner={isOwner} saving={saving} onEdit={handleEdit} />

      {/* Onglets */}
      <div className="panel flex border-b">
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
          <div className="divide-y divide-border">
            {posts.map((post) => (
              <PostCard key={post.id} post={post} onDeleted={handleDeleted} onUpdated={handleUpdated} />
            ))}
          </div>
        ) : (
          <EmptyTab message="Aucun post publié pour le moment." />
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
