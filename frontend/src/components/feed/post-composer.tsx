'use client'

import { useEffect, useRef, useState } from 'react'
import { Image as ImageIcon, Smile, BarChart2, Loader2 } from 'lucide-react'

import { cn } from '@/lib/utils'
import { getMyProfil, subscribeProfilUpdated } from '@/lib/profil-client'
import { createPost, notifyPostCreated } from '@/lib/posts'
import { useToast } from '@/hooks/use-toast'
import type { ProfilDetails } from '@/types'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { EmojiPicker } from '@/components/feed/emoji-picker'

const MAX_CHARS = 280

interface PostComposerProps {
  /** Classes du conteneur externe (padding/bordure gérés par le parent). */
  className?: string
  /** Place le curseur dans le champ dès le montage (utile en modale). */
  autoFocus?: boolean
  /** Libellé du bouton d'envoi. */
  submitLabel?: string
  /** Appelé après une publication réussie (ex. fermer la popup). */
  onPosted?: (content: string) => void
}

/**
 * Formulaire de rédaction d'un post (avatar + zone de saisie + barre d'outils).
 *
 * Partagé entre la zone de composition inline du fil (`CreatePost`) et la
 * popup déclenchée depuis la sidebar (`CreatePostDialog`) — une seule source
 * de vérité pour la limite de caractères et la validation.
 */
export function PostComposer({
  className,
  autoFocus = false,
  submitLabel = 'Breezer',
  onPosted,
}: PostComposerProps) {
  const { toast } = useToast()
  const [content, setContent] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [initial, setInitial] = useState('U')
  const [submitting, setSubmitting] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const remaining = MAX_CHARS - content.length
  const isEmpty = content.trim().length === 0
  const isOver = remaining < 0

  // Avatar de l'utilisateur courant (resync sur édition du profil, comme la
  // sidebar). Repli silencieux sur l'initiale si la session/le profil manque.
  useEffect(() => {
    let cancelled = false
    function apply(profil: ProfilDetails) {
      if (cancelled) return
      setAvatarUrl(profil.avatarUrl)
      setInitial((profil.displayName || profil.username || 'U').charAt(0).toUpperCase())
    }
    getMyProfil().then(apply).catch(() => {})
    const unsubscribe = subscribeProfilUpdated(apply)
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  async function handleSubmit() {
    if (isEmpty || isOver || submitting) return
    setSubmitting(true)
    try {
      const post = await createPost(content.trim())
      notifyPostCreated(post) // le fil prépend sans refetch
      onPosted?.(content)
      setContent('')
    } catch {
      toast({ title: 'Publication impossible', variant: 'destructive' })
    } finally {
      setSubmitting(false)
    }
  }

  /** Insère l'emoji à la position du curseur (ou à la fin) et restaure le focus. */
  function insertEmoji(emoji: string) {
    const el = textareaRef.current
    const start = el?.selectionStart ?? content.length
    const end = el?.selectionEnd ?? content.length
    setContent(content.slice(0, start) + emoji + content.slice(end))

    requestAnimationFrame(() => {
      if (!el) return
      const caret = start + emoji.length
      el.focus()
      el.setSelectionRange(caret, caret)
    })
  }

  return (
    <div className={cn('flex gap-3', className)}>
      <Avatar className="mt-1 h-10 w-10 shrink-0 shadow-[0_12px_30px_rgba(91,108,255,0.22)]">
        {avatarUrl && <AvatarImage src={avatarUrl} alt="" />}
        <AvatarFallback className="bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white">
          {initial}
        </AvatarFallback>
      </Avatar>

      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <textarea
          ref={textareaRef}
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Ça breez ? 🌴"
          rows={3}
          autoFocus={autoFocus}
          className="w-full cursor-text resize-none bg-transparent text-xl text-foreground caret-[#5B6CFF] placeholder:text-muted-foreground focus:outline-none"
        />

        <Separator className="bg-border" />

        {/* Toolbar */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1 text-[#5B6CFF]">
            <ActionIcon icon={ImageIcon} label="Ajouter une image" />
            <EmojiPicker onSelect={insertEmoji}>
              <button
                type="button"
                aria-label="Ajouter un emoji"
                className="rounded-full p-2 transition-colors hover:bg-primary/10"
              >
                <Smile className="h-5 w-5" />
              </button>
            </EmojiPicker>
            <ActionIcon icon={BarChart2} label="Ajouter un sondage" />
          </div>

          <div className="flex items-center gap-3">
            {/* Compteur de caractères */}
            {content.length > 0 && (
              <span
                className={
                  isOver
                    ? 'text-sm font-bold text-destructive'
                    : remaining <= 20
                      ? 'text-sm text-amber-500'
                      : 'text-sm text-muted-foreground'
                }
              >
                {remaining}
              </span>
            )}

            <Button
              size="sm"
              className="rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white shadow-[0_12px_30px_rgba(91,108,255,0.28)]"
              disabled={isEmpty || isOver || submitting}
              onClick={handleSubmit}
            >
              {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : submitLabel}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

function ActionIcon({ icon: Icon, label }: { icon: React.ElementType; label: string }) {
  return (
    <button
      type="button"
      aria-label={label}
      className="rounded-full p-2 transition-colors hover:bg-primary/10"
    >
      <Icon className="h-5 w-5" />
    </button>
  )
}
