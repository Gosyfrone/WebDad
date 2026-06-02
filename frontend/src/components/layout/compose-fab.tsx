import { Plus } from 'lucide-react'

import { CreatePostDialog } from '@/components/feed/create-post-dialog'

/**
 * Bouton d'action flottant (masqué ≥ lg) ouvrant la popup de publication.
 * Positionné en bas à droite (convention X.com), au-dessus de la barre d'onglets.
 */
export function ComposeFab() {
  return (
    <CreatePostDialog>
      <button
        aria-label="Créer un post"
        className="fixed bottom-20 right-4 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-transform hover:scale-105 active:scale-95 lg:hidden"
      >
        <Plus className="h-6 w-6" />
      </button>
    </CreatePostDialog>
  )
}
