'use client'

import { useEffect, useState } from 'react'
import { Monitor, Moon, Sun } from 'lucide-react'
import { useTheme } from 'next-themes'

import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'

/**
 * Sélecteur d'apparence :
 *   - un interrupteur façon iOS clair/sombre (curseur sur le Soleil ou la Lune) ;
 *   - une ligne « Mode système » (icône écran) qui suit la préférence de l'OS.
 *
 * Quand le mode système est activé, l'interrupteur clair/sombre est grisé
 * (désactivé) et reflète l'apparence effective déduite de l'OS (`resolvedTheme`).
 * S'appuie sur next-themes ; persistance auto (localStorage). Le flag `mounted`
 * évite le mismatch d'hydratation (le serveur ignore le thème).
 */
export function ThemeToggle() {
  const t = useT()
  const { theme, resolvedTheme, setTheme } = useTheme()
  const [mounted, setMounted] = useState(false)
  useEffect(() => setMounted(true), [])

  // Sens du dernier basculement, pour jouer la bonne animation de slide.
  // `null` = aucun (montage initial → pas d'animation parasite).
  const [slide, setSlide] = useState<'left' | 'right' | null>(null)

  const systemOn = mounted && theme === 'system'
  const isDark = mounted && (systemOn ? resolvedTheme === 'dark' : theme === 'dark')

  function toggleLightDark() {
    if (systemOn) return
    const goingDark = !isDark
    setSlide(goingDark ? 'right' : 'left')
    setTheme(goingDark ? 'dark' : 'light')
  }

  function toggleSystem() {
    // Désactiver le système : on fige l'apparence courante en choix manuel.
    if (systemOn) setTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
    else setTheme('system')
  }

  return (
    <div className="px-2 py-1">
      <span className="text-xs font-medium text-muted-foreground">{t('theme.title')}</span>

      <div className="mt-2 flex flex-col gap-1">
        {/* Interrupteur clair / sombre */}
        <div className="flex items-center justify-between rounded-lg px-1 py-2">
          <span className="text-sm">{t('theme.appearance')}</span>
          <button
            type="button"
            role="switch"
            aria-checked={isDark}
            aria-label={t('theme.toggle_aria')}
            disabled={systemOn}
            onClick={toggleLightDark}
            className={cn(
              'relative inline-flex h-8 w-[3.75rem] shrink-0 items-center rounded-full bg-muted transition-opacity',
              systemOn && 'cursor-not-allowed opacity-50',
            )}
          >
            {/* Curseur qui glisse (animation directionnelle au clic) */}
            <span
              className={cn(
                'absolute left-1 z-0 h-6 w-6 rounded-full bg-background shadow',
                isDark ? 'translate-x-7' : 'translate-x-0',
                slide === 'right' && 'animate-theme-thumb-right',
                slide === 'left' && 'animate-theme-thumb-left',
              )}
            />
            {/* Icônes aux deux extrémités */}
            <Sun
              className={cn(
                'absolute left-2 z-10 h-4 w-4 transition-colors',
                !isDark ? 'text-amber-500' : 'text-muted-foreground/50',
              )}
              aria-hidden
            />
            <Moon
              className={cn(
                'absolute right-2 z-10 h-4 w-4 transition-colors',
                isDark ? 'text-foreground' : 'text-muted-foreground/50',
              )}
              aria-hidden
            />
          </button>
        </div>

        {/* Ligne Mode système */}
        <button
          type="button"
          aria-pressed={systemOn}
          onClick={toggleSystem}
          className="flex items-center justify-between rounded-lg px-1 py-2 transition-colors hover:bg-accent"
        >
          <span className="flex items-center gap-2 text-sm">
            <Monitor className="h-5 w-5" aria-hidden />
            {t('theme.system')}
          </span>
          <span
            className={cn(
              'text-xs font-medium',
              systemOn ? 'text-primary' : 'text-muted-foreground',
            )}
          >
            {systemOn ? t('theme.on') : t('theme.off')}
          </span>
        </button>
      </div>
    </div>
  )
}

/**
 * Interrupteur clair/sombre COMPACT et FLOTTANT, pour les pages publiques
 * (login / register) qui n'ont pas de menu. Posé en bas à gauche, **translucide**
 * (`backdrop-blur` + fond très léger) pour laisser le dégradé de fond visible.
 *
 * Contrairement à {@link ThemeToggle}, pas d'option « système » : un clic bascule
 * simplement clair ↔ sombre (si le thème courant est « système », on part de
 * l'apparence effective `resolvedTheme` et on fige un choix manuel).
 */
export function FloatingThemeToggle() {
  const t = useT()
  const { resolvedTheme, setTheme } = useTheme()
  const [mounted, setMounted] = useState(false)
  const [slide, setSlide] = useState<'left' | 'right' | null>(null)
  useEffect(() => setMounted(true), [])

  // Tant que le thème n'est pas connu côté client, on ne rend RIEN : évite
  // d'afficher un état erroné (curseur côté Soleil alors qu'on est en sombre)
  // au chargement complet de la page — cas typique de l'arrivée sur /login
  // après déconnexion (window.location.assign) — et tout mismatch d'hydratation.
  if (!mounted) return null

  // Source de vérité : le thème EFFECTIF (`resolvedTheme`, qui résout aussi
  // « système »). Repli sur la classe `.dark` réellement posée sur <html> par
  // next-themes (avant le paint) si `resolvedTheme` n'est pas encore défini.
  const isDark = resolvedTheme
    ? resolvedTheme === 'dark'
    : document.documentElement.classList.contains('dark')

  function toggle() {
    const goingDark = !isDark
    setSlide(goingDark ? 'right' : 'left')
    setTheme(goingDark ? 'dark' : 'light')
  }

  return (
    <button
      type="button"
      role="switch"
      aria-checked={isDark}
      aria-label={t('theme.toggle_aria')}
      onClick={toggle}
      className="fixed bottom-4 left-4 z-50 inline-flex h-9 w-16 animate-in items-center rounded-full border border-white/40 bg-white/20 shadow-lg backdrop-blur-md transition-colors fade-in hover:bg-white/30 dark:border-white/15 dark:bg-white/10 dark:hover:bg-white/20"
    >
      {/* Curseur qui glisse (animation directionnelle au clic) */}
      <span
        className={cn(
          'absolute left-1 z-0 h-7 w-7 rounded-full bg-background shadow',
          isDark ? 'translate-x-7' : 'translate-x-0',
          slide === 'right' && 'animate-theme-thumb-right',
          slide === 'left' && 'animate-theme-thumb-left',
        )}
      />
      <Sun
        className={cn(
          'absolute left-2 z-10 h-4 w-4 transition-colors',
          !isDark ? 'text-amber-500' : 'text-white/50',
        )}
        aria-hidden
      />
      <Moon
        className={cn(
          'absolute right-2 z-10 h-4 w-4 transition-colors',
          isDark ? 'text-white' : 'text-slate-500/60',
        )}
        aria-hidden
      />
    </button>
  )
}
