import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { markLoginActivity, provisionUser } from '@/lib/provision'
import { setRefreshCookie } from '@/lib/server/auth-cookie'

type AuthPayload = {
  data?: { token?: string; refresh_token?: string; user?: unknown }
  error?: string
  code?: string
}

/**
 * POST /api/auth/mfa/verify — second facteur du login MFA. Le client envoie le
 * `challenge` (reçu de /api/auth/login) + le `code` TOTP. Le BFF relaie au back,
 * et — comme le login normal — pose le cookie refresh httpOnly, renvoie l'access
 * token au client (→ localStorage) et provisionne l'identité (best-effort).
 */
export async function POST(request: NextRequest) {
  let body: { challenge?: string; code?: string }
  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 }
    )
  }

  if (!body.challenge || !body.code) {
    return NextResponse.json(
      { error: 'Le challenge et le code sont requis.' },
      { status: 400 }
    )
  }

  let upstream: Response
  try {
    upstream = await fetch(apiUrl('/auth/mfa/verify'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ challenge: body.challenge, code: body.code }),
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
      { error: payload?.error ?? 'Vérification impossible.', code: payload?.code },
      { status: upstream.status }
    )
  }

  const accessToken = payload?.data?.token ?? null
  const refreshToken = payload?.data?.refresh_token ?? null

  const res = NextResponse.json(
    { accessToken, user: payload?.data?.user },
    { status: 200 }
  )

  if (refreshToken) {
    setRefreshCookie(res, refreshToken)
  }
  if (accessToken) {
    void provisionUser(accessToken)
    void markLoginActivity(accessToken)
  }

  return res
}
