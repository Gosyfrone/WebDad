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
  let body: {
    username?: string
    birthDate?: string
    gender?: string
    email?: string
    password?: string
  }

  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 }
    )
  }

  if (!body.username || !body.email || !body.password) {
    return NextResponse.json(
      {
        error: 'Le nom d’utilisateur, l’adresse e-mail et le mot de passe sont requis.',
      },
      { status: 400 }
    )
  }

  let upstreamResponse: Response

  try {
    upstreamResponse = await fetch(apiUrl('/auth/register'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: body.username,
        email: body.email,
        password: body.password,
      }),
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
    const message = payload?.error ?? payload?.message ?? 'L’inscription a échoué.'
    return NextResponse.json({ error: message }, { status: upstreamResponse.status })
  }

  const accessToken = payload?.data?.token ?? null
  const refreshToken = payload?.data?.refresh_token ?? null

  // L'access token repart au client (→ localStorage) ; le refresh token reste
  // dans un cookie httpOnly posé ici (jamais exposé au JS).
  const nextResponse = NextResponse.json(
    { accessToken, user: payload?.data?.user, message: 'Inscription réussie.' },
    { status: upstreamResponse.status }
  )

  if (refreshToken) {
    setRefreshCookie(nextResponse, refreshToken)
  }

  // Provisioning : crée la ligne `users` avec le username CHOISI par l'utilisateur
  // (POST /users). L'inscription auto-connecte, d'où le provisioning ici.
  // Best-effort + repli dérivé email si le handle est pris (cf. lib/provision).
  if (accessToken) {
    await provisionUser(accessToken, {
      username: body.username,
      birthDate: body.birthDate,
      gender: body.gender,
    })
  }

  return nextResponse
}
