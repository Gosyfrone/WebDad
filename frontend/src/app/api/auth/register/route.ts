import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'

type RegisterResponse = {
  token?: string
  accessToken?: string
  jwt?: string
  data?: {
    token?: string
    accessToken?: string
    jwt?: string
  }
  message?: string
  error?: string
}

function extractToken(payload: RegisterResponse | null): string | null {
  if (!payload || typeof payload !== 'object') {
    return null
  }

  return (
    payload.token ??
    payload.accessToken ??
    payload.jwt ??
    payload.data?.token ??
    payload.data?.accessToken ??
    payload.data?.jwt ??
    null
  )
}

export async function POST(request: NextRequest) {
  let body: { username?: string; email?: string; password?: string }

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
      headers: {
        'Content-Type': 'application/json',
      },
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

  const contentType = upstreamResponse.headers.get('content-type') ?? ''
  const rawBody =
    contentType.includes('application/json')
      ? await upstreamResponse.json().catch(() => null)
      : await upstreamResponse.text().catch(() => '')

  if (!upstreamResponse.ok) {
    const payload = rawBody as Partial<RegisterResponse> | string | null
    const message =
      (payload && typeof payload === 'object' && (payload.error ?? payload.message)) ||
      (typeof payload === 'string' && payload.trim()) ||
      'L’inscription a échoué.'

    return NextResponse.json({ error: message }, { status: upstreamResponse.status })
  }

  const payload = rawBody as RegisterResponse | string | null
  const responseBody =
    payload && typeof payload === 'object' ? payload : { message: 'Inscription réussie.' }

  const nextResponse = NextResponse.json(responseBody, {
    status: upstreamResponse.status,
  })

  const setCookieHeader = upstreamResponse.headers.get('set-cookie')
  const token = extractToken(payload && typeof payload === 'object' ? payload : null)

  if (setCookieHeader) {
    nextResponse.headers.set('set-cookie', setCookieHeader)
  } else if (token) {
    nextResponse.cookies.set({
      name: 'breezy-token',
      value: token,
      httpOnly: true,
      sameSite: 'lax',
      secure: process.env.NODE_ENV === 'production',
      path: '/',
      maxAge: 60 * 60 * 24 * 7,
    })
  }

  return nextResponse
}