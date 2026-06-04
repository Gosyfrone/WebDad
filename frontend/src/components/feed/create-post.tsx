import { PostComposer } from '@/components/feed/post-composer'

/**
 * Zone de composition inline affichée en tête du fil d'actualité.
 * Réutilise {@link PostComposer} (logique partagée avec la popup de la sidebar).
 */
export function CreatePost() {
  return (
    <PostComposer className="border-b border-[#D9C6FF]/70 bg-gradient-to-r from-[#F8F3FF] via-[#EADCFF] to-[#EEF9FF] px-4 py-3" />
  )
}
