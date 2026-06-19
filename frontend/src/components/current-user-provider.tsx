'use client'

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'

import { getMe } from '@/lib/api'
import { getAccessToken } from '@/lib/auth-client'
import { getMyProfil, subscribeProfilUpdated } from '@/lib/profil-client'
import { useSession, type Session } from '@/lib/session'
import type { ProfilDetails } from '@/types'

/**
 * Store de l'utilisateur connecté : source unique pour l'identité (JWT) et le
 * profil (username / display_name / avatar / rôle…). Avant, chaque composant
 * refetchait `getMyProfil` / `getMe` indépendamment (sidebar, header, composer…)
 * et redécodait le JWT — ici, un seul fetch au montage, partagé par contexte.
 *
 * Visiteur (pas de session) → `session`/`profil` à `null`, l'app reste utilisable.
 */
export interface CurrentUserValue {
  /** Identité dérivée du JWT (`null` si visiteur non connecté). */
  session: Session | null
  /** Profil complet (`null` tant qu'il n'est pas chargé, ou visiteur). */
  profil: ProfilDetails | null
  /** Username provisoire (compte créé par un admin) à remplacer. */
  usernamePending: boolean
  /** Langue choisie sur le compte (`null` = le navigateur fait foi). */
  preferredLocale: string | null
  isAdmin: boolean
  isModerator: boolean
  isLoading: boolean
  /** Recharge profil + métadonnées compte depuis le serveur. */
  refresh: () => Promise<void>
}

const CurrentUserContext = createContext<CurrentUserValue | null>(null)

export function CurrentUserProvider({ children }: { children: ReactNode }) {
  const session = useSession()
  const [profil, setProfil] = useState<ProfilDetails | null>(null)
  const [usernamePending, setUsernamePending] = useState(false)
  const [preferredLocale, setPreferredLocale] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const load = useCallback(async () => {
    if (!getAccessToken()) {
      setProfil(null)
      setUsernamePending(false)
      setPreferredLocale(null)
      return
    }
    setIsLoading(true)
    try {
      const [profilData, me] = await Promise.all([getMyProfil(), getMe()])
      setProfil(profilData)
      setUsernamePending(me.usernamePending)
      setPreferredLocale(me.preferredLocale)
    } catch {
      // Session expirée / réseau : on conserve les fallbacks des consommateurs.
    } finally {
      setIsLoading(false)
    }
  }, [])

  // (Re)charge quand l'utilisateur change (login / refresh / logout) ; dédupliqué
  // pour ne pas refetch au simple tick d'hydratation de la session.
  const loadedFor = useRef<string | null>(null)
  useEffect(() => {
    const uid = session?.userId ?? null
    if (loadedFor.current === uid) return
    loadedFor.current = uid
    void load()
  }, [session?.userId, load])

  // Édition de profil : applique la version à jour sans refetch (event profil-client).
  useEffect(() => subscribeProfilUpdated(setProfil), [])

  const value: CurrentUserValue = {
    session,
    profil,
    usernamePending,
    preferredLocale,
    isAdmin: session?.role === 'administrator',
    isModerator: session?.role === 'moderator' || session?.role === 'administrator',
    isLoading,
    refresh: load,
  }

  return (
    <CurrentUserContext.Provider value={value}>{children}</CurrentUserContext.Provider>
  )
}

/** Utilisateur connecté (store). Lève hors d'un `CurrentUserProvider`. */
export function useCurrentUser(): CurrentUserValue {
  const ctx = useContext(CurrentUserContext)
  if (!ctx) {
    throw new Error('useCurrentUser doit être utilisé dans un CurrentUserProvider')
  }
  return ctx
}
