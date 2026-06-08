'use client'

import Link from 'next/link'

import { ROUTES } from '@/lib/routes'
import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'

export default function NotFound() {
  const t = useT()
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
      <h1 className="text-4xl font-bold">404</h1>
      <p className="text-muted-foreground">{t('notfound.message')}</p>
      <Button asChild>
        <Link href={ROUTES.home}>{t('notfound.back')}</Link>
      </Button>
    </main>
  )
}
