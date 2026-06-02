'use client'

import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

/**
 * Sélection d'emojis courants. Liste statique côté client (pas de dépendance
 * lourde) : suffisant pour la composition de posts. Étoffer si besoin.
 */
const EMOJIS = [
  '😀', '😂', '😍', '🥰', '😎', '🤔', '😅', '🙃',
  '😭', '😤', '😡', '🥳', '😴', '🤯', '🫡', '🤝',
  '👍', '👎', '👏', '🙌', '🙏', '💪', '🔥', '✨',
  '🎉', '💯', '❤️', '💔', '👀', '🚀', '⭐', '☕',
]

interface EmojiPickerProps {
  /** Appelé avec l'emoji choisi. */
  onSelect: (emoji: string) => void
  /** Déclencheur (rendu via `asChild`). */
  children: React.ReactNode
}

export function EmojiPicker({ onSelect, children }: EmojiPickerProps) {
  return (
    <Popover>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-2">
        <div className="grid grid-cols-8 gap-0.5">
          {EMOJIS.map((emoji) => (
            <button
              key={emoji}
              type="button"
              aria-label={`Emoji ${emoji}`}
              onClick={() => onSelect(emoji)}
              className="rounded p-1 text-lg leading-none transition-colors hover:bg-accent"
            >
              {emoji}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
