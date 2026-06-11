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
  code?: string
}

type UserLookupPayload = {
  data?: {
    id?: string
  }
  error?: string
  message?: string
}

const emailLikePattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export async function POST(request: NextRequest) {
  let body: { email?: string; identifier?: string; password?: string }

  try {
    body = await request.json()
  } catch {
    return NextResponse.json(
      { error: 'Le corps de la requête est invalide.' },
      { status: 400 }
    )
  }

  const identifier = (body.identifier ?? body.email ?? '').trim()

  if (!identifier || !body.password) {
    return NextResponse.json(
      { error: 'L’identifiant et le mot de passe sont requis.' },
      { status: 400 }
    )
  }

  let loginBody: { email?: string; user_id?: string; password: string } = {
    password: body.password,
  }

  if (emailLikePattern.test(identifier)) {
    loginBody = { ...loginBody, email: identifier }
  } else {
    let userResponse: Response
    try {
      userResponse = await fetch(
        apiUrl(`/users/by-username/${encodeURIComponent(identifier)}`),
        { cache: 'no-store' }
      )
    } catch {
      return NextResponse.json(
        { error: 'Impossible de joindre l’API Gateway.' },
        { status: 502 }
      )
    }

    if (!userResponse.ok) {
      return NextResponse.json(
        { error: 'Les identifiants fournis sont invalides.' },
        { status: 401 }
      )
    }

    const userPayload = (await userResponse.json().catch(() => null)) as
      | UserLookupPayload
      | null
    const userId = userPayload?.data?.id
    if (!userId) {
      return NextResponse.json(
        { error: 'Les identifiants fournis sont invalides.' },
        { status: 401 }
      )
    }

    loginBody = { ...loginBody, user_id: userId }
  }

  let upstreamResponse: Response

  try {
    upstreamResponse = await fetch(apiUrl('/auth/login'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(loginBody),
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
    // E-mail non vérifié : on relaie le code machine pour que la page login
    // propose le renvoi du mail de vérification.
    if (payload?.code === 'email_not_verified') {
      return NextResponse.json(
        { error: message, code: 'email_not_verified' },
        { status: upstreamResponse.status }
      )
    }
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
    void provisionUser(accessToken)
  }

  return nextResponse
}
