'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

import { PostDetail } from '@/components/feed/post-detail'

/**
 * Détail d'une publication rendu **pile au-dessus de la colonne centrale (le
 * feed)**, via une intercepting route (`@modal/(.)posts/[id]`). Le feed reste
 * monté en arrière-plan (scroll, posts chargés et WebSocket « a posté »
 * préservés) ; on revient dessus instantanément par `router.back()` (bouton
 * retour de `PostDetail`) ou la touche Échap.
 *
 * Overlay `fixed` qui reproduit la grille du layout `(app)` (rangée centrée
 * `max-w-[1265px]` + espaceurs aux largeurs exactes des deux sidebars) pour que
 * le panneau opaque recouvre exactement le feed tout en laissant les sidebars
 * visibles et cliquables. `pointer-events-none` sur l'enveloppe, `auto` sur le
 * seul panneau. Sur accès direct/refresh d'une URL `/posts/[id]`, l'interception
 * ne s'applique pas → la vraie page plein écran prend le relais.
 */
export function PostDetailModal({ id }: { id: string }) {
  const router = useRouter()

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') router.back()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [router])

  return (
    <div className="pointer-events-none fixed inset-0 z-40 flex justify-center">
      <div className="flex w-full max-w-[1265px]">
        {/* Espaceur = SidebarLeft (w-[275px], visible ≥ lg) */}
        <div className="hidden w-[275px] shrink-0 lg:block" />

        {/* Panneau central, opaque, aligné sur la colonne du feed */}
        <div className="bg-page pointer-events-auto flex min-w-0 flex-1 flex-col overflow-y-auto lg:border-x">
          <PostDetail id={id} />
        </div>

        {/* Espaceur = SidebarRight (w-[min(350px,30vw)] min-w-[290px], visible ≥ xl) */}
        <div className="hidden w-[min(350px,30vw)] min-w-[290px] shrink-0 xl:block" />
      </div>
    </div>
  )
}
