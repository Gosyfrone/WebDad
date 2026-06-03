import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'

/**
 * Vérifie la disponibilité d'un nom d'utilisateur AVANT l'inscription.
 *
 * Réutilise l'endpoint public `GET /users/by-username/:username` du
 * user-service (via la gateway) : `404` = libre, `200` = déjà pris. On ne
 * renvoie au client qu'un booléen (pas les données du user).
 *
 * Appelé en same-origin par la page register (le client ne joint pas la
 * gateway directement ; cf. cookie httpOnly + réseau Docker).
 */
export async function GET(request: NextRequest) {
  const username = request.nextUrl.searchParams.get('username')?.trim()

  if (!username) {
    return NextResponse.json({ error: 'Le nom d’utilisateur est requis.' }, { status: 400 })
  }

  let upstream: Response
  try {
    upstream = await fetch(apiUrl(`/users/by-username/${encodeURIComponent(username)}`), {
      cache: 'no-store',
    })
  } catch {
    return NextResponse.json(
      { error: 'Impossible de joindre l’API Gateway.' },
      { status: 502 }
    )
  }

  if (upstream.status === 404) {
    return NextResponse.json({ available: true })
  }
  if (upstream.status === 200) {
    return NextResponse.json({ available: false })
  }

  // Statut inattendu : on ne tranche pas (le POST /users restera autoritatif).
  return NextResponse.json(
    { error: 'Vérification du nom d’utilisateur indisponible.' },
    { status: 502 }
  )
}
