'use client'

import { useEffect } from 'react'

import { bootstrapSession } from '@/lib/auth-client'

/**
 * Réhydrate la session au chargement depuis le cookie httpOnly refresh (24 h)
 * quand l'access token localStorage manque — typiquement après une éviction du
 * stockage par iOS Safari (ITP), où le cookie same-origin survit bien mieux que
 * le localStorage. Sans ce bootstrap, l'utilisateur apparaîtrait en visiteur
 * jusqu'au prochain 401 (cf. lib/auth-client → bootstrapSession).
 *
 * Monté une seule fois à la racine. Ne rend rien.
 */
export function SessionBootstrap() {
  useEffect(() => {
    void bootstrapSession()
  }, [])

  return null
}
