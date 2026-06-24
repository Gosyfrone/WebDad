'use client'

import { useEffect, useState } from 'react'
import { Check, Copy, Loader2, Share2 } from 'lucide-react'

import {
  IdentityLockedError,
  PeerKeyMissingError,
  currentUserId,
  sendMessage,
  startDM,
} from '@/lib/messages'
import {
  absoluteUrl,
  canNativeShare,
  copyLink,
  getRecentShareTargets,
  nativeShare,
  recordRecentShareTarget,
} from '@/lib/share'
import type { RelationUser } from '@/types'
import { useToast } from '@/hooks/use-toast'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { useLanguage } from '@/components/language-provider'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { UserSearch } from '@/components/messages/user-search'

interface ShareDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Chemin relatif vers la ressource (`/posts/<id>` ou `/profil/<handle>`). */
  url: string
  /** Nature du contenu partagé → choisit le préfixe du message privé. */
  kind: 'post' | 'profile'
  /** Titre passé à la feuille de partage native. */
  title?: string
}

/**
 * Dialogue de partage réutilisable (post + profil). Trois voies :
 *   - **message privé** : recherche d'une personne → `startDM` + `sendMessage`
 *     (chiffré E2EE, le lien voyage en clair DANS le message chiffré) ;
 *   - **partage natif** : `navigator.share()` si dispo (mobile/desktop) ;
 *   - **copier le lien** : presse-papiers (repli universel).
 */
export function ShareDialog({ open, onOpenChange, url, kind, title }: ShareDialogProps) {
  const { t } = useLanguage()
  const { toast } = useToast()
  const { isVisitor, promptLogin } = useAuthGate()

  const [hasNativeShare, setHasNativeShare] = useState(false)
  const [copied, setCopied] = useState(false)
  const [sending, setSending] = useState(false)
  const [recentUsers, setRecentUsers] = useState<RelationUser[]>([])

  // navigator.share n'est connu qu'au montage client (évite un mismatch SSR).
  useEffect(() => {
    setHasNativeShare(canNativeShare())
  }, [])

  // Réinitialise l'état visuel et recharge les récents à chaque ouverture.
  useEffect(() => {
    if (!open) return
    setCopied(false)
    setRecentUsers(getRecentShareTargets().map((u) => ({ ...u, bio: '', certification: 'none' })))
  }, [open])

  const absolute = absoluteUrl(url)
  // Profil : le lien seul (aperçu rendu côté chat). Post : court préfixe + lien.
  const messageText =
    kind === 'post' ? `${t('share.post_message_prefix')}\n${absolute}` : absolute

  async function onNativeShare() {
    const ok = await nativeShare({ title, url: absolute })
    if (ok) onOpenChange(false)
  }

  async function onCopy() {
    const ok = await copyLink(absolute)
    if (ok) {
      setCopied(true)
      toast({ title: t('share.copied') })
    } else {
      toast({ title: t('common.action_failed'), variant: 'destructive' })
    }
  }

  async function onPickUser(user: RelationUser) {
    if (sending) return
    setSending(true)
    try {
      const conv = await startDM(user.id)
      await sendMessage(conv, messageText)
      recordRecentShareTarget(user)
      toast({ title: t('share.sent') })
      onOpenChange(false)
    } catch (err) {
      if (err instanceof PeerKeyMissingError) {
        toast({ title: t('messages.peer_not_activated'), variant: 'brand' })
      } else if (err instanceof IdentityLockedError) {
        toast({ title: t('messages.passphrase.unlock_title'), variant: 'brand' })
      } else {
        toast({ title: t('share.send_failed'), variant: 'destructive' })
      }
    } finally {
      setSending(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('share.title')}</DialogTitle>
          <DialogDescription>
            {t(kind === 'post' ? 'share.post_desc' : 'share.profile_desc')}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {/* Partage natif + copier le lien */}
          <div className="flex gap-2">
            {hasNativeShare && (
              <button
                type="button"
                onClick={onNativeShare}
                className="flex flex-1 items-center justify-center gap-2 rounded-full border border-input bg-background/60 px-4 py-2.5 text-sm font-medium transition-colors hover:bg-accent"
              >
                <Share2 className="h-4 w-4" />
                {t('share.via')}
              </button>
            )}
            <button
              type="button"
              onClick={onCopy}
              className="flex flex-1 items-center justify-center gap-2 rounded-full border border-input bg-background/60 px-4 py-2.5 text-sm font-medium transition-colors hover:bg-accent"
            >
              {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
              {copied ? t('share.copied') : t('share.copy')}
            </button>
          </div>

          {/* Envoi en message privé */}
          <div className="flex flex-col gap-2">
            <div>
              <p className="text-sm font-semibold text-foreground">{t('share.send_dm')}</p>
              <p className="text-xs text-muted-foreground">{t('share.send_dm_hint')}</p>
            </div>
            {isVisitor ? (
              <button
                type="button"
                onClick={promptLogin}
                className="rounded-lg border border-dashed border-border px-3 py-3 text-center text-sm text-muted-foreground transition-colors hover:bg-accent"
              >
                {t('share.login_required')}
              </button>
            ) : sending ? (
              <div className="flex justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-[#5B6CFF] dark:text-[#9aa6ff]" />
              </div>
            ) : (
              <UserSearch
                excludeIds={[currentUserId()]}
                onPick={onPickUser}
                autoFocus={false}
                recentUsers={recentUsers}
                recentLabel={t('share.recent')}
              />
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
