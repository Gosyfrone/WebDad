'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { CircleAlert, CircleCheck, CircleX, Eye, EyeOff, Lock } from 'lucide-react'
import * as React from 'react'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { ROUTES } from '@/lib/routes'
import { useT } from '@/components/language-provider'

type Status = 'form' | 'success' | 'invalid'

function ResetPasswordContent() {
  const t = useT()
  const searchParams = useSearchParams()
  const token = searchParams.get('token') ?? ''

  // Token absent dès le départ → écran « lien invalide » sans appel réseau.
  const [status, setStatus] = React.useState<Status>(token ? 'form' : 'invalid')
  const [password, setPassword] = React.useState('')
  const [confirm, setConfirm] = React.useState('')
  const [showPassword, setShowPassword] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (password.length < 8) {
      setError(t('auth.err.password_min'))
      return
    }
    if (password !== confirm) {
      setError(t('auth.reset.err.mismatch'))
      return
    }

    setIsSubmitting(true)
    setError(null)
    try {
      const response = await fetch('/api/auth/password/reset', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, new_password: password }),
      })

      if (response.ok) {
        setStatus('success')
        return
      }

      const payload = await response.json().catch(() => null)
      // Token consommé / expiré / inconnu → écran « lien invalide ».
      if (payload?.code === 'invalid_token') {
        setStatus('invalid')
        return
      }
      setError(payload?.error ?? t('auth.reset.err.generic'))
    } catch {
      setError(t('auth.err.network'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="glass-strong relative w-full max-w-md overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(0,0,0,0.22)] backdrop-blur-2xl">
      <div className="pointer-events-none absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF]" />

      <CardHeader className="relative space-y-3 px-6 pb-2 pt-7 text-center">
        <Link
          href={ROUTES.home}
          className="group mx-auto inline-flex w-fit items-center transition duration-300 hover:scale-[1.03]"
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

        {status === 'success' ? (
          <div className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
            <CircleCheck className="h-7 w-7" />
          </div>
        ) : status === 'invalid' ? (
          <div className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-red-500/15 text-red-600 dark:text-red-400">
            <CircleX className="h-7 w-7" />
          </div>
        ) : (
          <div className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-[#5B6CFF]/15 text-[#5B6CFF]">
            <Lock className="h-7 w-7" />
          </div>
        )}

        <div className="space-y-2">
          <CardTitle className="brand-text text-[26px] font-semibold leading-tight">
            {status === 'success'
              ? t('auth.reset.success_title')
              : status === 'invalid'
                ? t('auth.reset.invalid_title')
                : t('auth.reset.title')}
          </CardTitle>
          <CardDescription className="text-sm leading-relaxed text-muted-foreground">
            {status === 'success'
              ? t('auth.reset.success_desc')
              : status === 'invalid'
                ? t('auth.reset.invalid_desc')
                : t('auth.reset.subtitle')}
          </CardDescription>
        </div>
      </CardHeader>

      <CardContent className="relative space-y-4 px-6 pb-7 pt-2">
        {status === 'success' ? (
          <Button
            asChild
            className="h-11 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-semibold text-white"
          >
            <Link href={ROUTES.login}>{t('auth.reset.go_to_login')}</Link>
          </Button>
        ) : status === 'invalid' ? (
          <Button
            asChild
            className="h-11 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-semibold text-white"
          >
            <Link href={ROUTES.forgotPassword}>{t('auth.reset.request_new')}</Link>
          </Button>
        ) : (
          <form className="space-y-4" onSubmit={handleSubmit} noValidate>
            <div className="space-y-2">
              <label
                htmlFor="reset-password"
                className="text-sm font-medium text-foreground/80"
              >
                {t('auth.reset.new_password_label')}
              </label>
              <div className="group relative">
                <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />
                <Input
                  id="reset-password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="new-password"
                  placeholder="••••••••"
                  className="h-11 rounded-2xl pl-11 pr-12"
                  value={password}
                  onChange={(event) => {
                    setPassword(event.target.value)
                    if (error) setError(null)
                  }}
                />
                <button
                  type="button"
                  aria-label={
                    showPassword ? t('auth.hide_password') : t('auth.show_password')
                  }
                  aria-pressed={showPassword}
                  className="absolute right-3 top-1/2 grid h-8 w-8 -translate-y-1/2 place-items-center rounded-full text-muted-foreground transition hover:bg-[#5B6CFF]/10 hover:text-[#5B6CFF]"
                  onClick={() => setShowPassword((current) => !current)}
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
            </div>

            <div className="space-y-2">
              <label
                htmlFor="reset-confirm"
                className="text-sm font-medium text-foreground/80"
              >
                {t('auth.reset.confirm_password_label')}
              </label>
              <div className="group relative">
                <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />
                <Input
                  id="reset-confirm"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="new-password"
                  placeholder="••••••••"
                  className="h-11 rounded-2xl pl-11"
                  value={confirm}
                  onChange={(event) => {
                    setConfirm(event.target.value)
                    if (error) setError(null)
                  }}
                />
              </div>
            </div>

            {error ? (
              <p className="flex items-center gap-1.5 text-xs text-red-600">
                <CircleAlert className="h-3.5 w-3.5" />
                {error}
              </p>
            ) : null}

            <Button
              type="submit"
              disabled={isSubmitting}
              className="h-11 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-semibold text-white disabled:cursor-not-allowed disabled:opacity-70"
            >
              {isSubmitting ? t('auth.reset.submitting') : t('auth.reset.submit')}
            </Button>
          </form>
        )}

        {status !== 'form' ? (
          <Link
            href={ROUTES.login}
            className="block text-center text-sm font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
          >
            {t('auth.check_email.back_to_login')}
          </Link>
        ) : null}
      </CardContent>
    </Card>
  )
}

export default function ResetPasswordPage() {
  return (
    <main className="bg-page relative flex min-h-dvh w-full items-center justify-center overflow-hidden px-4 py-8">
      <div className="bg-page-glow-1 pointer-events-none absolute inset-0" />
      <div className="bg-page-glow-2 pointer-events-none absolute inset-0" />
      <section className="relative w-full max-w-md">
        <React.Suspense fallback={null}>
          <ResetPasswordContent />
        </React.Suspense>
      </section>
    </main>
  )
}
