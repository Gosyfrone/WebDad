import { PostDetailModal } from '@/components/feed/post-detail-modal'

/**
 * Intercepting route : capte les navigations soft vers `/posts/[id]` depuis
 * l'espace `(app)` (feed, explorer, profil, notifications) et affiche le détail
 * en panneau plein écran, le feed restant monté derrière. L'accès direct/refresh
 * d'une URL `/posts/[id]` tombe sur la vraie page `(app)/posts/[id]/page.tsx`.
 */
export default async function InterceptedPostPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  return <PostDetailModal id={id} />
}
