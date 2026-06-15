'use client'

import * as React from 'react'
import Link from 'next/link'
import { useRouter, useSearchParams } from 'next/navigation'
import { CircleCheck, CircleX, Loader2 } from 'lucide-react'

import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { setAccessToken } from '@/lib/auth-client'
import { ROUTES } from '@/lib/routes'
import { notifySessionChanged } from '@/lib/session'

type Status = 'loading' | 'success' | 'invalid'

function Content() {
  const t = useT()
  const router = useRouter()
  const token = useSearchParams().get('token') ?? ''
  const [status, setStatus] = React.useState<Status>('loading')
  const once = React.useRef(false)

  React.useEffect(() => {
    if (!token) {
      setStatus('invalid')
      return
    }
    if (once.current) return
    once.current = true
    void fetch('/api/auth/email/change/confirm', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token }),
    }).then(async (response) => {
      const payload = await response.json().catch(() => null)
      if (!response.ok) {
        setStatus('invalid')
        return
      }
      if (payload?.accessToken) setAccessToken(payload.accessToken)
      // La nouvelle adresse vit désormais dans le JWT : on resynchronise la
      // session puis on ramène l'utilisateur sur l'écran de changement d'e-mail,
      // où un bandeau « Adresse e-mail modifiée » s'affichera (cf. ?email_changed).
      notifySessionChanged()
      setStatus('success')
      router.replace(`${ROUTES.parametres}?email_changed=1`)
    }).catch(() => setStatus('invalid'))
  }, [token, router])

  return (
    <Card className="glass-strong w-full max-w-md rounded-[30px] border text-center shadow-xl">
      <CardHeader className="space-y-4 pt-8">
        <div className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-primary/10 text-primary">
          {status === 'loading' ? <Loader2 className="h-7 w-7 animate-spin" /> : status === 'success' ? <CircleCheck className="h-7 w-7" /> : <CircleX className="h-7 w-7 text-destructive" />}
        </div>
        <CardTitle>{t(`settings.email.confirm_${status}_title`)}</CardTitle>
        <CardDescription>{t(`settings.email.confirm_${status}_desc`)}</CardDescription>
      </CardHeader>
      <CardContent className="pb-8">
        {status !== 'loading' ? (
          <Button asChild className="w-full">
            <Link href={status === 'success' ? ROUTES.parametres : ROUTES.login}>
              {status === 'success' ? t('settings.email.confirm_back') : t('auth.check_email.back_to_login')}
            </Link>
          </Button>
        ) : null}
      </CardContent>
    </Card>
  )
}

export default function VerifyEmailChangePage() {
  return (
    <main className="bg-page flex min-h-dvh items-center justify-center px-4 py-8">
      <React.Suspense fallback={null}><Content /></React.Suspense>
    </main>
  )
}
