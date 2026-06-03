'use client'

import { useRef, useState } from 'react'
import { Image as ImageIcon, Smile, BarChart2 } from 'lucide-react'

import { cn } from '@/lib/utils'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
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
  submitLabel = 'Poster',
  onPosted,
}: PostComposerProps) {
  const [content, setContent] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const remaining = MAX_CHARS - content.length
  const isEmpty = content.trim().length === 0
  const isOver = remaining < 0

  function handleSubmit() {
    if (isEmpty || isOver) return
    // TODO (issue post) : brancher l'envoi vers POST /posts via l'API Gateway
    onPosted?.(content)
    setContent('')
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
        {/* TODO (issue auth) : avatar de l'utilisateur courant */}
        <AvatarFallback className="bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white">
          U
        </AvatarFallback>
      </Avatar>

      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <textarea
          ref={textareaRef}
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Quoi de neuf ?"
          rows={3}
          autoFocus={autoFocus}
          className="w-full resize-none bg-transparent text-xl text-slate-950 placeholder:text-slate-500 focus:outline-none"
        />

        <Separator className="bg-white/45" />

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
              disabled={isEmpty || isOver}
              onClick={handleSubmit}
            >
              {submitLabel}
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
