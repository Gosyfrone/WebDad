import { NextRequest, NextResponse } from 'next/server'

import { REFRESH_COOKIE } from '@/lib/server/auth-cookie'

/**
 * Garde de session (minimale) : redirige vers /login si le cookie httpOnly
 * refresh est absent sur une route de l'espace authentifié (route group (app)).
 *
 * On ne VALIDE pas le token ici (présence suffit comme garde grossière) : la
 * vraie validation se fait à chaque appel API côté client (401 → refresh →
 * /login si le refresh échoue). Le `matcher` ci-dessous borne la portée aux
 * routes protégées (les assets, /api et les pages publiques en sont exclus).
 */
export function middleware(request: NextRequest) {
  const hasSession = Boolean(request.cookies.get(REFRESH_COOKIE)?.value)
  if (hasSession) {
    return NextResponse.next()
  }

  const loginUrl = new URL('/login', request.url)
  return NextResponse.redirect(loginUrl)
}

export const config = {
  matcher: [
    '/feed/:path*',
    '/explorer/:path*',
    '/notifications/:path*',
    '/posts/:path*',
    '/messages/:path*',
    '/profil/:path*',
    '/parametres/:path*',
    '/moderation/:path*',
    '/admin/:path*',
  ],
}
