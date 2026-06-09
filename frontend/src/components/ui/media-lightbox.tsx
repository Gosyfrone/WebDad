'use client'

import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import { Download, X } from 'lucide-react'

import { useT } from '@/components/language-provider'

interface MediaLightboxProps {
  open: boolean
  onClose: () => void
  /** URL de la ressource (objectURL déchiffré pour un message, ou URL gateway). */
  src: string
  type: 'image' | 'video'
  /** Nom de fichier proposé au téléchargement. */
  name?: string
  /** Affiche le bouton de téléchargement. */
  downloadable?: boolean
}

/**
 * Lightbox générique plein écran : affiche une image/vidéo en grand sur fond
 * sombre, avec fermeture (✕ ou Échap ou clic sur le fond) et, en option, un
 * bouton de téléchargement. Utilisé pour les pièces jointes de messagerie
 * (l'`src` est un objectURL déjà déchiffré → le téléchargement enregistre le
 * fichier en clair).
 */
export function MediaLightbox({ open, onClose, src, type, name, downloadable }: MediaLightboxProps) {
  const t = useT()

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    // Empêche le défilement de l'arrière-plan tant que la lightbox est ouverte.
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      window.removeEventListener('keydown', onKey)
      document.body.style.overflow = previous
    }
  }, [open, onClose])

  if (!open || typeof document === 'undefined') return null

  // Portail vers <body> : sans ça, le `position: fixed` serait confiné par un
  // ancêtre transformé/filtré (bulles de message en `backdrop-blur`) au lieu de
  // couvrir tout l'écran de l'app.
  return createPortal(
    <div
      className="fixed inset-0 z-[60] flex flex-col bg-black/90 backdrop-blur-sm"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
    >
      {/* Barre d'actions */}
      <div className="absolute right-3 top-3 z-10 flex items-center gap-2">
        {downloadable && (
          <a
            href={src}
            download={name || 'media'}
            onClick={(e) => e.stopPropagation()}
            aria-label={t('media.download')}
            className="rounded-full bg-white/10 p-2.5 text-white transition hover:bg-white/20"
          >
            <Download className="h-5 w-5" />
          </a>
        )}
        <button
          type="button"
          onClick={onClose}
          aria-label={t('common.close')}
          className="rounded-full bg-white/10 p-2.5 text-white transition hover:bg-white/20"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* Média : remplit l'espace dispo (grandit, ratio préservé). Le clic
          dessus ne ferme pas ; cliquer autour (fond) ferme. */}
      <div className="flex flex-1 items-center justify-center overflow-hidden p-4">
        {type === 'video' ? (
          <video
            src={src}
            controls
            autoPlay
            onClick={(e) => e.stopPropagation()}
            className="h-full w-full object-contain"
          />
        ) : (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={src}
            alt={name || ''}
            onClick={(e) => e.stopPropagation()}
            className="h-full w-full object-contain"
          />
        )}
      </div>
    </div>,
    document.body,
  )
}
