'use client'

import * as React from 'react'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import { CircleAlert, KeyRound, Loader2, Lock } from 'lucide-react'

import { getAccessToken, setAccessToken } from '@/lib/auth-client'
import { useSession } from '@/lib/session'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useT } from '@/components/language-provider'

/**
 * Gate BLOQUANTE de changement de mot de passe pour les comptes créés par un
 * admin avec un mot de passe TEMPORAIRE. Le signal vient directement du JWT
 * (`must_change_password`, lu par useSession) : la modale s'affiche dès la
 * première connexion, sur n'importe quelle page authentifiée, sans rechargement.
 *
 * Montée au niveau du layout (app), non-fermable (ni ESC, ni clic extérieur).
 * À la validation : POST /api/auth/password/change (BFF) → le back vérifie le
 * mot de passe temporaire, pose le nouveau, révoque les autres sessions et
 * ré-émet une paire de tokens SANS le drapeau. On stocke le nouvel access token
 * → la session se met à jour et la modale disparaît automatiquement.
 */
export function PasswordChangeGate() {
  const t = useT()
  const session = useSession()

  const [current, setCurrent] = React.useState('')
  const [next, setNext] = React.useState('')
  const [confirm, setConfirm] = React.useState('')
  const [error, setError] = React.useState<string>()
  const [submitting, setSubmitting] = React.useState(false)

  // Visiteur (pas de token) ou compte normal : aucune contrainte.
  if (!session?.mustChangePassword) return null

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(undefined)

    if (next.length < 8) {
      setError(t('account.password_change.hint'))
      return
    }
    if (next !== confirm) {
      setError(t('account.password_change.err.mismatch'))
      return
    }
    if (next === current) {
      setError(t('account.password_change.err.same'))
      return
    }

    setSubmitting(true)
    try {
      const res = await fetch('/api/auth/password/change', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getAccessToken() ?? ''}`,
        },
        body: JSON.stringify({ current_password: current, new_password: next }),
      })
      const payload = (await res.json().catch(() => null)) as
        | { accessToken?: string; code?: string }
        | null
      if (!res.ok) {
        setError(
          payload?.code === 'invalid_current_password'
            ? t('account.password_change.err.current')
            : t('account.password_change.err.generic'),
        )
        setSubmitting(false)
        return
      }
      // Nouveau token sans le drapeau → la session se rafraîchit, la modale se ferme.
      if (payload?.accessToken) {
        setAccessToken(payload.accessToken)
      }
    } catch {
      setError(t('account.password_change.err.generic'))
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
              <KeyRound className="h-6 w-6" />
            </span>
            <DialogPrimitive.Title className="text-lg font-bold">
              {t('account.password_change.title')}
            </DialogPrimitive.Title>
            <DialogPrimitive.Description className="text-sm text-muted-foreground">
              {t('account.password_change.subtitle')}
            </DialogPrimitive.Description>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <PasswordField
              id="pc-current"
              label={t('account.password_change.current_label')}
              value={current}
              onChange={(v) => {
                setCurrent(v)
                setError(undefined)
              }}
            />
            <PasswordField
              id="pc-new"
              label={t('account.password_change.new_label')}
              value={next}
              onChange={(v) => {
                setNext(v)
                setError(undefined)
              }}
            />
            <PasswordField
              id="pc-confirm"
              label={t('account.password_change.confirm_label')}
              value={confirm}
              onChange={(v) => {
                setConfirm(v)
                setError(undefined)
              }}
            />

            {error ? (
              <p className="flex items-center gap-1 text-xs text-destructive">
                <CircleAlert className="h-3 w-3" />
                {error}
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">{t('account.password_change.hint')}</p>
            )}

            <Button
              type="submit"
              disabled={submitting}
              className="mt-1 w-full rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
            >
              {submitting ? (
                <span className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  {t('account.password_change.submitting')}
                </span>
              ) : (
                t('account.password_change.submit')
              )}
            </Button>
          </form>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}

function PasswordField({
  id,
  label,
  value,
  onChange,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-foreground/80">
        {label}
      </label>
      <div className="relative">
        <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          id={id}
          type="password"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          autoComplete="off"
          className="pl-9"
        />
      </div>
    </div>
  )
}
