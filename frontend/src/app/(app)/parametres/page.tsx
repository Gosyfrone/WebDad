'use client'

import { Languages, Palette, Settings } from 'lucide-react'

import { useT } from '@/components/language-provider'
import { LanguageSelector } from '@/components/language-selector'
import { ThemeToggle } from '@/components/theme-toggle'

export default function ParametresPage() {
  const t = useT()

  return (
    <div className="flex flex-col">
      <div className="panel z-10 hidden border-b px-4 py-3 lg:sticky lg:top-0 lg:block">
        <h1 className="brand-text text-xl font-bold">{t('settings.title')}</h1>
      </div>

      <div className="flex flex-col gap-4 px-4 py-5">
        <section className="glass rounded-2xl border backdrop-blur-xl">
          <div className="flex items-start gap-3 border-b px-4 py-4">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
              <Settings className="h-5 w-5" aria-hidden />
            </span>
            <div className="min-w-0">
              <h2 className="text-lg font-bold text-foreground">
                {t('settings.account_title')}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('settings.account_desc')}
              </p>
            </div>
          </div>

          <div className="divide-y">
            <div className="grid gap-4 px-4 py-4 md:grid-cols-[minmax(0,1fr)_minmax(260px,340px)] md:items-start">
              <div className="flex items-start gap-3">
                <span className="mt-1 text-primary">
                  <Languages className="h-5 w-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <h3 className="font-semibold text-foreground">{t('lang.title')}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {t('lang.select_aria')}
                  </p>
                </div>
              </div>

              <div className="panel rounded-xl border px-2 py-2 shadow-sm">
                <LanguageSelector />
              </div>
            </div>

            <div className="grid gap-4 px-4 py-4 md:grid-cols-[minmax(0,1fr)_minmax(260px,340px)] md:items-start">
              <div className="flex items-start gap-3">
                <span className="mt-1 text-primary">
                  <Palette className="h-5 w-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <h3 className="font-semibold text-foreground">{t('theme.title')}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {t('theme.appearance')}
                  </p>
                </div>
              </div>

              <div className="panel rounded-xl border px-2 py-2 shadow-sm">
                <ThemeToggle />
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}
