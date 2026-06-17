'use client'

import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { CalendarDays, Check, CircleAlert, Loader2, User, UserPlus } from 'lucide-react'
import * as React from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useT } from '@/components/language-provider'
import { setAccessToken } from '@/lib/auth-client'
import { ROUTES } from '@/lib/routes'
import { hasReadTerms } from '@/lib/terms-consent'

const usernamePattern = /^[a-zA-Z0-9_]{3,24}$/
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

type PendingOAuth = {
  token: string
  email: string
}

function CallbackContent({ provider }: { provider: string }) {
  const t = useT()
  const searchParams = useSearchParams()
  const [error, setError] = React.useState<string | null>(null)
  const [pending, setPending] = React.useState<PendingOAuth | null>(null)
  const [username, setUsername] = React.useState('')
  const [birthDate, setBirthDate] = React.useState('')
  const [usernameError, setUsernameError] = React.useState<string>()
  const [birthDateError, setBirthDateError] = React.useState<string>()
  const [termsError, setTermsError] = React.useState<string>()
  const [termsRead, setTermsRead] = React.useState(false)
  const [acceptedTerms, setAcceptedTerms] = React.useState(false)
  const [submitting, setSubmitting] = React.useState(false)
  // Le code OAuth est à usage unique : on garantit un seul échange même sous le
  // double-montage de React StrictMode (dev), sinon le 2e POST réutilise un code
  // déjà consommé et fait flasher une fausse erreur sur un login pourtant réussi.
  const exchangeStarted = React.useRef(false)
  const todayDate = React.useMemo(() => toDateInputValue(new Date()), [])
  const minimumAgeBirthDate = React.useMemo(() => {
    const date = new Date()
    date.setFullYear(date.getFullYear() - 13)
    return toDateInputValue(date)
  }, [])

  React.useEffect(() => {
    const syncTermsRead = () => setTermsRead(hasReadTerms())
    syncTermsRead()
    window.addEventListener('focus', syncTermsRead)
    window.addEventListener('storage', syncTermsRead)
    return () => {
      window.removeEventListener('focus', syncTermsRead)
      window.removeEventListener('storage', syncTermsRead)
    }
  }, [])

  React.useEffect(() => {
    if (exchangeStarted.current) return
    exchangeStarted.current = true

    const code = searchParams.get('code')
    const state = searchParams.get('state')

    if (!code) {
      setError(t('auth.oauth.error'))
      return
    }

    async function exchange() {
      try {
        const response = await fetch(`/api/auth/oauth/${provider}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ code, state }),
        })
        const payload = await response.json().catch(() => null)

        if (!response.ok) {
          setError(payload?.error ?? t('auth.oauth.error'))
          return
        }

        if (payload?.onboardingRequired && payload?.pendingToken) {
          setPending({
            token: payload.pendingToken,
            email: payload.email ?? '',
          })
          return
        }

        if (payload?.accessToken) {
          setAccessToken(payload.accessToken)
        }

        // Navigation DURE (cf. login) : entrée cross-groupe `(auth)` → `(app)`,
        // repart sur un état d'app propre (feed persistant remonté).
        window.location.assign(ROUTES.feed)
      } catch {
        setError(t('auth.oauth.error'))
      }
    }

    exchange()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const validatePending = React.useCallback(() => {
    const next: { username?: string; birthDate?: string; terms?: string } = {}
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

    if (!acceptedTerms) {
      next.terms = termsRead
        ? t('auth.register.err.terms_required')
        : t('auth.register.err.terms_read_required')
    }

    return next
  }, [acceptedTerms, birthDate, minimumAgeBirthDate, termsRead, todayDate, username, t])

  const handlePendingSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!pending) return

    setError(null)
    const next = validatePending()
    setUsernameError(next.username)
    setBirthDateError(next.birthDate)
    setTermsError(next.terms)
    if (next.username || next.birthDate || next.terms) return

    setSubmitting(true)
    try {
      const availabilityResponse = await fetch(
        `/api/users/check-username?username=${encodeURIComponent(username.trim())}`
      )
      const availability = await availabilityResponse.json().catch(() => null)
      if (!availabilityResponse.ok) {
        setError(t('auth.register.err.username_check'))
        setSubmitting(false)
        return
      }
      if (!availability?.available) {
        setUsernameError(t('auth.register.err.username_taken'))
        setSubmitting(false)
        return
      }

      const response = await fetch(`/api/auth/oauth/${provider}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          pendingToken: pending.token,
          username: username.trim(),
          birthDate,
          acceptedTerms,
        }),
      })
      const payload = await response.json().catch(() => null)
      if (!response.ok) {
        setError(payload?.error ?? t('auth.oauth.error'))
        setSubmitting(false)
        return
      }
      if (payload?.accessToken) {
        setAccessToken(payload.accessToken)
      }
      window.location.assign(ROUTES.feed)
    } catch {
      setError(t('auth.oauth.error'))
      setSubmitting(false)
    }
  }

  if (error && !pending) {
    return (
      <main className="bg-page flex min-h-dvh items-center justify-center px-4">
        <div className="flex flex-col items-center gap-4 text-center">
          <CircleAlert className="h-10 w-10 text-red-500" />
          <p className="max-w-xs text-sm text-red-600 dark:text-red-400">{error}</p>
          <Link
            href={ROUTES.login}
            className="text-sm font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
          >
            {t('auth.oauth.back_to_login')}
          </Link>
        </div>
      </main>
    )
  }

  if (pending) {
    return (
      <main className="bg-page flex min-h-dvh items-center justify-center px-4 py-8">
        <form
          onSubmit={handlePendingSubmit}
          className="glass-strong w-full max-w-md rounded-2xl border p-6 shadow-xl"
        >
          <div className="mb-5 flex flex-col items-center gap-2 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-white">
              <UserPlus className="h-6 w-6" />
            </span>
            <h1 className="text-lg font-bold">{t('onboarding.title')}</h1>
            <p className="text-sm text-muted-foreground">
              {t('onboarding.oauth_subtitle', { email: pending.email })}
            </p>
          </div>

          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <label htmlFor="oauth-username" className="text-xs font-medium text-foreground/80">
                {t('onboarding.username_label')}
              </label>
              <div className="relative">
                <User className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="oauth-username"
                  value={username}
                  onChange={(event) => {
                    setUsername(event.target.value)
                    setUsernameError(undefined)
                  }}
                  placeholder={t('onboarding.username_placeholder')}
                  maxLength={24}
                  autoComplete="username"
                  className="pl-9"
                />
              </div>
              {usernameError ? (
                <p className="flex items-center gap-1 text-xs text-destructive">
                  <CircleAlert className="h-3 w-3" />
                  {usernameError}
                </p>
              ) : (
                <p className="text-xs text-muted-foreground">{t('onboarding.username_hint')}</p>
              )}
            </div>

            <div className="flex flex-col gap-1">
              <label htmlFor="oauth-birthdate" className="text-xs font-medium text-foreground/80">
                {t('onboarding.birthdate_label')}
              </label>
              <div className="relative">
                <CalendarDays className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="oauth-birthdate"
                  type="date"
                  value={birthDate}
                  min={minBirthDate}
                  max={todayDate}
                  onChange={(event) => {
                    setBirthDate(event.target.value)
                    setBirthDateError(undefined)
                  }}
                  className="pl-9"
                />
              </div>
              {birthDateError ? (
                <p className="flex items-center gap-1 text-xs text-destructive">
                  <CircleAlert className="h-3 w-3" />
                  {birthDateError}
                </p>
              ) : null}
            </div>

            <div className="space-y-2 rounded-2xl border bg-white/70 p-3 text-xs dark:bg-white/5">
              <p className="text-muted-foreground">
                {termsRead
                  ? t('auth.register.terms_unlocked')
                  : t('auth.register.terms_locked')}{' '}
                <Link
                  href={ROUTES.cgu}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-semibold text-[#5B6CFF] underline-offset-4 hover:underline"
                >
                  {t('auth.register.terms_link')}
                </Link>
              </p>
              <Button
                type="button"
                variant={acceptedTerms ? 'secondary' : 'outline'}
                disabled={!termsRead}
                onClick={() => {
                  setAcceptedTerms(true)
                  setTermsError(undefined)
                }}
                className="w-full rounded-full"
              >
                {acceptedTerms ? (
                  <span className="flex items-center gap-2">
                    <Check className="h-4 w-4" />
                    {t('onboarding.terms_accepted')}
                  </span>
                ) : (
                  t('onboarding.accept_terms')
                )}
              </Button>
              {termsError ? (
                <p className="flex items-center gap-1 text-xs text-destructive">
                  <CircleAlert className="h-3 w-3" />
                  {termsError}
                </p>
              ) : null}
            </div>

            {error ? (
              <p className="flex items-center gap-1 text-xs text-destructive">
                <CircleAlert className="h-3 w-3" />
                {error}
              </p>
            ) : null}

            <Button
              type="submit"
              disabled={submitting || !acceptedTerms}
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
          </div>
        </form>
      </main>
    )
  }

  return (
    <main className="bg-page flex min-h-dvh items-center justify-center">
      <div className="flex flex-col items-center gap-3">
        <Loader2 className="h-8 w-8 animate-spin text-[#5B6CFF]" />
        <p className="text-sm text-muted-foreground">{t('auth.oauth.loading')}</p>
      </div>
    </main>
  )
}

export default function CallbackPage({ params }: { params: { provider: string } }) {
  return (
    <React.Suspense
      fallback={
        <main className="bg-page flex min-h-dvh items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-[#5B6CFF]" />
        </main>
      }
    >
      <CallbackContent provider={params.provider} />
    </React.Suspense>
  )
}
