import { NextRequest, NextResponse } from 'next/server'

import { markLogoutActivity } from '@/lib/provision'

/**
 * POST /api/activity/offline — marque la session comme absente sans révoquer le
 * refresh token. Utilisé au `pagehide` quand l'utilisateur ferme l'onglet.
 */
export async function POST(request: NextRequest) {
  const authorization = request.headers.get('authorization')
  const accessToken = authorization?.toLowerCase().startsWith('bearer ')
    ? authorization.slice(7).trim()
    : ''

  if (!accessToken) {
    return NextResponse.json({ error: 'Non authentifié.' }, { status: 401 })
  }

  await markLogoutActivity(accessToken)
  return NextResponse.json({ ok: true }, { status: 200 })
}
