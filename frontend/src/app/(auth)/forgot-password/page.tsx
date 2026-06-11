'use client'

import Image from 'next/image'
import Link from 'next/link'
import { CircleAlert, Mail, MailCheck } from 'lucide-react'
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

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export default function ForgotPasswordPage() {
  const t = useT()
  const [email, setEmail] = React.useState('')
  const [error, setError] = React.useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  // Écran de confirmation générique (anti-énumération) une fois l'envoi tenté.
  const [sent, setSent] = React.useState(false)

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const trimmedEmail = email.trim()

    if (!trimmedEmail) {
      setError(t('auth.err.email_required'))
      return
    }
    if (!emailPattern.test(trimmedEmail)) {
      setError(t('auth.err.email_invalid'))
      return
    }

    setIsSubmitting(true)
    setError(null)
    try {
      await fetch('/api/auth/password/forgot', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: trimmedEmail }),
      })
      // Réponse générique (anti-énumération) : même écran que l'adresse existe
      // ou non, succès comme échec réseau.
      setSent(true)
    } catch {
      setSent(true)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="bg-page relative flex min-h-dvh w-full items-center justify-center overflow-hidden px-4 py-8">
      <div className="bg-page-glow-1 pointer-events-none absolute inset-0" />
      <div className="bg-page-glow-2 pointer-events-none absolute inset-0" />
      <section className="relative w-full max-w-md">
        <Card className="glass-strong relative w-full overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(0,0,0,0.22)] backdrop-blur-2xl">
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

            <div
              className={`mx-auto grid h-14 w-14 place-items-center rounded-full ${
                sent
                  ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                  : 'bg-[#5B6CFF]/15 text-[#5B6CFF]'
              }`}
            >
              {sent ? <MailCheck className="h-7 w-7" /> : <Mail className="h-7 w-7" />}
            </div>

            <div className="space-y-2">
              <CardTitle className="brand-text text-[26px] font-semibold leading-tight">
                {sent ? t('auth.forgot.sent_title') : t('auth.forgot.title')}
              </CardTitle>
              <CardDescription className="text-sm leading-relaxed text-muted-foreground">
                {sent ? t('auth.forgot.sent_desc') : t('auth.forgot.subtitle')}
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="relative space-y-4 px-6 pb-7 pt-2">
            {sent ? (
              <Link
                href={ROUTES.login}
                className="block text-center text-sm font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
              >
                {t('auth.check_email.back_to_login')}
              </Link>
            ) : (
              <form className="space-y-4" onSubmit={handleSubmit} noValidate>
                <div className="space-y-2">
                  <label
                    htmlFor="forgot-email"
                    className="text-sm font-medium text-foreground/80"
                  >
                    {t('auth.email_label')}
                  </label>
                  <div className="group relative">
                    <Mail className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />
                    <Input
                      id="forgot-email"
                      type="email"
                      inputMode="email"
                      autoComplete="email"
                      placeholder={t('auth.email_placeholder')}
                      className="h-11 rounded-2xl pl-11"
                      value={email}
                      onChange={(event) => {
                        setEmail(event.target.value)
                        if (error) setError(null)
                      }}
                      aria-invalid={Boolean(error)}
                      aria-describedby={error ? 'forgot-email-error' : undefined}
                    />
                  </div>
                  {error ? (
                    <p
                      id="forgot-email-error"
                      className="flex items-center gap-1.5 text-xs text-red-600"
                    >
                      <CircleAlert className="h-3.5 w-3.5" />
                      {error}
                    </p>
                  ) : null}
                </div>

                <Button
                  type="submit"
                  disabled={isSubmitting}
                  className="h-11 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-semibold text-white disabled:cursor-not-allowed disabled:opacity-70"
                >
                  {isSubmitting
                    ? t('auth.forgot.submitting')
                    : t('auth.forgot.submit')}
                </Button>

                <Link
                  href={ROUTES.login}
                  className="block text-center text-sm font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
                >
                  {t('auth.check_email.back_to_login')}
                </Link>
              </form>
            )}
          </CardContent>
        </Card>
      </section>
    </main>
  )
}
