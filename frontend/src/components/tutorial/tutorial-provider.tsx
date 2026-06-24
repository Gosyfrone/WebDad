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

import { saveMyTutorialDone } from '@/lib/profil-client'
import { TUTORIAL_STEPS, clampStepIndex } from '@/lib/tutorial-steps'
import { useCurrentUser } from '@/components/current-user-provider'
import { useAuthGate } from '@/components/auth-prompt-provider'

interface TutorialValue {
  /** Le tour est-il en cours d'affichage ? */
  isActive: boolean
  /** Index de l'étape courante (0-based). */
  stepIndex: number
  /** Démarre (ou redémarre) le tour depuis la première étape. */
  start: () => void
  /** Ferme le tour et le marque comme vu (« Ignorer » / fin). */
  stop: () => void
  /** Étape suivante (clôture le tour si on est à la dernière). */
  next: () => void
  /** Étape précédente (sans effet à la première). */
  prev: () => void
}

const TutorialContext = createContext<TutorialValue | null>(null)

/**
 * Pilote du didacticiel guidé. Auto-démarre à la « première connexion » : dès que
 * le profil est chargé pour un utilisateur connecté dont `tutorialDone === false`.
 * La clôture (fin ou « Ignorer ») persiste `tutorial_done=true` côté serveur
 * (PATCH /profils/me), donc le tour n'est plus proposé — sur tous les appareils.
 *
 * Le rendu visuel (overlay + infobulle) est délégué à <TutorialTooltip>, monté
 * en frère dans le layout ; ce provider ne porte que l'état et les transitions.
 */
export function TutorialProvider({ children }: { children: ReactNode }) {
  const { profil } = useCurrentUser()
  const { isVisitor } = useAuthGate()
  const [isActive, setIsActive] = useState(false)
  const [stepIndex, setStepIndex] = useState(0)

  // Garde-fou : auto-démarrage une seule fois par utilisateur chargé, pour ne
  // pas relancer le tour tant que la persistance serveur (asynchrone) n'a pas
  // encore rafraîchi `profil.tutorialDone`.
  const autoStartedFor = useRef<string | null>(null)

  const start = useCallback(() => {
    setStepIndex(0)
    setIsActive(true)
  }, [])

  const persistDone = useCallback(() => {
    // Best-effort : on n'attend pas, et une erreur réseau ne bloque pas l'UX
    // (le tour ne réapparaît pas dans la session courante ; au pire au prochain
    // appareil/visite). On évite le PATCH si déjà marqué vu.
    if (profil && !profil.tutorialDone) {
      void saveMyTutorialDone(true).catch(() => undefined)
    }
  }, [profil])

  const stop = useCallback(() => {
    setIsActive(false)
    persistDone()
  }, [persistDone])

  const next = useCallback(() => {
    setStepIndex((index) => {
      if (index >= TUTORIAL_STEPS.length - 1) {
        setIsActive(false)
        persistDone()
        return index
      }
      return clampStepIndex(index + 1)
    })
  }, [persistDone])

  const prev = useCallback(() => {
    setStepIndex((index) => clampStepIndex(index - 1))
  }, [])

  // Auto-démarrage à la première connexion (profil chargé, non-visiteur, non vu).
  useEffect(() => {
    if (isVisitor || !profil) return
    if (autoStartedFor.current === profil.userId) return
    autoStartedFor.current = profil.userId
    if (!profil.tutorialDone) start()
  }, [isVisitor, profil, start])

  const value: TutorialValue = { isActive, stepIndex, start, stop, next, prev }

  return <TutorialContext.Provider value={value}>{children}</TutorialContext.Provider>
}

/** Accès au pilote du didacticiel. Lève hors d'un <TutorialProvider>. */
export function useTutorial(): TutorialValue {
  const ctx = useContext(TutorialContext)
  if (!ctx) {
    throw new Error('useTutorial doit être utilisé dans un <TutorialProvider>')
  }
  return ctx
}
