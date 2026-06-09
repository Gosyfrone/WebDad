'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { ArrowLeft } from 'lucide-react'

import { ROUTES } from '@/lib/routes'
import { getPostById, type FeedPost } from '@/lib/posts'
import { useT } from '@/components/language-provider'
import { PostCard } from '@/components/feed/post-card'

/**
 * Page détail d'une publication (cible des liens de notification : « aller au
 * post direct »). Charge le post complet et réutilise `PostCard` (likes,
 * commentaires repliables…). En-tête avec bouton retour.
 */
export function PostDetail({ id }: { id: string }) {
  const t = useT()
  const router = useRouter()
  const [post, setPost] = useState<FeedPost | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    getPostById(id)
      .then((p) => {
        if (!cancelled) setPost(p)
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [id])

  return (
    <div className="flex flex-col">
      <header className="panel sticky top-0 z-10 flex items-center gap-4 border-b px-4 py-3 backdrop-blur-2xl">
        <button
          type="button"
          onClick={() => router.back()}
          aria-label={t('common.back')}
          className="rounded-full p-1 transition hover:bg-accent"
        >
          <ArrowLeft className="h-5 w-5" aria-hidden />
        </button>
        <h1 className="text-xl font-bold">{t('post.detail_title')}</h1>
      </header>

      {loading ? (
        <p className="px-4 py-10 text-center text-muted-foreground">{t('common.loading')}</p>
      ) : post ? (
        <PostCard post={post} onDeleted={() => router.push(ROUTES.feed)} />
      ) : (
        <p className="px-4 py-10 text-center text-muted-foreground">{t('post.not_found')}</p>
      )}
    </div>
  )
}
