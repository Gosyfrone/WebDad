'use client'

import Image from 'next/image'
import Link from 'next/link'
import {
  AtSign,
  Bell,
  CircleAlert,
  Eye,
  EyeOff,
  Heart,
  Loader2,
  Lock,
  MessageCircle,
  Search,
  Sparkles,
  UserPlus,
} from 'lucide-react'
import * as React from 'react'

// Fournisseurs sociaux (icône seule, même taille). `id` = segment de route
// (/auth/callback/<id>) ; l'ordre dicte l'affichage de la rangée.
const OAUTH_PROVIDERS = [
  { id: 'google', src: '/google-logo.jpg', label: 'Google' },
  { id: 'github', src: '/github.svg', label: 'GitHub' },
] as const

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { LegalLinks } from '@/components/legal/legal-links'
import { setAccessToken } from '@/lib/auth-client'
import { ROUTES } from '@/lib/routes'
import { useT } from '@/components/language-provider'

type FormErrors = Partial<{
  identifier: string
  password: string
  form: string
}>

function getMessage(error: unknown, fallback: string): string {
  if (typeof error === 'string' && error.trim()) return error
  if (error instanceof Error && error.message.trim()) return error.message
  return fallback
}

/**
 * Destination après connexion : le `?next=<chemin>` posé par le middleware (ex.
 * lien d'invitation `/messages?join=<id>`), sinon le feed. On n'accepte qu'un
 * chemin INTERNE absolu (`/…`) — jamais `//evil`, `/\evil` ni une URL absolue —
 * pour fermer tout open-redirect.
 */
function postLoginTarget(): string {
  if (typeof window === 'undefined') return ROUTES.feed
  const raw = new URLSearchParams(window.location.search).get('next')
  if (raw && raw.startsWith('/') && raw[1] !== '/' && raw[1] !== '\\') return raw
  return ROUTES.feed
}

