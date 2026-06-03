import type { NextResponse } from 'next/server'

/**
 * Gestion du cookie httpOnly portant le refresh token (côté serveur / BFF).
 *
 * Le refresh token (24 h) vit dans ce cookie httpOnly same-origin : invisible
 * au JS client (pas de vol via XSS) et renvoyé automatiquement par le
 * navigateur sur `/api/auth/{refresh,logout}`. L'access token, lui, est court
 * (15 min) et vit en localStorage côté client (cf. lib/auth-client).
 */

export const REFRESH_COOKIE = 'breezy-refresh'

/** Durée de vie du cookie, alignée sur REFRESH_EXPIRY du back (24 h). */
const REFRESH_MAX_AGE = 60 * 60 * 24

function baseCookie(value: string, maxAge: number) {
  return {
    name: REFRESH_COOKIE,
    value,
    httpOnly: true,
    sameSite: 'lax' as const,
    secure: process.env.NODE_ENV === 'production',
    path: '/',
    maxAge,
  }
}

/** Pose (ou renouvelle) le cookie refresh sur la réponse. */
export function setRefreshCookie(res: NextResponse, value: string): void {
  res.cookies.set(baseCookie(value, REFRESH_MAX_AGE))
}

/** Efface le cookie refresh (déconnexion / refresh invalide). */
export function clearRefreshCookie(res: NextResponse): void {
  res.cookies.set(baseCookie('', 0))
}
