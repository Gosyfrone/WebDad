import { FloatingThemeToggle } from '@/components/theme-toggle'

/**
 * Layout des pages légales publiques (mentions légales / CGU / confidentialité).
 *
 * Comme l'espace `(auth)`, ces pages n'ont pas de menu : on y monte donc le
 * `FloatingThemeToggle` (interrupteur clair/sombre flottant, bas gauche,
 * translucide) une seule fois pour les trois pages. Le fond et les surfaces sont
 * gérés par `LegalShell` (tokenisés → clair + sombre).
 */
export default function LegalLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-dvh bg-background">
      {children}
      <FloatingThemeToggle />
    </div>
  )
}
