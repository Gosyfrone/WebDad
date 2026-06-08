'use client'

import { useState } from 'react'

import { PostComposer } from '@/components/feed/post-composer'
import { useT } from '@/components/language-provider'
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
  const t = useT()
  const [open, setOpen] = useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="panel top-24 translate-y-0 border p-4 shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-w-xl">
        <DialogHeader className="sr-only">
          <DialogTitle>{t('post.create_aria')}</DialogTitle>
          <DialogDescription>{t('composer.dialog_desc')}</DialogDescription>
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
