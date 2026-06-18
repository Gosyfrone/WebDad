'use client'

import { useEffect, useRef } from 'react'

import { apiFetch, getAccessToken } from '@/lib/auth-client'

const OFFLINE_ENDPOINT = '/api/activity/offline'

/**
 * Synchronise la présence avec le cycle de vie de l'onglet :
 * - montage / retour BFCache : en ligne ;
 * - fermeture, refresh, navigation dure : hors ligne.
 */
export function ActivityLifecycle() {
  const onlineInFlight = useRef(false)

  useEffect(() => {
    async function markOnline() {
      if (onlineInFlight.current || !getAccessToken()) return
      onlineInFlight.current = true
      try {
        await apiFetch('/profils/me/activity', { method: 'PATCH' })
      } catch {
        // Best-effort : le prochain chargement ou refresh d'activité rattrapera.
      } finally {
        onlineInFlight.current = false
      }
    }

    function markOffline() {
      const token = getAccessToken()
      if (!token) return
      void fetch(OFFLINE_ENDPOINT, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        keepalive: true,
      }).catch(() => {})
    }

    void markOnline()

    function handlePageShow() {
      void markOnline()
    }

    window.addEventListener('pageshow', handlePageShow)
    window.addEventListener('pagehide', markOffline)
    return () => {
      window.removeEventListener('pageshow', handlePageShow)
      window.removeEventListener('pagehide', markOffline)
    }
  }, [])

  return null
}
