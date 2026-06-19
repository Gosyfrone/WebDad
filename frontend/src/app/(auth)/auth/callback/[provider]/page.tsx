'use client'

import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { CircleAlert, Loader2 } from 'lucide-react'
import * as React from 'react'

import { useT } from '@/components/language-provider'
import { setAccessToken } from '@/lib/auth-client'
import { ROUTES } from '@/lib/routes'
import { savePendingOAuthSignup } from '@/lib/oauth-pending'

function CallbackContent({ provider }: { provider: string }) {
  const t = useT()
  const searchParams = useSearchParams()
  const [error, setError] = React.useState<string | null>(null)
  const exchangeStarted = React.useRef(false)

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
          savePendingOAuthSignup({
            provider,
            pendingToken: payload.pendingToken,
            email: payload.email ?? '',
          })
          window.location.replace('/auth/oauth/terms')
          return
        }

        if (payload?.accessToken) {
          setAccessToken(payload.accessToken)
        }

        window.location.assign(ROUTES.feed)
      } catch {
        setError(t('auth.oauth.error'))
      }
    }

    exchange()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (error) {
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
