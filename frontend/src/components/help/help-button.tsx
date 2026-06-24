'use client'

import Link from 'next/link'
import { HelpCircle, Lightbulb, Sparkles } from 'lucide-react'

import { cn } from '@/lib/utils'
import { ROUTES } from '@/lib/routes'
import { useT } from '@/components/language-provider'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useTutorial } from '@/components/tutorial/tutorial-provider'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

/**
 * Bouton « lampe éclairée » ouvrant un menu à deux entrées : la page d'aide et
 * la relance du didacticiel guidé.
 *
 * Deux variantes de placement :
 *   - `desktop` : bouton flottant fixe en bas à droite (≥ lg uniquement) ;
 *   - `mobile`  : bouton inline, posé à droite de la cloche dans l'en-tête mobile.
 */
export function HelpButton({ variant }: { variant: 'desktop' | 'mobile' }) {
  const t = useT()
  const { start } = useTutorial()
  const { isVisitor } = useAuthGate()

  const isDesktop = variant === 'desktop'

  // La lampe flottante (desktop) ne s'affiche que pour un compte connecté ; en
  // mobile elle n'est rendue que dans le créneau « cloche » (déjà non-visiteur).
  if (isDesktop && isVisitor) return null

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          data-tour="help-button"
          aria-label={t('help.menu_aria')}
          className={cn(
            'flex items-center justify-center rounded-full text-foreground transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            isDesktop
              ? 'fixed bottom-6 right-6 z-40 hidden h-12 w-12 border bg-background shadow-[0_14px_34px_rgba(91,108,255,0.22)] hover:scale-105 lg:flex'
              : 'h-9 w-9',
          )}
        >
          <Lightbulb className={cn(isDesktop ? 'h-6 w-6' : 'h-6 w-6', 'text-[#5B6CFF] dark:text-[#9aa6ff]')} aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        side={isDesktop ? 'top' : 'bottom'}
        className="panel z-40 w-52 border shadow-[0_18px_44px_rgba(91,108,255,0.18)]"
      >
        <DropdownMenuItem asChild>
          <Link href={ROUTES.aide}>
            <HelpCircle className="mr-2 h-4 w-4" />
            {t('help.page_link')}
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => start()}>
          <Sparkles className="mr-2 h-4 w-4" />
          {t('help.replay_tutorial')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
