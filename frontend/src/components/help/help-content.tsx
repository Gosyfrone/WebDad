'use client'

import {
  Bell,
  Bookmark,
  HelpCircle,
  Home,
  Mail,
  Search,
  Settings,
  Sparkles,
  User,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useT } from '@/components/language-provider'
import { useTutorial } from '@/components/tutorial/tutorial-provider'

/** Sections décrites par la page d'aide (clés i18n dérivées de `id`). */
const SECTIONS: { id: string; icon: React.ElementType }[] = [
  { id: 'feed', icon: Home },
  { id: 'explorer', icon: Search },
  { id: 'notifications', icon: Bell },
  { id: 'messages', icon: Mail },
  { id: 'bookmarks', icon: Bookmark },
  { id: 'profil', icon: User },
  { id: 'parametres', icon: Settings },
]

/**
 * Contenu de la page d'aide : présentation de chaque section de Breezy, et un
 * appel à (re)lancer le didacticiel guidé. Styling clair/sombre via les tokens.
 */
export function HelpContent() {
  const t = useT()
  const { start } = useTutorial()

  return (
    <div className="flex flex-col">
      <div className="panel z-10 hidden border-b px-4 py-3 lg:sticky lg:top-0 lg:block">
        <h1 className="brand-text text-xl font-bold">{t('help.title')}</h1>
      </div>

      <div className="flex flex-col gap-4 px-4 py-5">
        {/* Bandeau didacticiel */}
        <section className="glass flex flex-col gap-3 rounded-2xl border p-4 backdrop-blur-xl sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-start gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-white">
              <Sparkles className="h-5 w-5" aria-hidden />
            </span>
            <div className="min-w-0">
              <h2 className="text-lg font-bold text-foreground">{t('help.tutorial_title')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('help.tutorial_desc')}</p>
            </div>
          </div>
          <Button
            type="button"
            onClick={() => start()}
            className="shrink-0 rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
          >
            {t('help.replay_tutorial')}
          </Button>
        </section>

        {/* Aide par section */}
        <section className="glass rounded-2xl border backdrop-blur-xl">
          <div className="flex items-start gap-3 border-b px-4 py-4">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
              <HelpCircle className="h-5 w-5" aria-hidden />
            </span>
            <div className="min-w-0">
              <h2 className="text-lg font-bold text-foreground">{t('help.sections_title')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('help.subtitle')}</p>
            </div>
          </div>

          <div className="divide-y">
            {SECTIONS.map(({ id, icon: Icon }) => (
              <div key={id} className="flex items-start gap-3 px-4 py-4">
                <span className="mt-1 shrink-0 text-primary">
                  <Icon className="h-5 w-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <h3 className="font-semibold text-foreground">{t(`help.section.${id}.title`)}</h3>
                  <p className="mt-1 text-sm leading-snug text-muted-foreground">
                    {t(`help.section.${id}.body`)}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </section>
      </div>
    </div>
  )
}
