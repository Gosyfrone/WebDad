import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { provisionUser } from '@/lib/provision'

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
    acceptedTerms?: boolean
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

  if (body.acceptedTerms !== true) {
    return NextResponse.json(
      { error: 'Les CGU doivent être acceptées pour créer un compte.' },
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

  // Blocage dur (vérification d'e-mail) : l'access token émis par auth NE repart
  // PAS au client et le cookie refresh N'est PAS posé. On l'utilise UNIQUEMENT
  // côté serveur, le temps de provisionner l'identité (users + profil) avec le
  // username/birth_date/gender du formulaire, puis on le jette. L'utilisateur
  // n'a donc aucune session : il doit d'abord vérifier son e-mail puis se
  // connecter. Best-effort + repli dérivé email si le handle est pris.
  if (accessToken) {
    await provisionUser(accessToken, {
      username: body.username,
      birthDate: body.birthDate,
      gender: body.gender,
    })
  }

  // 201 sans token : le front redirige vers la page « consulte ta boîte mail ».
  return NextResponse.json(
    { message: 'Inscription réussie.', emailVerificationRequired: true },
    { status: upstreamResponse.status }
  )
}
