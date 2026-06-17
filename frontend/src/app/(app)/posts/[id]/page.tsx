import { FeedOverlay } from '@/components/feed/feed-overlay'
import { PostDetail } from '@/components/feed/post-detail'

// Page détail d'une publication (/posts/<id>) — cible des notifications.
// Rendue en overlay au-dessus du feed persistant du layout.
export default async function PostPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  return (
    <FeedOverlay headerOffset={false}>
      <PostDetail id={id} />
    </FeedOverlay>
  )
}
