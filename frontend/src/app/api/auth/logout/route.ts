import { NextRequest, NextResponse } from 'next/server'

import { apiUrl } from '@/lib/config'
import { REFRESH_COOKIE, clearRefreshCookie } from '@/lib/server/auth-cookie'

/**
 * POST /api/auth/logout — révoque le refresh token côté back (best-effort) et
 * efface le cookie httpOnly dans tous les cas. Le client efface ensuite son
 * access token (localStorage) et redirige vers /login.
 */
export async function POST(request: NextRequest) {
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value

  if (refreshToken) {
    try {
      await fetch(apiUrl('/auth/logout'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })
    } catch {
      // best-effort : la déconnexion locale prime sur la révocation distante.
    }
  }

  const res = NextResponse.json({ message: 'Déconnecté.' }, { status: 200 })
  clearRefreshCookie(res)
  return res
}
