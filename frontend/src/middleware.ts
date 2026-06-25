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
 *
 * Mode visiteur : `/feed` et `/posts/:id` sont VOLONTAIREMENT hors matcher → un
 * utilisateur non connecté peut consulter le fil public et le détail d'un post
 * en lecture seule (le backend filtre déjà la visibilité côté serveur). Les
 * actions réservées sont gardées côté UI (cf. AuthPromptProvider).
 */
export function middleware(request: NextRequest) {
  const hasSession = Boolean(request.cookies.get(REFRESH_COOKIE)?.value)
  if (hasSession) {
    return NextResponse.next()
  }

  // Mémorise la cible (chemin + query) pour y revenir après connexion : un lien
  // d'invitation `/messages?join=<id>` ouvert par un visiteur le ramène ici une
  // fois connecté (cf. page /login). `searchParams.set` encode la valeur.
  const loginUrl = new URL('/login', request.url)
  const { pathname, search } = request.nextUrl
  loginUrl.searchParams.set('next', pathname + search)
  return NextResponse.redirect(loginUrl)
}

export const config = {
  matcher: [
    '/explorer/:path*',
    '/notifications/:path*',
    '/messages/:path*',
    '/signets/:path*',
    '/profil/:path*',
    '/parametres/:path*',
    '/moderation/:path*',
    '/admin/:path*',
  ],
}
