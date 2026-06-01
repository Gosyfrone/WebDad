import Link from 'next/link'

import { ROUTES } from '@/lib/routes'
import { Button } from '@/components/ui/button'

export default function NotFound() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
      <h1 className="text-4xl font-bold">404</h1>
      <p className="text-muted-foreground">Cette page n&apos;existe pas.</p>
      <Button asChild>
        <Link href={ROUTES.home}>Retour à l&apos;accueil</Link>
      </Button>
    </main>
  )
}
