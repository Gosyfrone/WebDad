import { FeedOverlay } from '@/components/feed/feed-overlay'
import { OverlayWhenPath } from '@/components/feed/overlay-when-path'
import { PostDetail } from '@/components/feed/post-detail'

/**
 * Intercepting route : capte les navigations soft vers `/posts/[id]` depuis
 * l'espace `(app)` et affiche le détail en panneau au-dessus du feed (gardé
 * monté). Accès direct/refresh → vraie page `(app)/posts/[id]/page.tsx`.
 */
export default async function InterceptedPostPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  return (
    <OverlayWhenPath path="/posts/" exact={false}>
      <FeedOverlay>
        <PostDetail id={id} />
      </FeedOverlay>
    </OverlayWhenPath>
  )
}
