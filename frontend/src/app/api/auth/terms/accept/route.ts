import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { setRefreshCookie } from '@/lib/server/auth-cookie'

type AuthPayload = {
  data?: { token?: string; refresh_token?: string }
  error?: string
}

/**
 * POST /api/auth/terms/accept — enregistre l'acceptation par l'utilisateur
 * authentifié de la version EN VIGUEUR des CGU.
 *
 * Le client envoie son access token (en-tête Authorization). Le BFF relaie au
 * back, récupère la NOUVELLE paire, pose le nouveau cookie refresh httpOnly et
 * renvoie le nouvel access token (→ localStorage). Le nouveau JWT porte
 * terms_accepted=true : la modale d'acceptation bloquante disparaît sans
 * rechargement.
 */
export async function POST(request: NextRequest) {
  const authorization = request.headers.get('authorization')
  if (!authorization) {
    return NextResponse.json({ error: 'Non authentifié.' }, { status: 401 })
  }

  let upstream: Response
  try {
    upstream = await fetch(apiUrl('/auth/terms/accept'), {
      method: 'POST',
      headers: { Authorization: authorization },
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
      { error: payload?.error ?? 'Acceptation des CGU impossible.' },
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
