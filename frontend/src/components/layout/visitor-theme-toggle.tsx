'use client'

import { useAuthGate } from '@/components/auth-prompt-provider'
import { ThemeSwitch } from '@/components/theme-toggle'

/**
 * Interrupteur clair/sombre flottant (bas gauche, comme login/register) pour les
 * pages visiteur de l'espace `(app)` (`/feed`, `/posts/:id`).
 *
 * Réservé au visiteur (les membres ont le sélecteur d'apparence dans leur menu)
 * et **desktop uniquement** (`hidden lg:inline-flex`) : sur mobile, la barre
 * d'onglets occupe le bas de l'écran — l'interrupteur est alors porté par
 * l'en-tête mobile (à la place de la cloche).
 */
export function VisitorThemeToggle() {
  const { isVisitor } = useAuthGate()
  if (!isVisitor) return null
  return (
    <ThemeSwitch className="fixed bottom-4 left-4 z-50 hidden animate-in fade-in lg:inline-flex" />
  )
}
