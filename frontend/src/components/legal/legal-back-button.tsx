'use client'

import { useRouter } from 'next/navigation'
import { ArrowLeft } from 'lucide-react'

import { ROUTES } from '@/lib/routes'
import { getLegalOrigin } from '@/lib/legal-origin'
import { useT } from '@/components/language-provider'

/**
 * Bouton « Retour » des pages légales : revient à la **dernière page Breezy
 * (non légale)** visitée (feed, messages, profil…), mémorisée par le
 * `RouteOriginTracker`. Naviguer ENTRE pages légales ne change pas cette origine
 * → on ne fait pas un simple retour arrière navigateur (qui ramènerait à la page
 * légale précédente).
 *
 * Repli : aucune origine connue (page légale ouverte directement / nouvel
 * onglet) → accueil (qui route vers feed ou login selon la session).
 *
 * Client Component (a besoin de `useRouter`), libellé localisé (`useT`).
 */
export function LegalBackButton() {
  const router = useRouter()
  const t = useT()

  function handleClick() {
    router.push(getLegalOrigin() ?? ROUTES.home)
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
    >
      <ArrowLeft className="h-4 w-4" aria-hidden />
      {t('legal.back')}
    </button>
  )
}
