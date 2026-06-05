'use client'

import { useEffect, useState } from 'react'
import { Plus } from 'lucide-react'
import { usePathname } from 'next/navigation'

import { CreatePostDialog } from '@/components/feed/create-post-dialog'
import { ROUTES } from '@/lib/routes'

/**
 * Bouton d'action flottant (masqué ≥ lg) ouvrant la popup de publication.
 * Positionné en bas à droite (convention X.com), au-dessus de la barre d'onglets.
 */
export function ComposeFab() {
  const pathname = usePathname()
  const [show, setShow] = useState(false)

  useEffect(() => {
    if (pathname !== ROUTES.feed) {
      setShow(false)
      return
    }

    let timeoutId: ReturnType<typeof setTimeout> | undefined
    let observer: IntersectionObserver | undefined

    function observeComposer() {
      const composer = document.getElementById('feed-composer')

      if (!composer) {
        timeoutId = setTimeout(observeComposer, 100)
        return
      }

      observer = new IntersectionObserver(
        ([entry]) => {
          setShow(!entry.isIntersecting)
        },
        {
          root: null,
          threshold: 0,
          rootMargin: '-56px 0px 0px 0px',
        }
      )

      observer.observe(composer)
    }

    observeComposer()

    return () => {
      if (timeoutId) clearTimeout(timeoutId)
      observer?.disconnect()
    }
  }, [pathname])

  if (pathname !== ROUTES.feed || !show) return null

  return (
    <CreatePostDialog>
      <button
        aria-label="Créer un post"
        className="fixed bottom-[4.25rem] right-3 z-40 flex h-11 w-11 items-center justify-center rounded-full border border-[#5B6CFF]/30 bg-white/35 text-[#5B6CFF] shadow-[0_12px_28px_rgba(91,108,255,0.22)] backdrop-blur-xl transition hover:bg-white/55 hover:text-[#8D3DFF] active:scale-95 dark:bg-white/10 dark:text-[#9aa6ff] dark:hover:bg-white/20 lg:hidden"
      >
        <Plus className="h-6 w-6 stroke-[2.7]" />
      </button>
    </CreatePostDialog>
  )
}
