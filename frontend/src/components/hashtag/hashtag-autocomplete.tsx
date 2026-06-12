'use client'

import { Hash } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { HashtagController } from '@/lib/use-hashtag'
import { useT } from '@/components/language-provider'

interface HashtagAutocompleteProps {
  controller: HashtagController
  placement?: 'top' | 'bottom'
  className?: string
}

export function HashtagAutocomplete({
  controller,
  placement = 'bottom',
  className,
}: HashtagAutocompleteProps) {
  const t = useT()
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
      {candidates.map((candidate, index) => (
        <button
          key={candidate.tag}
          type="button"
          role="option"
          aria-selected={index === activeIndex}
          onMouseEnter={() => setActiveIndex(index)}
          onMouseDown={(event) => {
            event.preventDefault()
            select(candidate)
          }}
          className={cn(
            'flex w-full items-center gap-2.5 rounded-xl px-2.5 py-1.5 text-left transition-colors',
            index === activeIndex ? 'bg-accent' : 'hover:bg-accent',
          )}
        >
          <span className="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-[#5B6CFF]/10 text-[#5B6CFF] dark:bg-[#9aa6ff]/15 dark:text-[#9aa6ff]">
            <Hash className="h-4 w-4" aria-hidden />
          </span>
          <span className="flex min-w-0 flex-col leading-tight">
            <span className="truncate text-sm font-semibold text-foreground">#{candidate.tag}</span>
            <span className="truncate text-xs text-muted-foreground">
              {typeof candidate.count === 'number'
                ? t(candidate.count > 1 ? 'trends.posts_other' : 'trends.posts_one', {
                    count: String(candidate.count),
                  })
                : t('search_suggestions.trend')}
            </span>
          </span>
        </button>
      ))}
    </div>
  )
}
