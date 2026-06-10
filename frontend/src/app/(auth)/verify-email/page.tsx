'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { CircleCheck, CircleX, Loader2, Mail } from 'lucide-react'
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

type Status = 'loading' | 'success' | 'invalid'

function VerifyEmailContent() {
  const t = useT()
  const searchParams = useSearchParams()
  const token = searchParams.get('token') ?? ''

  const [status, setStatus] = React.useState<Status>('loading')
  const [email, setEmail] = React.useState('')
  const [isResending, setIsResending] = React.useState(false)
  const [resendNotice, setResendNotice] = React.useState<string | null>(null)

  React.useEffect(() => {
    if (!token) {
      setStatus('invalid')
      return
    }

    let cancelled = false
    ;(async () => {
      try {
        const response = await fetch('/api/auth/verify-email', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token }),
        })
        if (cancelled) return
        setStatus(response.ok ? 'success' : 'invalid')
      } catch {
        if (!cancelled) setStatus('invalid')
      }
    })()

    return () => {
      cancelled = true
    }
  }, [token])

  const handleResend = async () => {
    const trimmedEmail = email.trim()
    if (!trimmedEmail) return
    setIsResending(true)
    setResendNotice(null)
    try {
      await fetch('/api/auth/verify-email/resend', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: trimmedEmail }),
      })
      setResendNotice(t('auth.verify.resend_done'))
    } catch {
      setResendNotice(t('auth.verify.resend_done'))
    } finally {
      setIsResending(false)
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

        {status === 'loading' ? (
          <div className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-[#5B6CFF]/15 text-[#5B6CFF]">
            <Loader2 className="h-7 w-7 animate-spin" />
          </div>
        ) : status === 'success' ? (
          <div className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
            <CircleCheck className="h-7 w-7" />
          </div>
        ) : (
          <div className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-red-500/15 text-red-600 dark:text-red-400">
            <CircleX className="h-7 w-7" />
          </div>
        )}

        <div className="space-y-2">
          <CardTitle className="brand-text text-[26px] font-semibold leading-tight">
            {status === 'loading'
              ? t('auth.verify.loading_title')
              : status === 'success'
                ? t('auth.verify.success_title')
                : t('auth.verify.invalid_title')}
          </CardTitle>
          <CardDescription className="text-sm leading-relaxed text-muted-foreground">
            {status === 'loading'
              ? t('auth.verify.loading_desc')
              : status === 'success'
                ? t('auth.verify.success_desc')
                : t('auth.verify.invalid_desc')}
          </CardDescription>
        </div>
      </CardHeader>

      <CardContent className="relative space-y-4 px-6 pb-7 pt-2">
        {status === 'success' ? (
          <Button asChild className="h-11 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-semibold text-white">
            <Link href={ROUTES.login}>{t('auth.verify.go_to_login')}</Link>
          </Button>
        ) : null}

        {status === 'invalid' ? (
          resendNotice ? (
            <p className="rounded-2xl border border-[#5B6CFF]/30 bg-[#5B6CFF]/10 px-4 py-2.5 text-center text-sm text-[#5B6CFF]">
              {resendNotice}
            </p>
          ) : (
            <div className="space-y-2">
              <label htmlFor="resend-email" className="text-sm font-medium text-foreground/80">
                {t('auth.verify.resend_label')}
              </label>
              <div className="group relative">
                <Mail className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />
                <Input
                  id="resend-email"
                  type="email"
                  inputMode="email"
                  autoComplete="email"
                  placeholder={t('auth.email_placeholder')}
                  className="h-11 rounded-2xl pl-11"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                />
              </div>
              <Button
                type="button"
                onClick={handleResend}
                disabled={isResending || !email.trim()}
                className="h-11 w-full rounded-2xl"
              >
                {isResending
                  ? t('auth.verify.resending')
                  : t('auth.verify.resend_cta')}
              </Button>
            </div>
          )
        ) : null}

        {status !== 'loading' ? (
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

export default function VerifyEmailPage() {
  return (
    <main className="bg-page relative flex min-h-dvh w-full items-center justify-center overflow-hidden px-4 py-8">
      <div className="bg-page-glow-1 pointer-events-none absolute inset-0" />
      <div className="bg-page-glow-2 pointer-events-none absolute inset-0" />
      <section className="relative w-full max-w-md">
        <React.Suspense fallback={null}>
          <VerifyEmailContent />
        </React.Suspense>
      </section>
    </main>
  )
}
