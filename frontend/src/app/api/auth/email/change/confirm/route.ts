import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { setRefreshCookie } from '@/lib/server/auth-cookie'

type Payload = {
  data?: { token?: string; refresh_token?: string; user?: unknown }
  error?: string
  code?: string
}

export async function POST(request: NextRequest) {
  let body: { token?: string }
  try {
    body = await request.json()
  } catch {
    return NextResponse.json({ error: 'Corps invalide.' }, { status: 400 })
  }
  if (!body.token) {
    return NextResponse.json({ error: 'Token manquant.' }, { status: 400 })
  }

  let upstream: Response
  try {
    upstream = await fetch(apiUrl('/auth/email/change/confirm'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: body.token }),
    })
  } catch {
    return NextResponse.json({ error: 'Impossible de joindre l’API Gateway.' }, { status: 502 })
  }

  const payload = (await upstream.json().catch(() => null)) as Payload | null
  if (!upstream.ok) {
    return NextResponse.json(
      { error: payload?.error ?? 'Confirmation impossible.', code: payload?.code },
      { status: upstream.status },
    )
  }

  const response = NextResponse.json({
    accessToken: payload?.data?.token,
    user: payload?.data?.user,
  })
  if (payload?.data?.refresh_token) {
    setRefreshCookie(response, payload.data.refresh_token)
  }
  return response
}
