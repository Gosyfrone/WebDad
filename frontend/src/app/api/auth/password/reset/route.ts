import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'

/**
 * POST /api/auth/password/reset — passthrough vers /auth/password/reset.
 * Consomme le token reçu par e-mail et remplace le mot de passe. Aucun cookie
 * ni session à gérer : le back révoque les sessions et l'utilisateur se
 * reconnecte ensuite. Relaie tel quel le statut + le corps (succès
 * {data:{message}} ou {error, code:'invalid_token'}).
 */
export async function POST(request: NextRequest) {
  let body: { token?: string; new_password?: string }

  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 }
    )
  }

  if (!body.token || !body.new_password) {
    return NextResponse.json(
      { error: 'Token ou mot de passe manquant.' },
      { status: 400 }
    )
  }

  let upstreamResponse: Response
  try {
    upstreamResponse = await fetch(apiUrl('/auth/password/reset'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: body.token, new_password: body.new_password }),
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
