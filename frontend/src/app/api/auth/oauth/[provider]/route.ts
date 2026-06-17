import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { markLoginActivity, provisionUser } from '@/lib/provision'
import { setRefreshCookie } from '@/lib/server/auth-cookie'

type OAuthURLPayload = {
  data?: {
    url?: string
    state?: string
  }
  message?: string
  error?: string
}

type OAuthExchangePayload = {
  data?: {
    token?: string
    refresh_token?: string
    user?: unknown
    onboarding_required?: boolean
    pending_token?: string
    email?: string
  }
  message?: string
  error?: string
}

const usernamePattern = /^[a-zA-Z0-9_]{3,24}$/
const reservedUsernames = new Set([
  'me',
  'admin',
  'root',
  'users',
  'by-username',
  'null',
  'undefined',
])
const minBirthDate = '1900-01-01'

function isDateInputValue(value: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(value)
}

function minimumAgeBirthDate(): string {
  const date = new Date()
  date.setFullYear(date.getFullYear() - 13)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// GET /api/auth/oauth/[provider] → renvoie l'URL d'autorisation du provider
export async function GET(
  request: NextRequest,
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

  const payload = (await upstreamResponse.json().catch(() => null)) as OAuthURLPayload | null

  if (!upstreamResponse.ok) {
    return NextResponse.json(
      { error: payload?.error ?? payload?.message ?? 'Erreur OAuth.' },
      { status: upstreamResponse.status }
    )
  }

  // Aplatit data.{url,state} pour simplifier la lecture côté client
  return NextResponse.json(
    { url: payload?.data?.url, state: payload?.data?.state },
    { status: 200 }
  )
}

// POST /api/auth/oauth/[provider] → échange le code contre un token
export async function POST(
  request: NextRequest,
  { params }: { params: { provider: string } }
) {
  const { provider } = params

  let body: {
    code?: string
    state?: string
    pendingToken?: string
    username?: string
    birthDate?: string
    acceptedTerms?: boolean
  }
  try {
    body = await request.json()
  } catch {
    return NextResponse.json({ error: 'Corps de requête invalide.' }, { status: 400 })
  }

  if (body.pendingToken) {
    return completeOAuthSignup(provider, body)
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
  if (payload?.data?.onboarding_required) {
    return NextResponse.json(
      {
        onboardingRequired: true,
        pendingToken: payload.data.pending_token,
        email: payload.data.email,
      },
      { status: 200 }
    )
  }

  const nextResponse = NextResponse.json(
    { accessToken, user: payload?.data?.user, message: 'Connexion réussie.' },
    { status: 200 }
  )

  if (refreshToken) {
    setRefreshCookie(nextResponse, refreshToken)
  }

  if (accessToken) {
    await provisionUser(accessToken)
    void markLoginActivity(accessToken)
  }

  return nextResponse
}

async function completeOAuthSignup(
  provider: string,
  body: {
    pendingToken?: string
    username?: string
    birthDate?: string
    acceptedTerms?: boolean
  }
) {
  const username = body.username?.trim()
  if (!body.pendingToken || !username || !body.birthDate) {
    return NextResponse.json(
      { error: "Le token, le nom d'utilisateur et la date de naissance sont requis." },
      { status: 400 }
    )
  }
  if (body.acceptedTerms !== true) {
    return NextResponse.json(
      { error: 'Les CGU doivent être acceptées pour créer un compte.' },
      { status: 400 }
    )
  }
  if (
    !usernamePattern.test(username) ||
    reservedUsernames.has(username.toLowerCase())
  ) {
    return NextResponse.json(
      { error: "Nom d'utilisateur invalide." },
      { status: 400 }
    )
  }
  if (
    !isDateInputValue(body.birthDate) ||
    body.birthDate < minBirthDate ||
    body.birthDate > minimumAgeBirthDate()
  ) {
    return NextResponse.json(
      { error: 'Date de naissance invalide.' },
      { status: 400 }
    )
  }
  let usernameLookup: Response
  try {
    usernameLookup = await fetch(
      apiUrl(`/users/by-username/${encodeURIComponent(username)}`),
      { cache: 'no-store' }
    )
  } catch {
    return NextResponse.json(
      { error: "Impossible de vérifier le nom d'utilisateur." },
      { status: 502 }
    )
  }
  if (usernameLookup.status === 200) {
    return NextResponse.json(
      { error: "Ce nom d'utilisateur est déjà pris." },
      { status: 409 }
    )
  }
  if (usernameLookup.status !== 404) {
    return NextResponse.json(
      { error: "Impossible de vérifier le nom d'utilisateur." },
      { status: 502 }
    )
  }

  let upstreamResponse: Response
  try {
    upstreamResponse = await fetch(apiUrl(`/auth/oauth/${provider}/complete`), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pending_token: body.pendingToken,
        accepted_terms: true,
      }),
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
      { error: payload?.error ?? payload?.message ?? 'Création OAuth échouée.' },
      { status: upstreamResponse.status }
    )
  }

  const accessToken = payload?.data?.token ?? null
  const refreshToken = payload?.data?.refresh_token ?? null
  const nextResponse = NextResponse.json(
    { accessToken, user: payload?.data?.user, message: 'Compte créé.' },
    { status: 200 }
  )

  if (refreshToken) {
    setRefreshCookie(nextResponse, refreshToken)
  }

  if (accessToken) {
    await provisionUser(accessToken, {
      username,
      birthDate: body.birthDate,
    })
    void markLoginActivity(accessToken)
  }

  return nextResponse
}
