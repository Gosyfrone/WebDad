import { FloatingThemeToggle } from '@/components/theme-toggle'

/**
 * Layout des pages publiques d'authentification (login / register).
 * Centre le contenu (formulaire) verticalement et horizontalement.
 *
 * Interrupteur clair/sombre flottant (bas gauche, translucide) commun aux deux
 * pages : elles n'ont pas de menu pour basculer le thème autrement.
 */
export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col items-stretch overflow-hidden bg-background">
      <div className="h-full w-full">{children}</div>
      <FloatingThemeToggle />
    </div>
  )
}
