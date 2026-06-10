import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'

/**
 * POST /api/auth/password/forgot — passthrough vers /auth/password/forgot.
 * Déclenche l'envoi du mail de réinitialisation. Réponse TOUJOURS générique
 * côté back (anti-énumération) : on relaie le statut + le corps tels quels.
 * Aucun cookie ni session à gérer.
 */
export async function POST(request: NextRequest) {
  let body: { email?: string }

  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 }
    )
  }

  if (!body.email) {
    return NextResponse.json({ error: 'Adresse e-mail manquante.' }, { status: 400 })
  }

  let upstreamResponse: Response
  try {
    upstreamResponse = await fetch(apiUrl('/auth/password/forgot'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: body.email }),
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
