import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import {
  REFRESH_COOKIE,
  clearRefreshCookie,
  setRefreshCookie,
} from '@/lib/server/auth-cookie'

type AuthPayload = {
  data?: { token?: string; refresh_token?: string }
  error?: string
}

/**
 * POST /api/auth/refresh — lit le cookie httpOnly, demande une nouvelle paire
 * au back (rotation) et renvoie le nouvel access token au client (→ localStorage).
 * Échec (cookie absent / refresh expiré/révoqué) → 401 + cookie effacé : le
 * client redirige vers /login.
 */
export async function POST(request: NextRequest) {
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value

  if (!refreshToken) {
    const res = NextResponse.json({ error: 'Session expirée.' }, { status: 401 })
    clearRefreshCookie(res)
    return res
  }

  let upstream: Response

  try {
    upstream = await fetch(apiUrl('/auth/refresh'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
  } catch {
    return NextResponse.json(
      { error: 'Impossible de joindre l’API Gateway.' },
      { status: 502 }
    )
  }

  const payload = (await upstream.json().catch(() => null)) as AuthPayload | null

  if (!upstream.ok) {
    const res = NextResponse.json(
      { error: payload?.error ?? 'Session expirée.' },
      { status: upstream.status }
    )
    clearRefreshCookie(res)
    return res
  }

  const accessToken = payload?.data?.token ?? null
  const newRefresh = payload?.data?.refresh_token ?? null

  const res = NextResponse.json({ accessToken }, { status: 200 })
  // Rotation : le back a émis un nouveau refresh token, on remplace le cookie.
  if (newRefresh) {
    setRefreshCookie(res, newRefresh)
  }
  return res
}