export default function LoginPage() {
  const t = useT()
  const [identifier, setIdentifier] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [showPassword, setShowPassword] = React.useState(false)
  const [errors, setErrors] = React.useState<FormErrors>({})
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  // Compte non vérifié : on bascule sur une bannière avec renvoi du mail.
  const [needsVerification, setNeedsVerification] = React.useState(false)
  const [isResending, setIsResending] = React.useState(false)
  const [resendNotice, setResendNotice] = React.useState<string | null>(null)
  const [oauthLoading, setOauthLoading] = React.useState<string | null>(null)
  // MFA : si le mot de passe est bon mais la double auth active, le BFF renvoie
  // un challenge. On bascule alors sur l'écran de saisie du code (TOTP).
  const [mfaChallenge, setMfaChallenge] = React.useState<string | null>(null)
  const [mfaCode, setMfaCode] = React.useState('')
  const [mfaError, setMfaError] = React.useState<string | null>(null)
  const [mfaBusy, setMfaBusy] = React.useState(false)

  const validate = React.useCallback((): FormErrors => {
    const nextErrors: FormErrors = {}
    const trimmedIdentifier = identifier.trim()

    if (!trimmedIdentifier) {
      nextErrors.identifier = t('auth.err.identifier_required')
    }

    if (!password) {
      nextErrors.password = t('auth.err.password_required')
    } else if (password.length < 8) {
      nextErrors.password = t('auth.err.password_min')
    }

    return nextErrors
  }, [identifier, password, t])

  const handleOAuth = async (provider: string) => {
    setOauthLoading(provider)
    try {
      const response = await fetch(`/api/auth/oauth/${provider}`)
      const payload = await response.json().catch(() => null)
      if (!response.ok || !payload?.url) {
        setErrors({ form: payload?.error ?? t('auth.oauth.error') })
        setOauthLoading(null)
        return
      }
      window.location.href = payload.url
    } catch {
      setErrors({ form: t('auth.err.network') })
      setOauthLoading(null)
    }
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const nextErrors = validate()
    setErrors(nextErrors)

    if (Object.keys(nextErrors).length > 0) return

    setIsSubmitting(true)
    setNeedsVerification(false)
    setResendNotice(null)

    try {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          identifier: identifier.trim(),
          password,
        }),
      })

      const payload = await response.json().catch(() => null)

      if (!response.ok) {
        // E-mail non vérifié : bannière dédiée + bouton de renvoi (voie de
        // secours des comptes existants verrouillés par la Phase 1).
        if (payload?.code === 'email_not_verified') {
          setNeedsVerification(true)
          setErrors({})
          return
        }
        // 403 = compte désactivé/banni (auth-service ErrUserInactive). On ne
        // remonte pas le message brut du back (« compte désactivé » minuscule) :
        // on affiche un message dédié, localisé, qui oriente vers le support.
        const form =
          response.status === 403
            ? t('auth.login.account_disabled')
            : (payload?.error ?? payload?.message ?? t('auth.login.failed'))
        setErrors({ form })
        return
      }

      // MFA active : aucun token encore. On bascule sur l'écran de code ; la
      // session sera ouverte par /api/auth/mfa/verify.
      if (payload?.mfaRequired && payload?.challenge) {
        setMfaChallenge(payload.challenge)
        setMfaCode('')
        setMfaError(null)
        return
      }

      // L'access token court (5 min) vit en localStorage ; le refresh token
      // a été posé en cookie httpOnly par le BFF (/api/auth/login).
      if (payload?.accessToken) {
        setAccessToken(payload.accessToken)
      }

      // Navigation DURE vers la cible (`?next=` ou feed, et non `router.replace`
      // soft) : on entre dans l'espace `(app)` depuis le groupe `(auth)` avec un
      // access token tout juste posé. Un chargement complet repart sur un état
      // d'app propre (feed persistant remonté, providers réinitialisés).
      window.location.assign(postLoginTarget())
    } catch (error) {
      setErrors({
        form: getMessage(error, t('auth.err.network')),
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleResendVerification = async () => {
    const trimmedEmail = identifier.trim()
    if (!trimmedEmail) {
      setErrors({ identifier: t('auth.err.identifier_required') })
      return
    }

    setIsResending(true)
    setResendNotice(null)
    try {
      await fetch('/api/auth/verify-email/resend', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: trimmedEmail }),
      })
      // Réponse générique (anti-énumération) : on affiche toujours le même
      // message, succès comme échec réseau.
      setResendNotice(t('auth.verify.resend_done'))
    } catch {
      setResendNotice(t('auth.verify.resend_done'))
    } finally {
      setIsResending(false)
    }
  }

  const handleMfaVerify = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!mfaChallenge || !mfaCode.trim()) return

    setMfaBusy(true)
    setMfaError(null)
    try {
      const response = await fetch('/api/auth/mfa/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ challenge: mfaChallenge, code: mfaCode.trim() }),
      })
      const payload = await response.json().catch(() => null)

      if (!response.ok) {
        // Challenge expiré/invalide → on renvoie à l'écran mot de passe.
        if (payload?.code === 'invalid_challenge') {
          setMfaChallenge(null)
          setErrors({ form: t('auth.mfa.error_challenge') })
          return
        }
        setMfaError(t('auth.mfa.error_code'))
        return
      }

      if (payload?.accessToken) {
        setAccessToken(payload.accessToken)
      }
      window.location.assign(postLoginTarget())
    } catch (error) {
      setMfaError(getMessage(error, t('auth.err.network')))
    } finally {
      setMfaBusy(false)
    }
  }

  return (
    <main className="bg-page relative flex min-h-dvh w-full items-center justify-center overflow-hidden px-4 py-5 sm:px-6 lg:px-10">
      <div className="bg-page-glow-1 pointer-events-none absolute inset-0" />
      <div className="bg-page-glow-2 pointer-events-none absolute inset-0" />
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-white/70 dark:bg-white/10" />

      <section className="relative grid w-full max-w-6xl items-center gap-6 lg:grid-cols-[1.08fr_0.92fr]">
        <div className="hidden min-h-[620px] flex-col justify-between lg:flex">
          <Link
            href={ROUTES.home}
            className="group relative inline-flex w-fit items-center transition duration-300 hover:scale-[1.03]"
          >
            <span className="absolute inset-0 bg-gradient-to-r from-[#8D3DFF]/30 to-[#47D9FF]/25 blur-2xl" />

            <Image
              src="/logo_breezy.png"
              alt="Breezy"
              width={1106}
              height={336}
              className="relative h-12 w-auto object-contain drop-shadow-sm"
              priority
            />
          </Link>

          <div className="relative mt-8 h-[500px]">
            <div className="glass absolute left-8 top-0 w-[410px] overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(91,108,255,0.28)] backdrop-blur-2xl">
              <div className="flex items-center justify-between border-b border-white/60 px-5 py-4 dark:border-white/10">
                <div>
                  <p className="text-xs font-semibold uppercase text-[#5B6CFF]">
                    {t('auth.login.demo.kicker')}
                  </p>
                  <h1 className="text-2xl font-semibold text-foreground">
                    {t('auth.login.demo.heading')}
                  </h1>
                </div>

                <button
                  type="button"
                  aria-label={t('auth.search_aria')}
                  className="grid h-10 w-10 place-items-center rounded-full bg-white/90 text-foreground/80 shadow-sm transition hover:scale-105 hover:text-[#5B6CFF] dark:bg-white/10"
                >
                  <Search className="h-5 w-5" />
                </button>
              </div>

              <div className="space-y-1 px-4 py-4">
                <article className="glass rounded-[22px] border p-4 shadow-sm">
                  <div className="flex gap-3">
                    <div className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-gradient-to-br from-[#8D3DFF] to-[#47D9FF] text-sm font-bold text-white">
                      ML
                    </div>

                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1 text-sm">
                        <span className="font-bold text-foreground">Mila</span>
                        <span className="truncate text-muted-foreground">@mila</span>
                        <span className="text-muted-foreground">·</span>
                        <span className="text-muted-foreground">2 min</span>
                      </div>
                      <p className="mt-1 text-sm leading-relaxed text-foreground/80">
                        {t('auth.login.demo.post1')}
                      </p>
                      <div className="mt-3 h-28 rounded-[18px] bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] p-px">
                        <div className="h-full rounded-[17px] bg-white/20 p-3">
                          <div className="h-full rounded-[14px] bg-white/25" />
                        </div>
                      </div>
                      <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
                        <span className="inline-flex items-center gap-1">
                          <MessageCircle className="h-4 w-4" />
                          124
                        </span>
                        <span className="inline-flex items-center gap-1 text-red-500">
                          <Heart className="h-4 w-4 fill-red-500" />
                          2.8K
                        </span>
                        <span>{t('auth.login.demo.views')}</span>
                      </div>
                    </div>
                  </div>
                </article>

                <article className="glass rounded-[22px] border p-4 shadow-sm">
                  <div className="flex items-center gap-3">
                    <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-slate-950 text-sm font-bold text-white dark:bg-white/15">
                      NO
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-bold text-foreground">
                        {t('auth.login.demo.joined')}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {t('auth.login.demo.joined_sub')}
                      </p>
                    </div>
                    <UserPlus className="h-5 w-5 text-[#8D3DFF]" />
                  </div>
                </article>
              </div>
            </div>

            <div className="absolute right-8 top-20 w-64 rounded-[28px] border border-white/40 bg-slate-950/90 p-4 text-white shadow-[0_28px_70px_rgba(15,23,42,0.32)] backdrop-blur-xl dark:border-white/10">
              <div className="flex items-center justify-between">
                <span className="text-sm font-semibold">{t('trends.title')}</span>
                <Sparkles className="h-4 w-4 text-[#47D9FF]" />
              </div>

              <div className="mt-4 space-y-3">
                {['#DesignSprint', '#CampusLife', '#DevDistribue'].map(
                  (trend, index) => (
                    <div key={trend} className="rounded-2xl bg-white/10 px-3 py-2">
                      <p className="text-xs text-white/50">
                        {t('auth.login.demo.rank', { n: index + 1 })}
                      </p>
                      <p className="text-sm font-semibold">{trend}</p>
                    </div>
                  )
                )}
              </div>
            </div>

            <div className="glass absolute bottom-0 right-24 flex w-72 items-center gap-3 rounded-[24px] border px-4 py-3 shadow-[0_24px_70px_rgba(141,61,255,0.22)] backdrop-blur-xl">
              <div className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-[#47D9FF]/20 text-[#5B6CFF]">
                <Bell className="h-5 w-5" />
              </div>
              <div>
                <p className="text-sm font-bold text-foreground">
                  {t('auth.login.demo.interactions')}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t('auth.login.demo.interactions_sub')}
                </p>
              </div>
            </div>
          </div>
        </div>

        <Card className="glass-strong relative w-full overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(0,0,0,0.22)] backdrop-blur-2xl sm:max-w-md sm:justify-self-center lg:max-w-none">
          <div className="pointer-events-none absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)]" />
          <div className="pointer-events-none absolute inset-x-8 top-1 h-24 bg-gradient-to-b from-white/70 to-transparent dark:hidden" />

          <CardHeader className="relative space-y-4 px-5 pb-2 pt-5 sm:px-8 sm:pt-7">
            <Link
              href={ROUTES.home}
              className="group inline-flex w-fit items-center transition duration-300 hover:scale-[1.03] lg:hidden"
            >
              <Image
                src="/logo_breezy.png"
                alt="Breezy"
                width={1106}
                height={336}
                className="h-9 w-auto object-contain drop-shadow-sm"
                priority
              />
            </Link>

            <div className="inline-flex w-fit items-center gap-2 rounded-full border border-white/70 bg-white/80 px-3 py-1 text-xs font-semibold text-[#5B6CFF] shadow-sm dark:border-white/15 dark:bg-white/10">
              <span className="h-2 w-2 rounded-full bg-[#47D9FF]" />
              {t('auth.login.badge')}
            </div>

            <div className="space-y-2">
              <CardTitle className="brand-text max-w-md text-[31px] font-semibold leading-tight sm:text-[38px]">
                {t('auth.login.title')}
              </CardTitle>

              <CardDescription className="max-w-sm text-sm leading-relaxed text-muted-foreground">
                {t('auth.login.subtitle')}
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="relative px-5 pb-5 sm:px-8 sm:pb-7">
            {mfaChallenge ? (
              <form className="space-y-4" onSubmit={handleMfaVerify} noValidate>
                <div className="space-y-1.5">
                  <h2 className="text-lg font-semibold text-foreground">
                    {t('auth.mfa.title')}
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    {t('auth.mfa.subtitle')}
                  </p>
                </div>

                <div className="space-y-1.5">
                  <label htmlFor="mfa-login-code" className="text-sm font-medium text-foreground/80">
                    {t('auth.mfa.code_label')}
                  </label>
                  <Input
                    id="mfa-login-code"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    placeholder="123456"
                    autoFocus
                    className="h-12 rounded-2xl border-white/70 bg-white/90 text-center text-lg tracking-[0.4em] shadow-sm transition-all focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
                    value={mfaCode}
                    onChange={(event) => {
                      setMfaCode(event.target.value)
                      if (mfaError) setMfaError(null)
                    }}
                  />
                </div>

                {mfaError ? (
                  <div className="flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50/90 px-4 py-2.5 text-sm text-red-700 shadow-sm dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
                    <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                    <p>{mfaError}</p>
                  </div>
                ) : null}

                <Button
                  type="submit"
                  className="h-12 w-full rounded-2xl bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-base font-semibold text-white shadow-[0_18px_44px_rgba(91,108,255,0.34)] transition duration-300 hover:scale-[1.015] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
                  disabled={mfaBusy || !mfaCode.trim()}
                >
                  {mfaBusy ? t('auth.mfa.verifying') : t('auth.mfa.verify')}
                </Button>

                <button
                  type="button"
                  onClick={() => {
                    setMfaChallenge(null)
                    setMfaCode('')
                    setMfaError(null)
                  }}
                  className="w-full text-center text-sm font-medium text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
                >
                  {t('auth.mfa.back')}
                </button>
              </form>
            ) : (
            <form className="space-y-3.5" onSubmit={handleSubmit} noValidate>
              <div className="space-y-1.5">
                <label htmlFor="identifier" className="text-sm font-medium text-foreground/80">
                  {t('auth.login.identifier_label')}
                </label>

                <div className="group relative">
                  <AtSign className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                  <Input
                    id="identifier"
                    name="identifier"
                    type="text"
                    autoComplete="username"
                    inputMode="text"
                    placeholder={t('auth.login.identifier_placeholder')}
                    className="h-12 rounded-2xl border-white/70 bg-white/90 pl-11 text-[15px] shadow-sm shadow-slate-200/60 transition-all placeholder:text-muted-foreground hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
                    value={identifier}
                    onChange={(event) => {
                      setIdentifier(event.target.value)
                      if (errors.identifier) {
                        setErrors((current) => ({ ...current, identifier: undefined }))
                      }
                    }}
                    aria-invalid={Boolean(errors.identifier)}
                    aria-describedby={errors.identifier ? 'identifier-error' : undefined}
                  />
                </div>

                {errors.identifier ? (
                  <p id="identifier-error" className="text-xs text-red-600">
                    {errors.identifier}
                  </p>
                ) : null}
              </div>

              <div className="space-y-1.5">
                <label htmlFor="password" className="text-sm font-medium text-foreground/80">
                  {t('auth.password_label')}
                </label>

                <div className="group relative">
                  <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                  <Input
                    id="password"
                    name="password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    placeholder="••••••••"
                    className="h-12 rounded-2xl border-white/70 bg-white/90 pl-11 pr-12 text-[15px] shadow-sm shadow-slate-200/60 transition-all placeholder:text-muted-foreground hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
                    value={password}
                    onChange={(event) => {
                      setPassword(event.target.value)
                      if (errors.password) {
                        setErrors((current) => ({
                          ...current,
                          password: undefined,
                        }))
                      }
                    }}
                    aria-invalid={Boolean(errors.password)}
                    aria-describedby={errors.password ? 'password-error' : undefined}
                  />

                  <button
                    type="button"
                    aria-label={
                      showPassword ? t('auth.hide_password') : t('auth.show_password')
                    }
                    aria-pressed={showPassword}
                    className="absolute right-3 top-1/2 grid h-8 w-8 -translate-y-1/2 place-items-center rounded-full text-muted-foreground transition hover:bg-[#5B6CFF]/10 hover:text-[#5B6CFF] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                    onClick={() => setShowPassword((current) => !current)}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </button>
                </div>

                {errors.password ? (
                  <p id="password-error" className="text-xs text-red-600">
                    {errors.password}
                  </p>
                ) : null}
              </div>

              <div className="flex items-center justify-between text-sm">
                <Link
                  href={ROUTES.forgotPassword}
                  className="font-medium text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
                >
                  {t('auth.login.forgot')}
                </Link>
              </div>

              {needsVerification ? (
                <div className="space-y-2 rounded-2xl border border-amber-300 bg-amber-50/90 px-4 py-3 text-sm text-amber-800 shadow-sm dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
                  <div className="flex items-start gap-3">
                    <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                    <p>{t('auth.verify.login_blocked')}</p>
                  </div>
                  {resendNotice ? (
                    <p className="pl-7 text-xs text-amber-700 dark:text-amber-300">
                      {resendNotice}
                    </p>
                  ) : (
                    <button
                      type="button"
                      onClick={handleResendVerification}
                      disabled={isResending}
                      className="ml-7 font-semibold underline underline-offset-4 transition hover:text-amber-900 disabled:cursor-not-allowed disabled:opacity-70 dark:hover:text-amber-100"
                    >
                      {isResending
                        ? t('auth.verify.resending')
                        : t('auth.verify.resend_cta')}
                    </button>
                  )}
                </div>
              ) : null}

              {errors.form ? (
                <div className="flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50/90 px-4 py-2.5 text-sm text-red-700 shadow-sm dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
                  <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                  <p>{errors.form}</p>
                </div>
              ) : null}

              <Button
                type="submit"
                className="h-12 w-full rounded-2xl bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-base font-semibold text-white shadow-[0_18px_44px_rgba(91,108,255,0.34)] transition duration-300 hover:scale-[1.015] hover:shadow-[0_24px_56px_rgba(91,108,255,0.42)] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
                disabled={isSubmitting}
              >
                {isSubmitting ? t('auth.login.submitting') : t('auth.login.submit')}
              </Button>

              <div className="space-y-3 pt-1">
                <div className="flex items-center gap-3">
                  <div className="h-px flex-1 bg-gradient-to-r from-transparent via-border to-transparent" />

                  <span className="text-xs font-medium text-muted-foreground">
                    {t('auth.login.or')}
                  </span>

                  <div className="h-px flex-1 bg-gradient-to-r from-transparent via-border to-transparent" />
                </div>

                <div className="grid grid-cols-2 gap-2.5">
                  {OAUTH_PROVIDERS.map((provider) => (
                    <button
                      key={provider.id}
                      type="button"
                      aria-label={t('auth.oauth.continue_with', { provider: provider.label })}
                      title={provider.label}
                      disabled={oauthLoading !== null || isSubmitting}
                      onClick={() => handleOAuth(provider.id)}
                      className="flex h-11 w-full items-center justify-center gap-2 rounded-2xl border border-gray-300 bg-white text-sm font-medium text-gray-700 transition hover:scale-[1.03] hover:bg-white hover:shadow-md disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {oauthLoading === provider.id ? (
                        <Loader2 className="h-4 w-4 animate-spin text-gray-500" />
                      ) : (
                        <Image src={provider.src} alt={provider.label} width={20} height={20} />
                      )}
                      <span>{provider.label}</span>
                    </button>
                  ))}
                </div>
              </div>

              <p className="text-center text-sm text-muted-foreground">
                {t('auth.login.no_account')}{' '}
                <Link
                  href={ROUTES.register}
                  className="font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
                >
                  {t('auth.login.create_account')}
                </Link>
              </p>

              <LegalLinks className="items-center text-center" />
            </form>
            )}
          </CardContent>
        </Card>
      </section>
    </main>
  )
}
