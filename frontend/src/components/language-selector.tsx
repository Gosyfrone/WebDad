'use client'

import * as React from 'react'
import { Check, Globe } from 'lucide-react'

import { cn } from '@/lib/utils'
import { LOCALES, type Locale } from '@/lib/i18n'
import { useLanguage } from '@/components/language-provider'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

/** Définition de la locale courante (pour le libellé/flag du déclencheur). */
function useCurrentLocaleDef() {
  const { locale } = useLanguage()
  return LOCALES.find((l) => l.id === locale) ?? LOCALES[0]
}

/** Liste des langues, partagée par les deux variantes du sélecteur. */
function LocaleItems({
  current,
  onSelect,
}: {
  current: Locale
  onSelect: (id: Locale) => void
}) {
  return (
    <>
      {LOCALES.map((l) => {
        const active = l.id === current
        return (
          <DropdownMenuItem
            key={l.id}
            onSelect={() => onSelect(l.id)}
            className="flex items-center gap-2"
          >
            <span aria-hidden className="text-base leading-none">
              {l.flag}
            </span>
            <span className="flex-1">{l.label}</span>
            {active && <Check className="h-4 w-4 text-primary" aria-hidden />}
          </DropdownMenuItem>
        )
      })}
    </>
  )
}

/**
 * Sélecteur de langue (dropdown globe 🌐) — variante de menu, montée au-dessus
 * du sélecteur de thème dans la sidebar (PC) et le tiroir (mobile).
 *
 * Pilote LanguageProvider (cf. language-provider.tsx) : scalable à N langues, la
 * liste vient du registre LOCALES (lib/i18n.ts). Le flag `mounted` évite le
 * mismatch d'hydratation (le serveur rend toujours la langue par défaut).
 */
export function LanguageSelector() {
  const { locale, setLocale, t } = useLanguage()
  const current = useCurrentLocaleDef()
  const [mounted, setMounted] = React.useState(false)
  React.useEffect(() => setMounted(true), [])

  return (
    <div className="px-2 py-1">
      <span className="text-xs font-medium text-muted-foreground">
        {t('lang.title')}
      </span>
      <div className="mt-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              aria-label={t('lang.select_aria')}
              className="flex w-full items-center gap-2 rounded-lg px-1 py-2 text-sm transition-colors hover:bg-accent"
            >
              <Globe className="h-5 w-5 shrink-0" aria-hidden />
              <span className="flex-1 text-left">
                {mounted ? current.label : LOCALES[0].label}
              </span>
              <span aria-hidden className="text-base leading-none">
                {mounted ? current.flag : LOCALES[0].flag}
              </span>
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="panel w-48 border">
            <LocaleItems current={locale} onSelect={setLocale} />
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}

/**
 * Sélecteur de langue COMPACT et FLOTTANT, pour les pages publiques
 * (login / register) qui n'ont pas de menu. Posé en bas à gauche, **juste
 * au-dessus** du FloatingThemeToggle et **translucide** (`backdrop-blur` + fond
 * léger) pour laisser voir le dégradé de fond.
 *
 * Déclencheur réduit au seul **bouton globe** (rond) ; le menu déroulant liste
 * Français / English avec une coche sur la langue active (cf. {@link LocaleItems}).
 */
export function FloatingLanguageSelector() {
  const { locale, setLocale, t } = useLanguage()
  const [mounted, setMounted] = React.useState(false)
  React.useEffect(() => setMounted(true), [])

  if (!mounted) return null

  return (
    <div className="fixed bottom-16 left-4 z-50">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label={t('lang.select_aria')}
            className={cn(
              'inline-flex h-9 w-9 items-center justify-center rounded-full border border-white/40 bg-white/20 shadow-lg backdrop-blur-md transition-colors',
              'animate-in fade-in hover:bg-white/30 dark:border-white/15 dark:bg-white/10 dark:hover:bg-white/20',
            )}
          >
            <Globe className="h-4 w-4" aria-hidden />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" side="top" className="panel w-48 border">
          <LocaleItems current={locale} onSelect={setLocale} />
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
