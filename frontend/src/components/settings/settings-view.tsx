'use client'

import {
  EyeOff,
  Languages,
  Lock,
  Scale,
  Settings,
  ShieldAlert,
  UserX,
} from 'lucide-react'

import { useT } from '@/components/language-provider'
import { LanguageSelector } from '@/components/language-selector'
import { LegalLinks } from '@/components/legal/legal-links'
import { BlockedUsersSettings } from '@/components/settings/blocked-users-settings'
import { MutedWordsSettings } from '@/components/settings/muted-words-settings'
import { NsfwSettings } from '@/components/settings/nsfw-settings'
import { VisibilitySettings } from '@/components/settings/visibility-settings'
import { UserAccountSettings } from '@/components/settings/user-account-settings'

export function SettingsView() {
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
                {t('settings.general_title')}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('settings.general_desc')}
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
                  <Lock className="h-5 w-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <h3 className="font-semibold text-foreground">{t('visibility.title')}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {t('visibility.desc')}
                  </p>
                </div>
              </div>

              <VisibilitySettings />
            </div>

            <div className="grid gap-4 px-4 py-4 md:grid-cols-[minmax(0,1fr)_minmax(260px,340px)] md:items-start">
              <div className="flex items-start gap-3">
                <span className="mt-1 text-primary">
                  <ShieldAlert className="h-5 w-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <h3 className="font-semibold text-foreground">{t('nsfw.title')}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {t('nsfw.desc')}
                  </p>
                </div>
              </div>

              <NsfwSettings />
            </div>

            <div className="grid gap-4 px-4 py-4 md:grid-cols-[minmax(0,1fr)_minmax(260px,340px)] md:items-start">
              <div className="flex items-start gap-3">
                <span className="mt-1 text-primary">
                  <EyeOff className="h-5 w-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <h3 className="font-semibold text-foreground">{t('filters.title')}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {t('filters.desc')}
                  </p>
                </div>
              </div>

              <MutedWordsSettings />
            </div>

            <div className="grid gap-4 px-4 py-4 md:grid-cols-[minmax(0,1fr)_minmax(260px,340px)] md:items-start">
              <div className="flex items-start gap-3">
                <span className="mt-1 text-primary">
                  <UserX className="h-5 w-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <h3 className="font-semibold text-foreground">{t('block.settings_title')}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {t('block.settings_desc')}
                  </p>
                </div>
              </div>

              <BlockedUsersSettings />
            </div>
          </div>
        </section>

        <section className="glass rounded-2xl border backdrop-blur-xl">
          <div className="flex items-start gap-3 border-b px-4 py-4">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
              <Lock className="h-5 w-5" aria-hidden />
            </span>
            <div className="min-w-0">
              <h2 className="text-lg font-bold text-foreground">{t('settings.user_title')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('settings.user_desc')}</p>
            </div>
          </div>
          <UserAccountSettings />
        </section>
      </div>

      {/* Section Légal (point d'accès principal aux pages légales sur mobile) */}
      <div className="glass mx-4 mb-6 mt-4 flex flex-col gap-3 rounded-[26px] border px-6 py-6 backdrop-blur-xl">
        <h2 className="flex items-center gap-2 text-base font-bold text-foreground">
          <Scale className="h-4 w-4 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
          {t('legal.section')}
        </h2>
        <LegalLinks className="text-sm" />
      </div>
    </div>
  )
}
