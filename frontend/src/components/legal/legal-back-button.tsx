'use client'

import { useRouter } from 'next/navigation'
import { ArrowLeft } from 'lucide-react'

import { ROUTES } from '@/lib/routes'

/**
 * Bouton « Retour » des pages légales : revient à la page précédente
 * (`router.back()`) afin que l'utilisateur connecté retrouve l'onglet d'où il
 * vient (feed, profil, paramètres…) plutôt que d'atterrir sur l'accueil/login.
 *
 * Repli : si la page légale a été ouverte directement (aucun historique de
 * navigation interne — `history.length <= 1`), on redirige vers l'accueil.
 *
 * Client Component (a besoin de `useRouter`) embarqué dans `LegalShell`
 * (Server Component).
 */
export function LegalBackButton() {
  const router = useRouter()

  function handleClick() {
    if (typeof window !== 'undefined' && window.history.length > 1) {
      router.back()
    } else {
      router.push(ROUTES.home)
    }
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
    >
      <ArrowLeft className="h-4 w-4" aria-hidden />
      Retour à Breezy
    </button>
  )
}
