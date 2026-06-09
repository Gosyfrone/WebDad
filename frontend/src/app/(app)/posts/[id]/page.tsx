import { PostDetail } from '@/components/feed/post-detail'

// Page détail d'une publication (/posts/<id>) — cible des notifications.
export default async function PostPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  return <PostDetail id={id} />
}
