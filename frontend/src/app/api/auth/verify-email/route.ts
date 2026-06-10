import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'

/**
 * POST /api/auth/verify-email — passthrough vers /auth/verify-email/confirm.
 * Consomme le token reçu par e-mail. Aucun cookie ni session à gérer : le
 * compte reste à connecter ensuite. Relaie tel quel le statut + le corps
 * (succès {data:{message}} ou {error, code:'invalid_token'}).
 */
export async function POST(request: NextRequest) {
  let body: { token?: string }

  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 }
    )
  }

  if (!body.token) {
    return NextResponse.json({ error: 'Token manquant.' }, { status: 400 })
  }

  let upstreamResponse: Response
  try {
    upstreamResponse = await fetch(apiUrl('/auth/verify-email/confirm'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: body.token }),
    })
  } catch {
    return NextResponse.json(
      {
        error:
          'Impossible de joindre l’API Gateway. Vérifie que la stack est démarrée.',
      },
      { status: 502 }
    )
  }

  const payload = await upstreamResponse.json().catch(() => null)
  return NextResponse.json(payload ?? {}, { status: upstreamResponse.status })
}
