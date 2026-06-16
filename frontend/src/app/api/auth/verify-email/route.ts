import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { markLoginActivity, provisionUser } from '@/lib/provision'
import { setRefreshCookie } from '@/lib/server/auth-cookie'

type AuthPayload = {
  data?: {
    message?: string
    token?: string
    refresh_token?: string
    user?: unknown
  }
  error?: string
  code?: string
}

/**
 * POST /api/auth/verify-email — confirme le token reçu par e-mail PUIS ouvre la
 * session : pose le cookie refresh httpOnly et renvoie l'access token au client
 * (→ localStorage). Cliquer le lien prouve la possession de la boîte, donc
 * l'utilisateur entre directement dans l'app sans se reconnecter. Symétrique du
 * BFF /api/auth/login.
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

  const payload = (await upstreamResponse.json().catch(() => null)) as AuthPayload | null

  if (!upstreamResponse.ok) {
    const message = payload?.error ?? 'La vérification a échoué.'
    return NextResponse.json(
      { error: message, code: payload?.code },
      { status: upstreamResponse.status }
    )
  }

  const accessToken = payload?.data?.token ?? null
  const refreshToken = payload?.data?.refresh_token ?? null

  // L'access token repart au client (→ localStorage) ; le refresh token reste
  // dans un cookie httpOnly posé ici (jamais exposé au JS).
  const nextResponse = NextResponse.json(
    { accessToken, user: payload?.data?.user, message: 'Adresse vérifiée.' },
    { status: upstreamResponse.status }
  )

  if (refreshToken) {
    setRefreshCookie(nextResponse, refreshToken)
  }

  // Provisioning paresseux best-effort (idempotent) : l'identité a déjà été
  // créée au register, on s'aligne sur le flux login par robustesse.
  if (accessToken) {
    void provisionUser(accessToken)
    void markLoginActivity(accessToken)
  }

  return nextResponse
}
