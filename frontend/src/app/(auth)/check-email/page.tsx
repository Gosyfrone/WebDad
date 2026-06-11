'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { MailCheck } from 'lucide-react'
import * as React from 'react'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ROUTES } from '@/lib/routes'
import { useT } from '@/components/language-provider'

function CheckEmailContent() {
  const t = useT()
  const searchParams = useSearchParams()
  const email = searchParams.get('email') ?? ''
  const [isResending, setIsResending] = React.useState(false)
  const [notice, setNotice] = React.useState<string | null>(null)

  const handleResend = async () => {
    if (!email) return
    setIsResending(true)
    setNotice(null)
    try {
      await fetch('/api/auth/verify-email/resend', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      })
      setNotice(t('auth.verify.resend_done'))
    } catch {
      setNotice(t('auth.verify.resend_done'))
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

        <div className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-[#5B6CFF]/15 text-[#5B6CFF]">
          <MailCheck className="h-7 w-7" />
        </div>

        <div className="space-y-2">
          <CardTitle className="brand-text text-[26px] font-semibold leading-tight">
            {t('auth.check_email.title')}
          </CardTitle>
          <CardDescription className="text-sm leading-relaxed text-muted-foreground">
            {email
              ? t('auth.check_email.subtitle', { email })
              : t('auth.check_email.subtitle_generic')}
          </CardDescription>
        </div>
      </CardHeader>

      <CardContent className="relative space-y-4 px-6 pb-7 pt-2 text-center">
        <p className="text-xs leading-relaxed text-muted-foreground">
          {t('auth.check_email.hint')}
        </p>

        {notice ? (
          <p className="rounded-2xl border border-[#5B6CFF]/30 bg-[#5B6CFF]/10 px-4 py-2.5 text-sm text-[#5B6CFF]">
            {notice}
          </p>
        ) : email ? (
          <Button
            type="button"
            variant="outline"
            onClick={handleResend}
            disabled={isResending}
            className="h-11 w-full rounded-2xl"
          >
            {isResending
              ? t('auth.verify.resending')
              : t('auth.check_email.resend')}
          </Button>
        ) : null}

        <Link
          href={ROUTES.login}
          className="block text-sm font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
        >
          {t('auth.check_email.back_to_login')}
        </Link>
      </CardContent>
    </Card>
  )
}

export default function CheckEmailPage() {
  return (
    <main className="bg-page relative flex min-h-dvh w-full items-center justify-center overflow-hidden px-4 py-8">
      <div className="bg-page-glow-1 pointer-events-none absolute inset-0" />
      <div className="bg-page-glow-2 pointer-events-none absolute inset-0" />
      <section className="relative w-full max-w-md">
        <React.Suspense fallback={null}>
          <CheckEmailContent />
        </React.Suspense>
      </section>
    </main>
  )
}
