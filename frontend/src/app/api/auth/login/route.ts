import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { provisionUser } from '@/lib/provision'
import { setRefreshCookie } from '@/lib/server/auth-cookie'

type AuthPayload = {
  data?: {
    token?: string
    refresh_token?: string
    user?: unknown
  }
  message?: string
  error?: string
}

export async function POST(request: NextRequest) {
  let body: { email?: string; password?: string }

  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 }
    )
  }

  if (!body.email || !body.password) {
    return NextResponse.json(
      { error: 'L’adresse e-mail et le mot de passe sont requis.' },
      { status: 400 }
    )
  }

  let upstreamResponse: Response

  try {
    upstreamResponse = await fetch(apiUrl('/auth/login'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: body.email, password: body.password }),
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
    const message =
      payload?.error ?? payload?.message ?? 'Les identifiants fournis sont invalides.'
    return NextResponse.json({ error: message }, { status: upstreamResponse.status })
  }

  const accessToken = payload?.data?.token ?? null
  const refreshToken = payload?.data?.refresh_token ?? null

  // L'access token repart au client (→ localStorage) ; le refresh token reste
  // dans un cookie httpOnly posé ici (jamais exposé au JS).
  const nextResponse = NextResponse.json(
    { accessToken, user: payload?.data?.user, message: 'Connexion réussie.' },
    { status: upstreamResponse.status }
  )

  if (refreshToken) {
    setRefreshCookie(nextResponse, refreshToken)
  }

  // Provisioning paresseux : crée la ligne `users` à partir du JWT (best-effort).
  if (accessToken) {
    await provisionUser(accessToken)
  }

  return nextResponse
}
