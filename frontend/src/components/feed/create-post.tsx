import { PostComposer } from '@/components/feed/post-composer'

/**
 * Zone de composition inline affichée en tête du fil d'actualité.
 * Réutilise {@link PostComposer} (logique partagée avec la popup de la sidebar).
 */
export function CreatePost() {
  return (
    <PostComposer className="panel border-b px-4 py-3" />
  )
}
