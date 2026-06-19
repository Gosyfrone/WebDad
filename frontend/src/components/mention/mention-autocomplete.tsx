'use client'

import { cn, initialOf } from '@/lib/utils'
import type { MentionController } from '@/lib/use-mention'
import { ActivityPresenceDot } from '@/components/profil/activity-presence-dot'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'

interface MentionAutocompleteProps {
  controller: MentionController
  /** Placement vertical par rapport au champ. */
  placement?: 'top' | 'bottom'
  className?: string
}

/**
 * Pop-up de suggestions de mentions : avatar + nom affiché + @username (gris).
 * Positionnée en absolu par rapport au conteneur `relative` parent. Le composant
 * parent fournit le `controller` issu de `useMention` ; le rendu (et le
 * placement) vit ici pour rester homogène entre posts, commentaires et messages.
 *
 * On utilise `onMouseDown` (et non `onClick`) pour insérer AVANT que le champ ne
 * perde le focus (sinon le blur referme la pop-up avant la sélection).
 */
export function MentionAutocomplete({
  controller,
  placement = 'bottom',
  className,
}: MentionAutocompleteProps) {
  const { open, candidates, activeIndex, select, setActiveIndex } = controller
  if (!open || candidates.length === 0) return null

  return (
    <div
      role="listbox"
      className={cn(
        'glass absolute z-50 max-h-64 w-72 max-w-[90vw] overflow-y-auto rounded-2xl border p-1 shadow-xl backdrop-blur',
        placement === 'bottom' ? 'top-full mt-1' : 'bottom-full mb-1',
        className,
      )}
    >
      {candidates.map((c, i) => (
        <button
          key={c.id}
          type="button"
          role="option"
          aria-selected={i === activeIndex}
          onMouseEnter={() => setActiveIndex(i)}
          onMouseDown={(e) => {
            e.preventDefault()
            select(c)
          }}
          className={cn(
            'flex w-full items-center gap-2.5 rounded-xl px-2.5 py-1.5 text-left transition-colors',
            i === activeIndex ? 'bg-accent' : 'hover:bg-accent',
          )}
        >
          <Avatar className="h-8 w-8 shrink-0">
            {c.avatarUrl && <AvatarImage src={c.avatarUrl} alt="" />}
            <AvatarFallback className="bg-gradient-to-br from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] text-xs font-bold text-white">
              {initialOf(c.displayName, c.username)}
            </AvatarFallback>
            <ActivityPresenceDot userId={c.id} className="h-2.5 w-2.5" />
          </Avatar>
          <div className="flex min-w-0 flex-col leading-tight">
            <span className="truncate text-sm font-semibold text-foreground">{c.displayName}</span>
            <span className="truncate text-xs text-muted-foreground">@{c.username}</span>
          </div>
        </button>
      ))}
    </div>
  )
}
