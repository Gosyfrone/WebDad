import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { provisionUser } from '@/lib/provision'
import { setRefreshCookie } from '@/lib/server/auth-cookie'

type OAuthExchangePayload = {
  data?: {
    token?: string
    refresh_token?: string
    user?: unknown
  }
  message?: string
  error?: string
}

// GET /api/auth/oauth/[provider] → renvoie l'URL d'autorisation du provider
export async function GET(
  _request: NextRequest,
  { params }: { params: { provider: string } }
) {
  const { provider } = params

  let upstreamResponse: Response
  try {
    upstreamResponse = await fetch(apiUrl(`/auth/oauth/${provider}/url`))
  } catch {
    return NextResponse.json(
      { error: "Impossible de joindre l'API Gateway." },
      { status: 502 }
    )
  }

  const payload = await upstreamResponse.json().catch(() => null)

  if (!upstreamResponse.ok) {
    return NextResponse.json(
      { error: payload?.error ?? payload?.message ?? 'Erreur OAuth.' },
      { status: upstreamResponse.status }
    )
  }

  return NextResponse.json(payload, { status: 200 })
}

// POST /api/auth/oauth/[provider] → échange le code contre un token
export async function POST(
  request: NextRequest,
  { params }: { params: { provider: string } }
) {
  const { provider } = params

  let body: { code?: string; state?: string }
  try {
    body = await request.json()
  } catch {
    return NextResponse.json({ error: 'Corps de requête invalide.' }, { status: 400 })
  }

  if (!body.code) {
    return NextResponse.json(
      { error: "Code d'autorisation manquant." },
      { status: 400 }
    )
  }

  let upstreamResponse: Response
  try {
    upstreamResponse = await fetch(apiUrl(`/auth/oauth/${provider}/exchange`), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code: body.code, state: body.state }),
    })
  } catch {
    return NextResponse.json(
      { error: "Impossible de joindre l'API Gateway." },
      { status: 502 }
    )
  }

  const payload = (await upstreamResponse.json().catch(() => null)) as OAuthExchangePayload | null

  if (!upstreamResponse.ok) {
    return NextResponse.json(
      { error: payload?.error ?? payload?.message ?? 'Échange OAuth échoué.' },
      { status: upstreamResponse.status }
    )
  }

  const accessToken = payload?.data?.token ?? null
  const refreshToken = payload?.data?.refresh_token ?? null

  const nextResponse = NextResponse.json(
    { accessToken, user: payload?.data?.user, message: 'Connexion réussie.' },
    { status: 200 }
  )

  if (refreshToken) {
    setRefreshCookie(nextResponse, refreshToken)
  }

  if (accessToken) {
    await provisionUser(accessToken)
  }

  return nextResponse
}
