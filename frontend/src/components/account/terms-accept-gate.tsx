'use client'

import * as React from 'react'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import { CircleAlert, FileText, Loader2 } from 'lucide-react'

import { getAccessToken, logout, setAccessToken } from '@/lib/auth-client'
import { useCurrentUser } from '@/components/current-user-provider'
import { Button } from '@/components/ui/button'
import { useT } from '@/components/language-provider'
import { ROUTES } from '@/lib/routes'

/**
 * Gate BLOQUANTE d'acceptation des CGU. Le signal vient directement du JWT
 * (`terms_accepted`, lu par useSession) : la modale s'affiche dès la connexion,
 * sur n'importe quelle page authentifiée, sans rechargement. Elle vise les
 * comptes qui n'ont jamais consenti explicitement (tous les comptes déjà en
 * base au déploiement, terms_accepted_version=0) ou dont le consentement est
 * antérieur à la version EN VIGUEUR des CGU (rebump de CurrentTermsVersion).
 *
 * Montée au niveau du layout (app), non-fermable (ni ESC, ni clic extérieur).
 * Deux issues :
 *   - « J'accepte » → POST /api/auth/terms/accept (BFF) → le back enregistre le
 *     consentement, ré-émet une paire de tokens SANS le drapeau. On stocke le
 *     nouvel access token → la session se met à jour et la modale disparaît.
 *   - « Refuser » → déconnexion (logout) : consentement libre, pas de blocage
 *     définitif.
 */
export function TermsAcceptGate() {
  const t = useT()
  const { session } = useCurrentUser()

  const [error, setError] = React.useState<string>()
  const [submitting, setSubmitting] = React.useState(false)

  // Visiteur (pas de session) ou CGU déjà acceptées : aucune contrainte. On
  // s'efface aussi tant qu'un changement de mot de passe est imposé pour ne
  // jamais empiler les deux modales (PasswordChangeGate prime ; UsernamePendingGate
  // attend ensuite l'acceptation des CGU → chaîne mot de passe → CGU → username).
  if (!session || session.mustChangePassword || session.termsAccepted) return null

  const handleAccept = async () => {
    setError(undefined)
    setSubmitting(true)
    try {
      const res = await fetch('/api/auth/terms/accept', {
        method: 'POST',
        headers: { Authorization: `Bearer ${getAccessToken() ?? ''}` },
      })
      const payload = (await res.json().catch(() => null)) as
        | { accessToken?: string }
        | null
      if (!res.ok) {
        setError(t('account.terms_gate.err'))
        setSubmitting(false)
        return
      }
      // Nouveau token avec terms_accepted=true → la session se rafraîchit, la modale se ferme.
      if (payload?.accessToken) {
        setAccessToken(payload.accessToken)
      }
    } catch {
      setError(t('account.terms_gate.err'))
      setSubmitting(false)
    }
  }

  return (
    <DialogPrimitive.Root open>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-[100] bg-black/80 backdrop-blur-sm" />
        <DialogPrimitive.Content
          onEscapeKeyDown={(e) => e.preventDefault()}
          onInteractOutside={(e) => e.preventDefault()}
          onPointerDownOutside={(e) => e.preventDefault()}
          className="fixed left-1/2 top-1/2 z-[101] w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border bg-background p-6 shadow-xl focus:outline-none"
        >
          <div className="mb-5 flex flex-col items-center gap-2 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-white">
              <FileText className="h-6 w-6" />
            </span>
            <DialogPrimitive.Title className="text-lg font-bold">
              {t('account.terms_gate.title')}
            </DialogPrimitive.Title>
            <DialogPrimitive.Description className="text-sm text-muted-foreground">
              {t('account.terms_gate.subtitle')}
            </DialogPrimitive.Description>
          </div>

          <p className="mb-4 text-sm text-foreground/80">{t('account.terms_gate.body')}</p>

          <div className="mb-5 flex flex-wrap justify-center gap-x-4 gap-y-1 text-sm">
            <a
              href={ROUTES.cgu}
              target="_blank"
              rel="noopener noreferrer"
              className="font-medium text-[var(--brand-via)] underline-offset-2 hover:underline"
            >
              {t('account.terms_gate.read_cgu')}
            </a>
            <a
              href={ROUTES.confidentialite}
              target="_blank"
              rel="noopener noreferrer"
              className="font-medium text-[var(--brand-via)] underline-offset-2 hover:underline"
            >
              {t('account.terms_gate.read_privacy')}
            </a>
          </div>

          {error ? (
            <p className="mb-3 flex items-center justify-center gap-1 text-xs text-destructive">
              <CircleAlert className="h-3 w-3" />
              {error}
            </p>
          ) : null}

          <div className="flex flex-col gap-2">
            <Button
              type="button"
              onClick={() => void handleAccept()}
              disabled={submitting}
              className="w-full rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
            >
              {submitting ? (
                <span className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  {t('account.terms_gate.accepting')}
                </span>
              ) : (
                t('account.terms_gate.accept')
              )}
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={() => void logout()}
              disabled={submitting}
              className="w-full rounded-full text-sm text-muted-foreground"
            >
              {t('account.terms_gate.decline')}
            </Button>
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
