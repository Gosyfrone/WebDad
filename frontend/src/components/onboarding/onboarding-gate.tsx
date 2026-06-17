'use client'

import * as React from 'react'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import { CalendarDays, CircleAlert, Info, Loader2, User, UserPlus } from 'lucide-react'

import { getMe, getProfilMe } from '@/lib/api'
import { apiFetch, getAccessToken } from '@/lib/auth-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useT } from '@/components/language-provider'

// Règles alignées sur la page register (parité de validation username + âge).
const usernamePattern = /^(?=.{3,24}$)[a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)*$/
const reservedUsernames = new Set([
  'me',
  'admin',
  'root',
  'users',
  'by-username',
  'null',
  'undefined',
])
const minBirthDate = '1900-01-01'

function toDateInputValue(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function isDateInputValue(value: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(value)
}

type Status = 'checking' | 'needed' | 'done'
type Availability = 'idle' | 'checking' | 'available' | 'taken'

/**
 * Gate d'onboarding pour les comptes créés via OAuth (Google) : ces comptes
 * n'ont pas de document profil (le register est la seule voie qui le crée). Le
 * signal « profil absent » (`GET /profils/me` → 404) déclenche une modale
 * BLOQUANTE — montée au niveau du layout (app), donc présente sur toutes les
 * pages authentifiées, non-fermable (ni ESC, ni clic extérieur, pas de bouton
 * fermer) et re-vérifiée à chaque chargement tant qu'elle n'est pas validée.
 *
 * L'utilisateur y choisit son username (pré-rempli avec le handle auto-dérivé
 * de l'email, qu'il peut conserver ou changer) et sa date de naissance (parité
 * avec la barrière d'âge ≥ 13 ans du register). À la validation : renommage du
 * username (PATCH /users/me, seulement s'il diffère du dérivé) puis création du
 * profil (POST /profils). Aucun changement backend : tout passe par des
 * endpoints existants.
 *
 * Limite assumée : c'est un gate côté front (UX). Un blocage serveur strict
 * (gateway refusant les actions sans profil) serait un chantier séparé.
 */
export function OnboardingGate() {
  const t = useT()

  const [status, setStatus] = React.useState<Status>('checking')
  const [currentUsername, setCurrentUsername] = React.useState('')
  const [username, setUsername] = React.useState('')
  const [birthDate, setBirthDate] = React.useState('')
  const [availability, setAvailability] = React.useState<Availability>('idle')
  const [usernameError, setUsernameError] = React.useState<string>()
  const [birthDateError, setBirthDateError] = React.useState<string>()
  const [formError, setFormError] = React.useState<string>()
  const [submitting, setSubmitting] = React.useState(false)

  const todayDate = React.useMemo(() => toDateInputValue(new Date()), [])
  const minimumAgeBirthDate = React.useMemo(() => {
    const date = new Date()
    date.setFullYear(date.getFullYear() - 13)
    return toDateInputValue(date)
  }, [])

  // Détection du besoin d'onboarding : profil absent ⟺ compte OAuth non finalisé.
  React.useEffect(() => {
    // Visiteur (pas de token) : aucun compte à finaliser. Surtout, `/profils/me`
    // renverrait 401 → `apiFetch` tenterait un refresh → échec → redirection
    // forcée vers /login (cassait la vue visiteur du fil public). On ne gate jamais.
    if (!getAccessToken()) {
      setStatus('done')
      return
    }
    let active = true
    void (async () => {
      try {
        const profil = await getProfilMe()
        if (!active) return
        if (profil) {
          setStatus('done')
          return
        }
        // Pré-remplit le username avec le handle auto-dérivé (best-effort).
        let derived = ''
        try {
          derived = (await getMe()).username
        } catch {
          /* le user-service peut être momentanément indisponible : champ vide */
        }
        if (!active) return
        setCurrentUsername(derived)
        setUsername(derived)
        setStatus('needed')
      } catch {
        // Fail-open : sur erreur réseau, on ne verrouille pas l'app (re-check au
        // prochain chargement). On ne bloque que sur un 404 franc (profil absent).
        if (active) setStatus('done')
      }
    })()
    return () => {
      active = false
    }
  }, [])

  // Vérification de disponibilité du username (débounce), même contrat que le
  // register. Le handle déjà détenu par le compte est considéré disponible.
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
    if (candidate.toLowerCase() === currentUsername.toLowerCase()) {
      setAvailability('available')
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
  }, [username, currentUsername, status])

  const validate = React.useCallback(() => {
    const next: { username?: string; birthDate?: string } = {}
    const trimmed = username.trim()

    if (!trimmed) {
      next.username = t('auth.register.err.username_required')
    } else if (!usernamePattern.test(trimmed)) {
      next.username = t('auth.register.err.username_format')
    } else if (reservedUsernames.has(trimmed.toLowerCase())) {
      next.username = t('auth.register.err.username_reserved')
    }

    if (!birthDate) {
      next.birthDate = t('auth.register.err.birthdate_required')
    } else if (!isDateInputValue(birthDate) || birthDate < minBirthDate) {
      next.birthDate = t('auth.register.err.birthdate_invalid')
    } else if (birthDate > todayDate) {
      next.birthDate = t('auth.register.err.birthdate_future')
    } else if (birthDate > minimumAgeBirthDate) {
      next.birthDate = t('auth.register.err.age')
    }

    return next
  }, [birthDate, minimumAgeBirthDate, todayDate, username, t])

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setFormError(undefined)

    const next = validate()
    setUsernameError(next.username)
    setBirthDateError(next.birthDate)
    if (next.username || next.birthDate) return
    if (availability === 'taken') {
      setUsernameError(t('auth.register.err.username_taken'))
      return
    }

    const handle = username.trim()
    setSubmitting(true)
    try {
      // 1) Renommage : uniquement si le username diffère du handle auto-dérivé
      //    (le 1er changement n'est jamais bloqué par le cooldown — changedAt nil).
      if (handle.toLowerCase() !== currentUsername.toLowerCase()) {
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
          setFormError(t('onboarding.err.generic'))
          setSubmitting(false)
          return
        }
      }

      // 2) Création du profil (display_name = username, date de naissance).
      //    409 = profil déjà créé entre-temps → traité comme un succès.
      const profRes = await apiFetch('/profils', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          display_name: handle,
          birth_date: new Date(birthDate).toISOString(),
        }),
      })
      if (!profRes.ok && profRes.status !== 409) {
        setFormError(t('onboarding.err.generic'))
        setSubmitting(false)
        return
      }

      // Succès : rechargement complet pour réhydrater header, profil et caches.
      window.location.reload()
    } catch {
      setFormError(t('auth.err.network'))
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
          // Non-contournable : on neutralise ESC et tout clic/focus extérieur.
          onEscapeKeyDown={(e) => e.preventDefault()}
          onInteractOutside={(e) => e.preventDefault()}
          onPointerDownOutside={(e) => e.preventDefault()}
          className="fixed left-1/2 top-1/2 z-[101] w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border bg-background p-6 shadow-xl focus:outline-none"
        >
          <div className="mb-5 flex flex-col items-center gap-2 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-white">
              <UserPlus className="h-6 w-6" />
            </span>
            <DialogPrimitive.Title className="text-lg font-bold">
              {t('onboarding.title')}
            </DialogPrimitive.Title>
            <DialogPrimitive.Description className="text-sm text-muted-foreground">
              {t('onboarding.subtitle')}
            </DialogPrimitive.Description>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            {/* Username */}
            <div className="flex flex-col gap-1">
              <label htmlFor="onboarding-username" className="text-xs font-medium text-foreground/80">
                {t('onboarding.username_label')}
              </label>
              <div className="relative">
                <User className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="onboarding-username"
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

            {/* Date de naissance */}
            <div className="flex flex-col gap-1">
              <label htmlFor="onboarding-birthdate" className="text-xs font-medium text-foreground/80">
                {t('onboarding.birthdate_label')}
              </label>
              <div className="relative">
                <CalendarDays className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="onboarding-birthdate"
                  type="date"
                  value={birthDate}
                  min={minBirthDate}
                  max={todayDate}
                  onChange={(e) => {
                    setBirthDate(e.target.value)
                    setBirthDateError(undefined)
                  }}
                  className="pl-9"
                />
              </div>
              {birthDateError && (
                <p className="flex items-center gap-1 text-xs text-destructive">
                  <CircleAlert className="h-3 w-3" />
                  {birthDateError}
                </p>
              )}
              {/* Disclaimer discret : set-once côté profil-service + filtrage NSFW < 18 ans. */}
              <p className="flex items-start gap-1 text-[11px] leading-snug text-muted-foreground/80">
                <Info className="mt-0.5 h-3 w-3 shrink-0" />
                {t('onboarding.birthdate_disclaimer')}
              </p>
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
                  {t('onboarding.submitting')}
                </span>
              ) : (
                t('onboarding.submit')
              )}
            </Button>
          </form>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
