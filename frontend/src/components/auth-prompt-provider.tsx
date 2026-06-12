'use client'

/**
 * Garde d'authentification pour le **mode visiteur** (utilisateur non connecté
 * qui consulte le fil public).
 *
 * Le backend autorise déjà la lecture publique du fil et des posts
 * (`OptionalJWTAuth` côté post-service) : un visiteur n'a pas de token. Ce
 * provider centralise deux choses pour l'UI :
 *   - `isVisitor` : vrai APRÈS montage et en l'absence de token (le rendu par
 *     défaut suppose « connecté » pour éviter un flash de l'UI visiteur chez les
 *     membres, même compromis d'hydratation que le thème / la session) ;
 *   - `requireAuth(action)` : enveloppe un handler d'action réservée (aimer,
 *     commenter, reposter, suivre, publier…). Si pas de token → ouvre une modale
 *     « Connecte-toi » ; sinon exécute l'action.
 *
 * ⚠️ Client uniquement (lit `localStorage` via `getAccessToken`).
 */

import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import Link from 'next/link'

import { getAccessToken } from '@/lib/auth-client'
import { SESSION_CHANGED_EVENT } from '@/lib/session'
import { ROUTES } from '@/lib/routes'
import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface AuthGateValue {
  /** L'utilisateur est-il un visiteur (monté + aucun token) ? */
  isVisitor: boolean
  /** Ouvre directement la modale d'invitation à se connecter. */
  promptLogin: () => void
  /**
   * Enveloppe une action réservée : renvoie un handler qui exécute `action` si
   * un token est présent, sinon ouvre la modale. Le test du token est fait au
   * moment du clic (synchrone, jamais périmé).
   */
  requireAuth: <A extends unknown[]>(action: (...args: A) => void) => (...args: A) => void
}

const AuthGateContext = createContext<AuthGateValue | null>(null)

/** Accès à la garde d'authentification (mode visiteur). */
export function useAuthGate(): AuthGateValue {
  const ctx = useContext(AuthGateContext)
  if (!ctx) {
    throw new Error('useAuthGate doit être utilisé dans <AuthPromptProvider>')
  }
  return ctx
}

export function AuthPromptProvider({ children }: { children: React.ReactNode }) {
  const t = useT()
  const [open, setOpen] = useState(false)
  // `false` au 1er rendu (SSR + hydratation : pas d'accès localStorage) → l'UI
  // membre s'affiche par défaut, puis bascule en mode visiteur au montage.
  const [isVisitor, setIsVisitor] = useState(false)

  useEffect(() => {
    const sync = () => setIsVisitor(!getAccessToken())
    sync()
    window.addEventListener(SESSION_CHANGED_EVENT, sync)
    window.addEventListener('storage', sync)
    return () => {
      window.removeEventListener(SESSION_CHANGED_EVENT, sync)
      window.removeEventListener('storage', sync)
    }
  }, [])

  const promptLogin = useCallback(() => setOpen(true), [])

  const requireAuth = useCallback(
    <A extends unknown[]>(action: (...args: A) => void) =>
      (...args: A) => {
        if (!getAccessToken()) {
          setOpen(true)
          return
        }
        action(...args)
      },
    [],
  )

  return (
    <AuthGateContext.Provider value={{ isVisitor, promptLogin, requireAuth }}>
      {children}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="panel border sm:max-w-sm">
          <DialogHeader>
            <DialogTitle className="brand-text text-xl font-bold">
              {t('visitor.prompt_title')}
            </DialogTitle>
            <DialogDescription>{t('visitor.prompt_desc')}</DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-2">
            <Button asChild variant="outline" className="flex-1">
              <Link href={ROUTES.login}>{t('visitor.login')}</Link>
            </Button>
            <Button
              asChild
              className="flex-1 bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
            >
              <Link href={ROUTES.register}>{t('visitor.register')}</Link>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </AuthGateContext.Provider>
  )
}
