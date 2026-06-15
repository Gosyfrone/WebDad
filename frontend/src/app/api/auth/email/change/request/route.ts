import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'

export async function POST(request: NextRequest) {
  const authorization = request.headers.get('authorization')
  if (!authorization) {
    return NextResponse.json({ error: 'Non authentifié.' }, { status: 401 })
  }

  let body: { email?: string }
  try {
    body = await request.json()
  } catch {
    return NextResponse.json({ error: 'Corps invalide.' }, { status: 400 })
  }

  let upstream: Response
  try {
    upstream = await fetch(apiUrl('/auth/email/change/request'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: authorization },
      body: JSON.stringify({ email: body.email }),
    })
  } catch {
    return NextResponse.json({ error: 'Impossible de joindre l’API Gateway.' }, { status: 502 })
  }

  const payload = await upstream.json().catch(() => null)
  return NextResponse.json(payload ?? {}, { status: upstream.status })
}
