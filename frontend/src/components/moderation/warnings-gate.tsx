'use client'

import { useCallback, useEffect, useState } from 'react'
import { ShieldAlert } from 'lucide-react'

import { ackWarning, fetchPendingWarnings, type Warning } from '@/lib/reports'
import { getAccessToken } from '@/lib/auth-client'
import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'

/**
 * Interception des avertissements de modération (système asynchrone) : à chaque
 * montée de l'app et à chaque retour de focus, on interroge les avertissements
 * NON acquittés de l'utilisateur courant ; s'il en existe, on affiche une modale
 * BLOQUANTE (un avertissement à la fois). « J'ai compris » acquitte côté serveur
 * et passe au suivant. C'est la « prochaine requête/connexion » qui révèle le
 * Warn émis par un modérateur.
 *
 * Monté dans le layout `(app)` (espace authentifié), à côté des autres gates.
 */
export function WarningsGate() {
  const t = useT()
  const [queue, setQueue] = useState<Warning[]>([])
  const [acking, setAcking] = useState(false)

  const poll = useCallback(async () => {
    if (!getAccessToken()) return // visiteur / non connecté : rien à faire
    try {
      const pending = await fetchPendingWarnings()
      if (pending.length > 0) setQueue(pending)
    } catch {
      /* best-effort : on réessaiera au prochain focus */
    }
  }, [])

  useEffect(() => {
    void poll()
    const onFocus = () => void poll()
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [poll])

  const current = queue[0]

  async function onAck() {
    if (!current) return
    setAcking(true)
    try {
      await ackWarning(current.id)
    } catch {
      /* on retire quand même de la file locale pour ne pas bloquer l'UI */
    } finally {
      setQueue((prev) => prev.slice(1))
      setAcking(false)
    }
  }

  return (
    // IMPORTANT : on pilote `open` par une variable (jamais d'unmount d'un Dialog
    // Radix encore « ouvert » — sinon le verrou de scroll / les styles posés sur
    // <body> ne sont pas nettoyés et bloquent les interactions de toute l'appli).
    // Modale bloquante : fermeture uniquement via « J'ai compris ».
    <Dialog open={Boolean(current)} onOpenChange={() => undefined}>
      {current && (
        <DialogContent className="[&>button]:hidden">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-destructive">
              <ShieldAlert className="h-5 w-5" aria-hidden />
              {t('warn.modal_title')}
            </DialogTitle>
          </DialogHeader>
          <p className="whitespace-pre-wrap break-words text-sm">{current.message}</p>
          <DialogFooter>
            <Button onClick={() => void onAck()} disabled={acking}>
              {t('warn.modal_ack')}
            </Button>
          </DialogFooter>
        </DialogContent>
      )}
    </Dialog>
  )
}
