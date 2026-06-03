import { PostComposer } from '@/components/feed/post-composer'

/**
 * Zone de composition inline affichée en tête du fil d'actualité.
 * Réutilise {@link PostComposer} (logique partagée avec la popup de la sidebar).
 */
export function CreatePost() {
  return (
    <PostComposer className="border-b border-white/35 bg-gradient-to-r from-[#8D3DFF]/10 via-white/20 to-[#47D9FF]/10 px-4 py-3 backdrop-blur-xl" />
  )
}
