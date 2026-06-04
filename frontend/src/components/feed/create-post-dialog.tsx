'use client'

import { useState } from 'react'

import { PostComposer } from '@/components/feed/post-composer'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

interface CreatePostDialogProps {
  /** Élément déclencheur (ex. bouton « Breezer »), rendu via `asChild`. */
  children: React.ReactNode
}

/**
 * Popup de publication déclenchée depuis la sidebar.
 *
 * Réutilise {@link PostComposer} et ferme la fenêtre après envoi. Le titre et
 * la description sont masqués visuellement mais présents pour l'accessibilité.
 */
export function CreatePostDialog({ children }: CreatePostDialogProps) {
  const [open, setOpen] = useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="top-24 translate-y-0 border-[#D9C6FF]/70 bg-gradient-to-br from-[#F8F3FF] via-[#EADCFF] to-[#EEF9FF] p-4 shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-w-xl">
        <DialogHeader className="sr-only">
          <DialogTitle>Créer un post</DialogTitle>
          <DialogDescription>
            Rédigez et publiez un nouveau post (280 caractères maximum).
          </DialogDescription>
        </DialogHeader>

        <PostComposer
          autoFocus
          className="pt-6"
          onPosted={() => setOpen(false)}
        />
      </DialogContent>
    </Dialog>
  )
}
