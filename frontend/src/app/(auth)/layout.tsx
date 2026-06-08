import { FloatingThemeToggle } from '@/components/theme-toggle'
import { FloatingLanguageSelector } from '@/components/language-selector'

/**
 * Layout des pages publiques d'authentification (login / register).
 * Centre le contenu (formulaire) verticalement et horizontalement.
 *
 * Sélecteur de langue + interrupteur clair/sombre flottants (bas gauche,
 * translucides) communs aux deux pages : elles n'ont pas de menu pour basculer
 * la langue / le thème autrement. La langue est posée juste au-dessus du thème.
 */
export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col items-stretch overflow-hidden bg-background">
      <div className="h-full w-full">{children}</div>
      <FloatingLanguageSelector />
      <FloatingThemeToggle />
    </div>
  )
}
