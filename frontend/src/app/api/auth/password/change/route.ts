import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { setRefreshCookie } from '@/lib/server/auth-cookie'

type AuthPayload = {
  data?: { token?: string; refresh_token?: string }
  error?: string
  code?: string
}

/**
 * POST /api/auth/password/change — change le mot de passe de l'utilisateur
 * authentifié (changement volontaire OU imposé après création par un admin).
 *
 * Le client envoie son access token (en-tête Authorization) + les mots de passe.
 * Le BFF relaie au back, récupère la NOUVELLE paire (le back a révoqué les
 * anciennes sessions), pose le nouveau cookie refresh httpOnly et renvoie le
 * nouvel access token (→ localStorage). Le nouveau JWT ne porte plus le drapeau
 * must_change_password : la modale bloquante disparaît sans rechargement.
 */
export async function POST(request: NextRequest) {
  const authorization = request.headers.get('authorization')
  if (!authorization) {
    return NextResponse.json({ error: 'Non authentifié.' }, { status: 401 })
  }

  let body: { current_password?: string; new_password?: string }
  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 }
    )
  }

  if (!body.current_password || !body.new_password) {
    return NextResponse.json(
      { error: 'Le mot de passe actuel et le nouveau mot de passe sont requis.' },
      { status: 400 }
    )
  }

  let upstream: Response
  try {
    upstream = await fetch(apiUrl('/auth/password/change'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: authorization },
      body: JSON.stringify({
        current_password: body.current_password,
        new_password: body.new_password,
      }),
    })
  } catch {
    return NextResponse.json(
      { error: 'Impossible de joindre l’API Gateway.' },
      { status: 502 }
    )
  }

  const payload = (await upstream.json().catch(() => null)) as AuthPayload | null

  if (!upstream.ok) {
    return NextResponse.json(
      {
        error: payload?.error ?? 'Changement de mot de passe impossible.',
        code: payload?.code,
      },
      { status: upstream.status }
    )
  }

  const accessToken = payload?.data?.token ?? null
  const newRefresh = payload?.data?.refresh_token ?? null

  const res = NextResponse.json({ accessToken }, { status: 200 })
  if (newRefresh) {
    setRefreshCookie(res, newRefresh)
  }
  return res
}
