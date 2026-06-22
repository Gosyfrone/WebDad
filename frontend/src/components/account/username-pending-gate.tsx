'use client'

import * as React from 'react'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import { AtSign, CircleAlert, Loader2, UserCog } from 'lucide-react'

import { apiFetch, getAccessToken } from '@/lib/auth-client'
import { useCurrentUser } from '@/components/current-user-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useT } from '@/components/language-provider'

// Parité de validation username avec le register / l'onboarding.
const usernamePattern = /^(?=.{3,24}$)[a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)*$/
const reservedUsernames = new Set([
  'me',
  'admin',
  'root',
  'users',
  'by-username',
  'null',
  'undefined',
  'search',
  'suggestions',
])

type Status = 'checking' | 'needed' | 'done'
type Availability = 'idle' | 'checking' | 'available' | 'taken'

/**
 * Gate BLOQUANTE de changement de username pour les comptes dont le handle a été
 * attribué d'office (suffixé) par un admin car le nom demandé était déjà pris
 * (`username_pending`, exposé par GET /users/me). La modale impose de choisir un
 * handle DISPONIBLE avant de continuer.
 *
 * Montée au niveau du layout (app). Ne se déclenche qu'APRÈS la résolution d'un
 * éventuel mot de passe temporaire (on attend `mustChangePassword === false`)
 * pour ne jamais empiler les deux modales. À la validation : PATCH /users/me
 * (le back repasse `username_pending` à false sur un changement effectif), puis
 * rechargement complet pour réhydrater header / profil / caches.
 */
export function UsernamePendingGate() {
  const t = useT()
  const { session, usernamePending, profil } = useCurrentUser()

  const [status, setStatus] = React.useState<Status>('checking')
  const [currentUsername, setCurrentUsername] = React.useState('')
  const [username, setUsername] = React.useState('')
  const [availability, setAvailability] = React.useState<Availability>('idle')
  const [usernameError, setUsernameError] = React.useState<string>()
  const [formError, setFormError] = React.useState<string>()
  const [submitting, setSubmitting] = React.useState(false)

  // Username provisoire (depuis le store) : on n'active la modale qu'une fois le
  // mot de passe temporaire réglé ET les CGU acceptées (sinon ces modales priment,
  // chaîne mot de passe → CGU → username — jamais d'empilement).
  React.useEffect(() => {
    if (!getAccessToken() || session?.mustChangePassword || !session?.termsAccepted) {
      setStatus('done')
      return
    }
    if (usernamePending) {
      setCurrentUsername(profil?.username ?? '')
      setStatus('needed')
    } else {
      setStatus('done')
    }
  }, [session?.mustChangePassword, session?.termsAccepted, usernamePending, profil?.username])

  // Disponibilité du username (débounce), même contrat que le register.
  React.useEffect(() => {
    if (status !== 'needed') return
    const candidate = username.trim()
    if (
      !candidate ||
      !usernamePattern.test(candidate) ||
      reservedUsernames.has(candidate.toLowerCase())
    ) {
      setAvailability('idle')
      return
    }
    setAvailability('checking')
    const handle = setTimeout(async () => {
      try {
        const res = await fetch(
          `/api/users/check-username?username=${encodeURIComponent(candidate)}`,
        )
        const data = (await res.json().catch(() => null)) as { available?: boolean } | null
        if (!res.ok) {
          setAvailability('idle')
          return
        }
        setAvailability(data?.available ? 'available' : 'taken')
      } catch {
        setAvailability('idle')
      }
    }, 400)
    return () => clearTimeout(handle)
  }, [username, status])

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setFormError(undefined)

    const handle = username.trim()
    if (!handle) {
      setUsernameError(t('auth.register.err.username_required'))
      return
    }
    if (!usernamePattern.test(handle)) {
      setUsernameError(t('auth.register.err.username_format'))
      return
    }
    if (reservedUsernames.has(handle.toLowerCase())) {
      setUsernameError(t('auth.register.err.username_reserved'))
      return
    }
    if (handle.toLowerCase() === currentUsername.toLowerCase() || availability === 'taken') {
      setUsernameError(t('auth.register.err.username_taken'))
      return
    }

    setSubmitting(true)
    try {
      const res = await apiFetch('/users/me', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: handle }),
      })
      if (res.status === 409) {
        setUsernameError(t('auth.register.err.username_taken'))
        setSubmitting(false)
        return
      }
      if (!res.ok) {
        setFormError(t('account.username_pending.err.generic'))
        setSubmitting(false)
        return
      }
      // Succès : rechargement complet pour réhydrater header, profil et caches.
      window.location.reload()
    } catch {
      setFormError(t('account.username_pending.err.generic'))
      setSubmitting(false)
    }
  }

  if (status !== 'needed') return null

  const canSubmit = !submitting && availability !== 'checking' && availability !== 'taken'

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
              <UserCog className="h-6 w-6" />
            </span>
            <DialogPrimitive.Title className="text-lg font-bold">
              {t('account.username_pending.title')}
            </DialogPrimitive.Title>
            <DialogPrimitive.Description className="text-sm text-muted-foreground">
              {t('account.username_pending.subtitle', { username: currentUsername })}
            </DialogPrimitive.Description>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <label htmlFor="username-pending" className="text-xs font-medium text-foreground/80">
                {t('account.username_pending.label')}
              </label>
              <div className="relative">
                <AtSign className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="username-pending"
                  value={username}
                  onChange={(e) => {
                    setUsername(e.target.value)
                    setUsernameError(undefined)
                  }}
                  placeholder={t('onboarding.username_placeholder')}
                  maxLength={24}
                  autoComplete="off"
                  className="pl-9"
                />
              </div>
              {usernameError ? (
                <p className="flex items-center gap-1 text-xs text-destructive">
                  <CircleAlert className="h-3 w-3" />
                  {usernameError}
                </p>
              ) : availability === 'checking' ? (
                <p className="flex items-center gap-1 text-xs text-muted-foreground">
                  <Loader2 className="h-3 w-3 animate-spin" />
                  {t('onboarding.username_checking')}
                </p>
              ) : availability === 'taken' ? (
                <p className="flex items-center gap-1 text-xs text-destructive">
                  <CircleAlert className="h-3 w-3" />
                  {t('auth.register.err.username_taken')}
                </p>
              ) : availability === 'available' ? (
                <p className="text-xs font-medium text-emerald-600 dark:text-emerald-400">
                  {t('onboarding.username_available')}
                </p>
              ) : (
                <p className="text-xs text-muted-foreground">{t('onboarding.username_hint')}</p>
              )}
            </div>

            {formError && (
              <p className="flex items-center gap-1 text-xs text-destructive">
                <CircleAlert className="h-3 w-3" />
                {formError}
              </p>
            )}

            <Button
              type="submit"
              disabled={!canSubmit}
              className="mt-1 w-full rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
            >
              {submitting ? (
                <span className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  {t('account.username_pending.submitting')}
                </span>
              ) : (
                t('account.username_pending.submit')
              )}
            </Button>
          </form>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
