'use client'

import * as React from 'react'

import {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  isLocale,
  localeFromBrowser,
  translate,
  type Locale,
} from '@/lib/i18n'
import { getAccessToken } from '@/lib/auth-client'
import { getMe, updatePreferredLocale } from '@/lib/api'
import { SESSION_CHANGED_EVENT } from '@/lib/session'

interface LanguageContextValue {
  /** Locale effective (DEFAULT_LOCALE tant que le client n'est pas monté). */
  locale: Locale
  /** Change la langue : met à jour l'état, persiste et pose <html lang>. */
  setLocale: (locale: Locale) => void
  /** Traduit une clé pour la locale courante (avec interpolation `{param}`). */
  t: (key: string, params?: Record<string, string | number>) => string
}

const LanguageContext = React.createContext<LanguageContextValue | null>(null)

/**
 * Fournit la langue de l'application (analogue à ThemeProvider pour next-themes).
 *
 * Hydratation : on rend TOUJOURS DEFAULT_LOCALE au premier rendu (serveur + 1er
 * rendu client) pour que les deux coïncident, PUIS on lit la préférence
 * localStorage dans un effet et on re-rend dans la bonne langue. Pas de mismatch
 * d'hydratation, au prix d'un bref flash (même compromis que le thème).
 */
export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = React.useState<Locale>(DEFAULT_LOCALE)

  const applyLocale = React.useCallback((next: Locale) => {
    setLocaleState(next)
    document.documentElement.lang = next
    document.documentElement.dir = next === 'ar' ? 'rtl' : 'ltr'
  }, [])

  React.useEffect(() => {
    let syncID = 0

    const sync = async () => {
      const currentSyncID = ++syncID
      const browserLocale = localeFromBrowser(window.navigator.language)

      if (!getAccessToken()) {
        let anonymousLocale = browserLocale
        try {
          const stored = window.localStorage.getItem(LOCALE_STORAGE_KEY)
          if (isLocale(stored)) anonymousLocale = stored
        } catch {
          // localStorage indisponible : langue navigateur.
        }
        applyLocale(anonymousLocale)
        return
      }

      try {
        const me = await getMe()
        if (currentSyncID !== syncID) return
        applyLocale(isLocale(me.preferredLocale) ? me.preferredLocale : browserLocale)
      } catch {
        if (currentSyncID === syncID) applyLocale(browserLocale)
      }
    }

    void sync()
    window.addEventListener(SESSION_CHANGED_EVENT, sync)
    return () => {
      syncID += 1
      window.removeEventListener(SESSION_CHANGED_EVENT, sync)
    }
  }, [applyLocale])

  const setLocale = React.useCallback((next: Locale) => {
    applyLocale(next)

    if (getAccessToken()) {
      void updatePreferredLocale(next).catch(() => {
        // L'état reste appliqué pour la session ; la prochaine connexion
        // rechargera la dernière préférence effectivement enregistrée.
      })
      return
    }

    try {
      window.localStorage.setItem(LOCALE_STORAGE_KEY, next)
    } catch {
      // Persistance anonyme best-effort.
    }
  }, [applyLocale])

  const value = React.useMemo<LanguageContextValue>(
    () => ({
      locale,
      setLocale,
      t: (key, params) => translate(locale, key, params),
    }),
    [locale, setLocale],
  )

  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>
}

/** Accès au contexte de langue. Erreur explicite si hors provider. */
export function useLanguage(): LanguageContextValue {
  const ctx = React.useContext(LanguageContext)
  if (!ctx) {
    throw new Error('useLanguage doit être utilisé dans <LanguageProvider>')
  }
  return ctx
}

/**
 * Hook de traduction : `const t = useT(); t('nav.feed')`.
 * Raccourci sur `useLanguage().t`, l'usage le plus courant dans les composants.
 */
export function useT(): LanguageContextValue['t'] {
  return useLanguage().t
}
